package mapdef

// MorpheMap represents a complete MorpheMap document (KA:MM1:YAML1)
type MorpheMap struct {
	Name      string            `yaml:"name"`
	Aliases   map[string]string `yaml:"aliases"`
	Extends string        `yaml:"extends,omitempty"`
	Fields  FieldMappings `yaml:"fields,omitempty"`
	Entries   map[string]string `yaml:"entries,omitempty"`
	Overrides FieldMappings     `yaml:"overrides,omitempty"`
	Hooks     *Hooks            `yaml:"hooks,omitempty"`
}

// FieldMappings holds the field mapping rules.
// In YAML, each value can be either a scalar (string/number/bool) or an object.
// We use a custom type to handle this polymorphism.
type FieldMappings map[string]FieldMappingValue

// FieldMappingValue represents either a scalar shorthand or a full mapping object.
type FieldMappingValue struct {
	// IsScalar indicates if the value is a scalar (access path or literal)
	IsScalar bool

	// Scalar holds the raw scalar value (string, number, bool) if IsScalar is true
	Scalar interface{}

	// Object holds the full mapping object if IsScalar is false
	Object *FieldMapping
}

// FieldMapping represents the full object form of a field mapping.
type FieldMapping struct {
	From     string            `yaml:"from"`
	Cast     string            `yaml:"cast,omitempty"`
	Required bool              `yaml:"required,omitempty"`
	ErrorCode string           `yaml:"errorCode,omitempty"`
	When     map[string]interface{} `yaml:"when,omitempty"`
	ValueMap map[string]string `yaml:"valueMap,omitempty"`
}

// Hooks defines escape hatch hook points for generated code.
type Hooks struct {
	AfterMap []Hook `yaml:"afterMap,omitempty"`
}

// Hook defines a named hook point.
type Hook struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// MapType indicates whether the map operates on fields or entries.
type MapType string

const (
	MapTypeField MapType = "field"
	MapTypeEnum  MapType = "enum"
)

// InferMapType determines the map type from the document structure.
func (m *MorpheMap) InferMapType() MapType {
	if len(m.Entries) > 0 {
		return MapTypeEnum
	}
	return MapTypeField
}
