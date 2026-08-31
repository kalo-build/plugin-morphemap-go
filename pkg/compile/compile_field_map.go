package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kalo-build/morphe-go/pkg/registry"
	"github.com/kalo-build/plugin-morphemap-go/pkg/mapdef"
)

// compileFieldMap generates a Go converter function for a field map.
func compileFieldMap(m *mapdef.MorpheMap, localReg, externalReg *registry.Registry, enumMapIndex map[enumMapKey]*mapdef.MorpheMap, config GoConverterConfig) error {
	data := BuildFieldMapTemplateData(m, localReg, externalReg, config)

	// Generate the converter function
	content, err := RenderFieldMapTemplate(data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Write the file
	filename := toSnakeCase(m.Name) + ".go"
	filePath := filepath.Join(config.OutputPath, filename)

	if err := os.MkdirAll(config.OutputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

// FieldMapTemplateData holds the data for the field map Go template.
type FieldMapTemplateData struct {
	PackageName       string
	ConverterName     string
	SourceType        string
	TargetType        string
	SourcePackagePath string
	TargetPackagePath string
	SourcePkgAlias    string
	TargetPkgAlias    string
	Fields            []FieldMappingData
	HasConditionals   bool
	HasCasts          bool
	HasErrors         bool
	ErrorHandling     ErrorHandling
	Hooks             []string
}

// FieldMappingData holds a single field mapping entry for template rendering.
type FieldMappingData struct {
	TargetField    string
	SourceExpr     string
	IsConstant     bool
	ConstValue     string
	ConstType      string
	Cast           string
	Required       bool
	ErrorCode      string
	Condition      string
	HasValueMap    bool
	ValueMap       map[string]string
	TargetOptional bool
	SourceOptional bool
}

// BuildFieldMapTemplateData constructs template data from a MorpheMap field map.
func BuildFieldMapTemplateData(m *mapdef.MorpheMap, localReg, externalReg *registry.Registry, config GoConverterConfig) FieldMapTemplateData {
	// Determine source/target aliases (heuristic: look at field mapping targets)
	var sourceAlias, targetAlias string
	for _, alias := range sortedKeys(m.Aliases) {
		if targetAlias == "" {
			for key := range m.Fields {
				if strings.HasPrefix(key, alias+".") {
					targetAlias = alias
					break
				}
			}
		}
	}
	for _, alias := range sortedKeys(m.Aliases) {
		if alias != targetAlias {
			sourceAlias = alias
			break
		}
	}

	sourceTypeName := m.Aliases[sourceAlias]
	targetTypeName := m.Aliases[targetAlias]

	data := FieldMapTemplateData{
		PackageName:       lastPathSegment(config.PackagePath),
		ConverterName:     m.Name,
		SourceType:        lastPathSegment(sourceTypeName),
		TargetType:        lastPathSegment(targetTypeName),
		SourcePackagePath: config.SourcePackagePath,
		TargetPackagePath: config.TargetPackagePath,
		SourcePkgAlias:    lastPathSegment(config.SourcePackagePath),
		TargetPkgAlias:    lastPathSegment(config.TargetPackagePath),
		ErrorHandling:     config.ErrorHandling,
	}

	for key, val := range m.Fields {
		targetField := stripAliasPrefix(key, targetAlias)

		fd := FieldMappingData{
			TargetField:    targetField,
			TargetOptional: isFieldOptional(targetTypeName, targetField, localReg, externalReg),
		}

		if val.IsScalar {
			switch v := val.Scalar.(type) {
			case string:
				if strings.Contains(v, ".") && containsAlias(v, m.Aliases) {
					fd.SourceExpr = stripAliasPrefix(v, sourceAlias)
					sourceField := stripAliasPrefix(v, sourceAlias)
					fd.SourceOptional = isFieldOptional(sourceTypeName, sourceField, localReg, externalReg)
				} else {
					fd.IsConstant = true
					fd.ConstValue = fmt.Sprintf("%q", v)
					fd.ConstType = "string"
				}
			case bool:
				fd.IsConstant = true
				fd.ConstValue = fmt.Sprintf("%t", v)
				fd.ConstType = "bool"
			case int:
				fd.IsConstant = true
				fd.ConstValue = fmt.Sprintf("%d", v)
				fd.ConstType = "int"
			case float64:
				fd.IsConstant = true
				fd.ConstValue = fmt.Sprintf("%f", v)
				fd.ConstType = "float64"
			}
		} else if val.Object != nil {
			obj := val.Object
			fd.SourceExpr = stripAliasPrefix(obj.From, sourceAlias)
			sourceField := stripAliasPrefix(obj.From, sourceAlias)
			fd.SourceOptional = isFieldOptional(sourceTypeName, sourceField, localReg, externalReg)
			fd.Cast = obj.Cast
			fd.Required = obj.Required
			fd.ErrorCode = obj.ErrorCode

			if obj.Cast != "" {
				data.HasCasts = true
			}
			if obj.Required && fd.SourceOptional {
				data.HasErrors = true
			}
			if len(obj.When) > 0 {
				data.HasConditionals = true
				var conditions []string
				for condKey, condVal := range obj.When {
					condField := stripAliasPrefix(condKey, targetAlias)
					conditions = append(conditions, fmt.Sprintf("target.%s == %v", condField, condVal))
				}
				fd.Condition = strings.Join(conditions, " && ")
			}
			if len(obj.ValueMap) > 0 {
				fd.HasValueMap = true
				fd.ValueMap = obj.ValueMap
			}
		}

		data.Fields = append(data.Fields, fd)
	}

	if m.Hooks != nil {
		for _, hook := range m.Hooks.AfterMap {
			data.Hooks = append(data.Hooks, hook.Name)
		}
	}

	return data
}

// isFieldOptional checks whether a field on a given Morphe type has the "optional"
// attribute by looking it up across models, structures, and entities in both registries.
func isFieldOptional(typeName, fieldName string, localReg, externalReg *registry.Registry) bool {
	for _, reg := range []*registry.Registry{localReg, externalReg} {
		if reg == nil {
			continue
		}
		if model, ok := reg.GetAllModels()[typeName]; ok {
			if f, ok := model.Fields[fieldName]; ok {
				return hasAttribute(f.Attributes, "optional")
			}
		}
		if structure, ok := reg.GetAllStructures()[typeName]; ok {
			if f, ok := structure.Fields[fieldName]; ok {
				return hasAttribute(f.Attributes, "optional")
			}
		}
	}
	return false
}

func hasAttribute(attrs []string, target string) bool {
	for _, a := range attrs {
		if a == target {
			return true
		}
	}
	return false
}

// RenderFieldMapTemplate renders a Go converter file from template data.
func RenderFieldMapTemplate(data FieldMapTemplateData) (string, error) {
	tmpl, err := template.New("converter").Parse(fieldMapGoTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

var fieldMapGoTemplate = `// Code generated by plugin-morphemap-go. DO NOT EDIT.
package {{.PackageName}}

import (
	{{.SourcePkgAlias}} "{{.SourcePackagePath}}"
	{{.TargetPkgAlias}} "{{.TargetPackagePath}}"
{{- if .HasErrors}}
	"fmt"
{{- end}}
)

// {{.ConverterName}} converts a {{.SourceType}} to a {{.TargetType}}.
func {{.ConverterName}}(source {{.SourcePkgAlias}}.{{.SourceType}}) ({{.TargetPkgAlias}}.{{.TargetType}}, error) {
	var target {{.TargetPkgAlias}}.{{.TargetType}}

{{- range .Fields}}
{{- if .IsConstant}}
{{- if .TargetOptional}}
	{
		v := {{.ConstValue}}
		target.{{.TargetField}} = &v
	}
{{- else}}
	target.{{.TargetField}} = {{.ConstValue}}
{{- end}}
{{- else if .Condition}}
	if {{.Condition}} {
{{- if and .TargetOptional (not .SourceOptional)}}
		v := source.{{.SourceExpr}}
		target.{{.TargetField}} = &v
{{- else if and (not .TargetOptional) .SourceOptional}}
		if source.{{.SourceExpr}} != nil {
			target.{{.TargetField}} = *source.{{.SourceExpr}}
		}
{{- else}}
		target.{{.TargetField}} = source.{{.SourceExpr}}
{{- end}}
	}
{{- else if .Cast}}
{{- if and .TargetOptional (not .SourceOptional)}}
	{
		v := {{.Cast}}(source.{{.SourceExpr}})
		target.{{.TargetField}} = &v
	}
{{- else if and (not .TargetOptional) .SourceOptional}}
	if source.{{.SourceExpr}} != nil {
		target.{{.TargetField}} = {{.Cast}}(*source.{{.SourceExpr}})
	}
{{- else if and .TargetOptional .SourceOptional}}
	if source.{{.SourceExpr}} != nil {
		v := {{.Cast}}(*source.{{.SourceExpr}})
		target.{{.TargetField}} = &v
	}
{{- else}}
	target.{{.TargetField}} = {{.Cast}}(source.{{.SourceExpr}})
{{- end}}
{{- else if .HasValueMap}}
{{- if .SourceOptional}}
	if source.{{.SourceExpr}} != nil {
		switch *source.{{.SourceExpr}} {
{{- range $src, $tgt := .ValueMap}}
		case "{{$src}}":
{{- if $.TargetOptional}}
			v := "{{$tgt}}"
			target.{{$.TargetField}} = &v
{{- else}}
			target.{{$.TargetField}} = "{{$tgt}}"
{{- end}}
{{- end}}
		}
	}
{{- else}}
	switch source.{{.SourceExpr}} {
{{- range $src, $tgt := .ValueMap}}
	case "{{$src}}":
{{- if $.TargetOptional}}
		v := "{{$tgt}}"
		target.{{$.TargetField}} = &v
{{- else}}
		target.{{$.TargetField}} = "{{$tgt}}"
{{- end}}
{{- end}}
	}
{{- end}}
{{- else if .SourceExpr}}
{{- if and .Required .SourceOptional}}
	if source.{{.SourceExpr}} == nil {
		return target, fmt.Errorf("required field {{.SourceExpr}} is nil")
	}
{{- if .TargetOptional}}
	target.{{.TargetField}} = source.{{.SourceExpr}}
{{- else}}
	target.{{.TargetField}} = *source.{{.SourceExpr}}
{{- end}}
{{- else if and .TargetOptional (not .SourceOptional)}}
	{
		v := source.{{.SourceExpr}}
		target.{{.TargetField}} = &v
	}
{{- else if and (not .TargetOptional) .SourceOptional}}
	if source.{{.SourceExpr}} != nil {
		target.{{.TargetField}} = *source.{{.SourceExpr}}
	}
{{- else}}
	target.{{.TargetField}} = source.{{.SourceExpr}}
{{- end}}
{{- end}}
{{- end}}

	return target, nil
}
{{- range .Hooks}}

// {{.}} is a hook point for custom post-mapping logic.
// Implement this function to handle cases that can't be expressed declaratively.
// func {{.}}(source *{{$.SourcePkgAlias}}.{{$.SourceType}}, target *{{$.TargetPkgAlias}}.{{$.TargetType}}) error {
//     // TODO: implement
//     return nil
// }
{{- end}}
`

// Helper functions

func stripAliasPrefix(path string, alias string) string {
	prefix := alias + "."
	if strings.HasPrefix(path, prefix) {
		return path[len(prefix):]
	}
	return path
}

func containsAlias(value string, aliases map[string]string) bool {
	for alias := range aliases {
		if strings.HasPrefix(value, alias+".") {
			return true
		}
	}
	return false
}

func toSnakeCase(name string) string {
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := rune(name[i-1])
			if prev >= 'a' && prev <= 'z' {
				result.WriteRune('_')
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func lastPathSegment(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple sort
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
