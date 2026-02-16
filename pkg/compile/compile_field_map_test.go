package compile_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/kalo-build/plugin-morphemap-go/pkg/compile"
	"github.com/kalo-build/plugin-morphemap-go/pkg/mapdef"
)

type FieldMapTestSuite struct {
	suite.Suite
}

func TestFieldMapTestSuite(t *testing.T) {
	suite.Run(t, new(FieldMapTestSuite))
}

func (suite *FieldMapTestSuite) TestBuildFieldMapTemplateData_SimpleScalarMapping() {
	m := &mapdef.MorpheMap{
		Name: "OrgToProject",
		Aliases: map[string]string{
			"Org":  "Organization",
			"Proj": "Project",
		},
		Fields: mapdef.FieldMappings{
			"Proj.Name": mapdef.FieldMappingValue{IsScalar: true, Scalar: "Org.Name"},
			"Proj.Code": mapdef.FieldMappingValue{IsScalar: true, Scalar: "Org.Code"},
		},
	}
	config := compile.GoConverterConfig{
		PackagePath:       "github.com/example/converters",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
		ErrorHandling:     compile.ErrorHandlingReturn,
	}

	data := compile.BuildFieldMapTemplateData(m, config)

	suite.Equal("converters", data.PackageName)
	suite.Equal("OrgToProject", data.ConverterName)
	suite.Equal("source", data.SourcePkgAlias)
	suite.Equal("target", data.TargetPkgAlias)
	suite.Len(data.Fields, 2)
}

func (suite *FieldMapTestSuite) TestBuildFieldMapTemplateData_ConstantValue() {
	m := &mapdef.MorpheMap{
		Name: "WithConstant",
		Aliases: map[string]string{
			"Src": "Source",
			"Tgt": "Target",
		},
		Fields: mapdef.FieldMappings{
			"Tgt.Status": mapdef.FieldMappingValue{IsScalar: true, Scalar: "active"},
		},
	}
	config := compile.GoConverterConfig{
		PackagePath:       "github.com/example/conv",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
	}

	data := compile.BuildFieldMapTemplateData(m, config)

	suite.Require().Len(data.Fields, 1)
	suite.True(data.Fields[0].IsConstant)
	suite.Equal(`"active"`, data.Fields[0].ConstValue)
	suite.Equal("string", data.Fields[0].ConstType)
}

func (suite *FieldMapTestSuite) TestBuildFieldMapTemplateData_ObjectWithCast() {
	m := &mapdef.MorpheMap{
		Name: "WithCast",
		Aliases: map[string]string{
			"Src": "Source",
			"Tgt": "Target",
		},
		Fields: mapdef.FieldMappings{
			"Tgt.Count": mapdef.FieldMappingValue{
				IsScalar: false,
				Object: &mapdef.FieldMapping{
					From: "Src.Count",
					Cast: "int",
				},
			},
		},
	}
	config := compile.GoConverterConfig{
		PackagePath:       "github.com/example/conv",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
	}

	data := compile.BuildFieldMapTemplateData(m, config)

	suite.Require().Len(data.Fields, 1)
	suite.Equal("int", data.Fields[0].Cast)
	suite.True(data.HasCasts)
}

func (suite *FieldMapTestSuite) TestBuildFieldMapTemplateData_ObjectWithRequired() {
	m := &mapdef.MorpheMap{
		Name: "WithRequired",
		Aliases: map[string]string{
			"Src": "Source",
			"Tgt": "Target",
		},
		Fields: mapdef.FieldMappings{
			"Tgt.ID": mapdef.FieldMappingValue{
				IsScalar: false,
				Object: &mapdef.FieldMapping{
					From:      "Src.ID",
					Required:  true,
					ErrorCode: "ID_REQUIRED",
				},
			},
		},
	}
	config := compile.GoConverterConfig{
		PackagePath:       "github.com/example/conv",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
	}

	data := compile.BuildFieldMapTemplateData(m, config)

	suite.Require().Len(data.Fields, 1)
	suite.True(data.Fields[0].Required)
	suite.Equal("ID_REQUIRED", data.Fields[0].ErrorCode)
	suite.True(data.HasErrors)
}

