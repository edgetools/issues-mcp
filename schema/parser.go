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
// statuses array is valid, and status references are consistent.
func validateSchema(s *Schema) error {
	// id, area, status, and depends_on are MCP-intrinsic and must not be declared in fields.
	for _, name := range []string{"id", "area", "status", "depends_on"} {
		if _, ok := s.Fields[name]; ok {
			return fmt.Errorf("schema error: '%s' must not be declared in fields (it is an MCP-intrinsic field)", name)
		}
	}

	// statuses must exist and contain at least one entry.
	if len(s.Statuses) == 0 {
		return fmt.Errorf("schema error: 'statuses' must contain at least one entry")
	}

	statusSet := make(map[string]bool, len(s.Statuses))
	for _, sv := range s.Statuses {
		statusSet[sv] = true
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
