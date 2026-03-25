package schema

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load reads and parses a schema.json file.
func Load(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading schema %q: %w", path, err)
	}
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing schema %q: %w", path, err)
	}
	if err := validateSchema(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// validateSchema checks schema-level invariants: intrinsic fields are absent,
// status is correctly declared, and status references are consistent.
func validateSchema(s *Schema) error {
	// id and area are MCP-intrinsic and must not be declared in fields.
	for _, name := range []string{"id", "area"} {
		if _, ok := s.Fields[name]; ok {
			return fmt.Errorf("schema error: '%s' must not be declared in fields (it is an MCP-intrinsic field)", name)
		}
	}

	// status must exist, be an enum, have values matching statuses, and have a default.
	statusDef, ok := s.Fields["status"]
	if !ok {
		return fmt.Errorf("schema error: 'status' must be declared in fields")
	}
	if statusDef.Type != FieldTypeEnum {
		return fmt.Errorf("schema error: 'status' must be type 'enum'")
	}
	if statusDef.Default == nil {
		return fmt.Errorf("schema error: 'status' must have a default value")
	}
	statusSet := make(map[string]bool, len(s.Statuses))
	for _, sv := range s.Statuses {
		statusSet[sv] = true
	}
	if len(statusDef.Values) != len(s.Statuses) {
		return fmt.Errorf("schema error: 'status.values' must exactly match 'statuses'")
	}
	for _, v := range statusDef.Values {
		if !statusSet[v] {
			return fmt.Errorf("schema error: 'status.values' contains %q which is not in 'statuses'", v)
		}
	}

	return validateSchemaRefs(s, statusSet)
}

// DefaultValue returns the parsed default value for a field, or nil if none defined.
func DefaultValue(fdef FieldDef) interface{} {
	if fdef.Default == nil {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal(fdef.Default, &v); err != nil {
		return nil
	}
	return v
}
