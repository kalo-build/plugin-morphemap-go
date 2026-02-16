package mapdef

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML implements custom YAML unmarshaling for FieldMappingValue.
// This handles the scalar-or-object polymorphism: a value can be either
// a simple scalar (string/number/bool) or a full FieldMapping object.
func (v *FieldMappingValue) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		v.IsScalar = true
		// Try to parse as different scalar types
		var s string
		if err := node.Decode(&s); err == nil {
			v.Scalar = s
			return nil
		}
		var b bool
		if err := node.Decode(&b); err == nil {
			v.Scalar = b
			return nil
		}
		var i int
		if err := node.Decode(&i); err == nil {
			v.Scalar = i
			return nil
		}
		var f float64
		if err := node.Decode(&f); err == nil {
			v.Scalar = f
			return nil
		}
		return fmt.Errorf("unsupported scalar type at line %d", node.Line)

	case yaml.MappingNode:
		v.IsScalar = false
		var fm FieldMapping
		if err := node.Decode(&fm); err != nil {
			return fmt.Errorf("failed to decode field mapping at line %d: %w", node.Line, err)
		}
		v.Object = &fm
		return nil

	default:
		return fmt.Errorf("unexpected YAML node kind %d at line %d", node.Kind, node.Line)
	}
}

// MarshalYAML implements custom YAML marshaling for FieldMappingValue.
func (v FieldMappingValue) MarshalYAML() (interface{}, error) {
	if v.IsScalar {
		return v.Scalar, nil
	}
	return v.Object, nil
}

// UnmarshalYAML implements custom YAML unmarshaling for FieldMappings.
// The special case is when "fields" is the literal string "auto" for casing maps.
func (fm *FieldMappings) UnmarshalYAML(node *yaml.Node) error {
	// Handle "fields: auto" special case
	if node.Kind == yaml.ScalarNode {
		var s string
		if err := node.Decode(&s); err != nil {
			return err
		}
		if s == "auto" {
			// Return empty map; the "auto" semantics are handled by the plugin
			*fm = make(FieldMappings)
			return nil
		}
		return fmt.Errorf("unexpected scalar value for fields: %q (only 'auto' is valid)", s)
	}

	// Normal case: map of field mappings
	if node.Kind == yaml.MappingNode {
		result := make(FieldMappings)
		for i := 0; i < len(node.Content)-1; i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			var key string
			if err := keyNode.Decode(&key); err != nil {
				return fmt.Errorf("failed to decode field mapping key at line %d: %w", keyNode.Line, err)
			}

			var value FieldMappingValue
			if err := value.UnmarshalYAML(valueNode); err != nil {
				return err
			}

			result[key] = value
		}
		*fm = result
		return nil
	}

	return fmt.Errorf("unexpected YAML node kind %d for fields at line %d", node.Kind, node.Line)
}
