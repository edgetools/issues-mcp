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
	return &s, nil
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