func (suite *FieldMapTestSuite) TestBuildFieldMapTemplateData_ObjectWithValueMap() {
	m := &mapdef.MorpheMap{
		Name: "WithValueMap",
		Aliases: map[string]string{
			"Src": "Source",
			"Tgt": "Target",
		},
		Fields: mapdef.FieldMappings{
			"Tgt.Status": mapdef.FieldMappingValue{
				IsScalar: false,
				Object: &mapdef.FieldMapping{
					From: "Src.Status",
					ValueMap: map[string]string{
						"active":   "Active",
						"inactive": "Inactive",
					},
				},
			},
		},
	}
	config := compile.GoConverterConfig{
		PackagePath:       "github.com/example/conv",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
	}

	data := compile.BuildFieldMapTemplateData(m, config)

	suite.Require().Len(data.Fields, 1)
	suite.True(data.Fields[0].HasValueMap)
	suite.Len(data.Fields[0].ValueMap, 2)
}

func (suite *FieldMapTestSuite) TestBuildFieldMapTemplateData_WithHooks() {
	m := &mapdef.MorpheMap{
		Name: "WithHooks",
		Aliases: map[string]string{
			"Src": "Source",
			"Tgt": "Target",
		},
		Fields: mapdef.FieldMappings{
			"Tgt.Name": mapdef.FieldMappingValue{IsScalar: true, Scalar: "Src.Name"},
		},
		Hooks: &mapdef.Hooks{
			AfterMap: []mapdef.Hook{
				{Name: "ApplyDefaults", Description: "Set default values"},
			},
		},
	}
	config := compile.GoConverterConfig{
		PackagePath:       "github.com/example/conv",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
	}

	data := compile.BuildFieldMapTemplateData(m, config)

	suite.Require().Len(data.Hooks, 1)
	suite.Equal("ApplyDefaults", data.Hooks[0])
}

func (suite *FieldMapTestSuite) TestRenderFieldMapTemplate_ProducesValidGo() {
	data := compile.FieldMapTemplateData{
		PackageName:       "converters",
		ConverterName:     "OrgToProject",
		SourceType:        "Organization",
		TargetType:        "Project",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
		SourcePkgAlias:    "source",
		TargetPkgAlias:    "target",
		ErrorHandling:     compile.ErrorHandlingReturn,
		Fields: []compile.FieldMappingData{
			{TargetField: "Name", SourceExpr: "Name"},
			{TargetField: "Code", SourceExpr: "Code"},
		},
	}

	result, renderErr := compile.RenderFieldMapTemplate(data)

	suite.NoError(renderErr)
	suite.Contains(result, "package converters")
	suite.Contains(result, "func OrgToProject(source source.Organization)")
	suite.Contains(result, "target.Name = source.Name")
	suite.Contains(result, "target.Code = source.Code")
	suite.Contains(result, "return target, nil")
}

func (suite *FieldMapTestSuite) TestRenderFieldMapTemplate_ConstantAssignment() {
	data := compile.FieldMapTemplateData{
		PackageName:       "conv",
		ConverterName:     "TestConv",
		SourceType:        "Src",
		TargetType:        "Tgt",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
		SourcePkgAlias:    "source",
		TargetPkgAlias:    "target",
		Fields: []compile.FieldMappingData{
			{TargetField: "Status", IsConstant: true, ConstValue: `"active"`, ConstType: "string"},
		},
	}

	result, renderErr := compile.RenderFieldMapTemplate(data)

	suite.NoError(renderErr)
	suite.Contains(result, `target.Status = "active"`)
}

func (suite *FieldMapTestSuite) TestRenderFieldMapTemplate_CastAssignment() {
	data := compile.FieldMapTemplateData{
		PackageName:       "conv",
		ConverterName:     "TestConv",
		SourceType:        "Src",
		TargetType:        "Tgt",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
		SourcePkgAlias:    "source",
		TargetPkgAlias:    "target",
		HasCasts:          true,
		Fields: []compile.FieldMappingData{
			{TargetField: "Count", SourceExpr: "Count", Cast: "int"},
		},
	}

	result, renderErr := compile.RenderFieldMapTemplate(data)

	suite.NoError(renderErr)
	suite.Contains(result, "target.Count = int(source.Count)")
}
