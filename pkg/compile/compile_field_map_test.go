package compile_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/kalo-build/morphe-go/pkg/registry"
	"github.com/kalo-build/morphe-go/pkg/yaml"
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

	data := compile.BuildFieldMapTemplateData(m, nil, nil, config)

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

	data := compile.BuildFieldMapTemplateData(m, nil, nil, config)

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

	data := compile.BuildFieldMapTemplateData(m, nil, nil, config)

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

	data := compile.BuildFieldMapTemplateData(m, nil, nil, config)

	suite.Require().Len(data.Fields, 1)
	suite.True(data.Fields[0].Required)
	suite.Equal("ID_REQUIRED", data.Fields[0].ErrorCode)
	suite.False(data.HasErrors, "HasErrors should be false when source field is not optional")
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

	data := compile.BuildFieldMapTemplateData(m, nil, nil, config)

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

	data := compile.BuildFieldMapTemplateData(m, nil, nil, config)

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

func (suite *FieldMapTestSuite) TestBuildFieldMapTemplateData_TargetOptionalSourceNot() {
	reg := &registry.Registry{}
	reg.SetModel("Source", yaml.Model{
		Fields: map[string]yaml.ModelField{
			"Email": {Type: "String"},
		},
	})
	reg.SetModel("Target", yaml.Model{
		Fields: map[string]yaml.ModelField{
			"Email": {Type: "String", Attributes: []string{"optional"}},
		},
	})

	m := &mapdef.MorpheMap{
		Name: "SrcToTgt",
		Aliases: map[string]string{
			"Src": "Source",
			"Tgt": "Target",
		},
		Fields: mapdef.FieldMappings{
			"Tgt.Email": mapdef.FieldMappingValue{IsScalar: true, Scalar: "Src.Email"},
		},
	}
	config := compile.GoConverterConfig{
		PackagePath:       "github.com/example/conv",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
	}

	data := compile.BuildFieldMapTemplateData(m, reg, nil, config)

	suite.Require().Len(data.Fields, 1)
	suite.True(data.Fields[0].TargetOptional)
	suite.False(data.Fields[0].SourceOptional)
}

func (suite *FieldMapTestSuite) TestBuildFieldMapTemplateData_SourceOptionalTargetNot() {
	reg := &registry.Registry{}
	reg.SetModel("Source", yaml.Model{
		Fields: map[string]yaml.ModelField{
			"Email": {Type: "String", Attributes: []string{"optional"}},
		},
	})
	reg.SetModel("Target", yaml.Model{
		Fields: map[string]yaml.ModelField{
			"Email": {Type: "String"},
		},
	})

	m := &mapdef.MorpheMap{
		Name: "SrcToTgt",
		Aliases: map[string]string{
			"Src": "Source",
			"Tgt": "Target",
		},
		Fields: mapdef.FieldMappings{
			"Tgt.Email": mapdef.FieldMappingValue{IsScalar: true, Scalar: "Src.Email"},
		},
	}
	config := compile.GoConverterConfig{
		PackagePath:       "github.com/example/conv",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
	}

	data := compile.BuildFieldMapTemplateData(m, reg, nil, config)

	suite.Require().Len(data.Fields, 1)
	suite.False(data.Fields[0].TargetOptional)
	suite.True(data.Fields[0].SourceOptional)
}

func (suite *FieldMapTestSuite) TestRenderFieldMapTemplate_TargetOptionalSourceNot() {
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
			{TargetField: "Email", SourceExpr: "Email", TargetOptional: true, SourceOptional: false},
		},
	}

	result, renderErr := compile.RenderFieldMapTemplate(data)

	suite.NoError(renderErr)
	suite.Contains(result, "v := source.Email")
	suite.Contains(result, "target.Email = &v")
	suite.NotContains(result, "target.Email = source.Email")
}

func (suite *FieldMapTestSuite) TestRenderFieldMapTemplate_SourceOptionalTargetNot() {
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
			{TargetField: "Email", SourceExpr: "Email", TargetOptional: false, SourceOptional: true},
		},
	}

	result, renderErr := compile.RenderFieldMapTemplate(data)

	suite.NoError(renderErr)
	suite.Contains(result, "if source.Email != nil")
	suite.Contains(result, "target.Email = *source.Email")
}

func (suite *FieldMapTestSuite) TestRenderFieldMapTemplate_BothOptional() {
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
			{TargetField: "Email", SourceExpr: "Email", TargetOptional: true, SourceOptional: true},
		},
	}

	result, renderErr := compile.RenderFieldMapTemplate(data)

	suite.NoError(renderErr)
	suite.Contains(result, "target.Email = source.Email")
}

func (suite *FieldMapTestSuite) TestRenderFieldMapTemplate_ConstantToOptionalTarget() {
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
			{TargetField: "Status", IsConstant: true, ConstValue: `"active"`, ConstType: "string", TargetOptional: true},
		},
	}

	result, renderErr := compile.RenderFieldMapTemplate(data)

	suite.NoError(renderErr)
	suite.Contains(result, `v := "active"`)
	suite.Contains(result, "target.Status = &v")
}

func (suite *FieldMapTestSuite) TestBuildFieldMapTemplateData_RequiredOptionalSourceSetsHasErrors() {
	reg := &registry.Registry{}
	reg.SetModel("Source", yaml.Model{
		Fields: map[string]yaml.ModelField{
			"ID": {Type: "UUID", Attributes: []string{"optional"}},
		},
	})
	reg.SetModel("Target", yaml.Model{
		Fields: map[string]yaml.ModelField{
			"ID": {Type: "UUID"},
		},
	})

	m := &mapdef.MorpheMap{
		Name: "SrcToTgt",
		Aliases: map[string]string{
			"Src": "Source",
			"Tgt": "Target",
		},
		Fields: mapdef.FieldMappings{
			"Tgt.ID": mapdef.FieldMappingValue{
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

	data := compile.BuildFieldMapTemplateData(m, reg, nil, config)

	suite.Require().Len(data.Fields, 1)
	suite.True(data.Fields[0].Required)
	suite.True(data.Fields[0].SourceOptional)
	suite.True(data.HasErrors)
}

func (suite *FieldMapTestSuite) TestBuildFieldMapTemplateData_ExternalRegistryOptional() {
	localReg := &registry.Registry{}
	localReg.SetModel("Target", yaml.Model{
		Fields: map[string]yaml.ModelField{
			"Email": {Type: "String", Attributes: []string{"optional"}},
		},
	})

	externalReg := &registry.Registry{}
	externalReg.SetStructure("ExternalSource", yaml.Structure{
		Fields: map[string]yaml.StructureField{
			"Email": {Type: "String"},
		},
	})

	m := &mapdef.MorpheMap{
		Name: "ExtToLocal",
		Aliases: map[string]string{
			"Ext":   "ExternalSource",
			"Local": "Target",
		},
		Fields: mapdef.FieldMappings{
			"Local.Email": mapdef.FieldMappingValue{IsScalar: true, Scalar: "Ext.Email"},
		},
	}
	config := compile.GoConverterConfig{
		PackagePath:       "github.com/example/conv",
		SourcePackagePath: "github.com/example/external",
		TargetPackagePath: "github.com/example/models",
	}

	data := compile.BuildFieldMapTemplateData(m, localReg, externalReg, config)

	suite.Require().Len(data.Fields, 1)
	suite.True(data.Fields[0].TargetOptional)
	suite.False(data.Fields[0].SourceOptional)
}
