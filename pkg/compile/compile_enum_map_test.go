package compile_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/kalo-build/plugin-morphemap-go/pkg/compile"
	"github.com/kalo-build/plugin-morphemap-go/pkg/mapdef"
)

type EnumMapTestSuite struct {
	suite.Suite
}

func TestEnumMapTestSuite(t *testing.T) {
	suite.Run(t, new(EnumMapTestSuite))
}

func (suite *EnumMapTestSuite) TestBuildEnumMapTemplateData_BasicEntries() {
	m := &mapdef.MorpheMap{
		Name: "StatusToPriority",
		Aliases: map[string]string{
			"Src": "Status",
			"Tgt": "Priority",
		},
		Entries: map[string]string{
			"Tgt.Low":    "Src.Inactive",
			"Tgt.Medium": "Src.Pending",
			"Tgt.High":   "Src.Active",
		},
	}
	config := compile.GoConverterConfig{
		PackagePath:       "github.com/example/conv",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
	}

	data := compile.BuildEnumMapTemplateData(m, config)

	suite.Equal("conv", data.PackageName)
	suite.Equal("StatusToPriority", data.FunctionName)
	suite.Equal("Status", data.SourceType)
	suite.Equal("Priority", data.TargetType)
	suite.Len(data.Entries, 3)
}

func (suite *EnumMapTestSuite) TestBuildEnumMapTemplateData_StripsAliasPrefix() {
	m := &mapdef.MorpheMap{
		Name: "SimpleEnum",
		Aliases: map[string]string{
			"A": "EnumA",
			"B": "EnumB",
		},
		Entries: map[string]string{
			"B.ValueX": "A.ValueY",
		},
	}
	config := compile.GoConverterConfig{
		PackagePath:       "github.com/example/conv",
		SourcePackagePath: "github.com/example/source",
		TargetPackagePath: "github.com/example/target",
	}

	data := compile.BuildEnumMapTemplateData(m, config)

	suite.Require().Len(data.Entries, 1)
	suite.Equal("ValueX", data.Entries[0].TargetEntry)
	suite.Equal("ValueY", data.Entries[0].SourceEntry)
}

func (suite *EnumMapTestSuite) TestRenderEnumMapTemplate_ProducesValidGo() {
	data := compile.EnumMapTemplateData{
		PackageName:  "conv",
		FunctionName: "StatusToPriority",
		SourceType:   "Status",
		TargetType:   "Priority",
		Entries: []compile.EnumEntryData{
			{SourceEntry: "Active", TargetEntry: "High"},
			{SourceEntry: "Inactive", TargetEntry: "Low"},
		},
	}

	result, renderErr := compile.RenderEnumMapTemplate(data)

	suite.NoError(renderErr)
	suite.Contains(result, "package conv")
	suite.Contains(result, "func StatusToPriority(source string)")
	suite.Contains(result, `case "Active":`)
	suite.Contains(result, `return "High", nil`)
	suite.Contains(result, `case "Inactive":`)
	suite.Contains(result, `return "Low", nil`)
	suite.Contains(result, "default:")
}

func (suite *EnumMapTestSuite) TestRenderEnumMapTemplate_EmptyEntries() {
	data := compile.EnumMapTemplateData{
		PackageName:  "conv",
		FunctionName: "EmptyEnum",
		SourceType:   "TypeA",
		TargetType:   "TypeB",
		Entries:      []compile.EnumEntryData{},
	}

	result, renderErr := compile.RenderEnumMapTemplate(data)

	suite.NoError(renderErr)
	suite.Contains(result, "func EmptyEnum(source string)")
	suite.Contains(result, "default:")
}
