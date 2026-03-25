package schema

import (
	"fmt"
	"strings"
)

// ValidationError describes a single validation failure.
type ValidationError struct {
	ID    string `json:"id"`
	Field string `json:"field"`
	Error string `json:"error"`
}

// IssueRecord is the minimal representation of an issue needed for validation.
type IssueRecord struct {
	ID        string
	Fields    map[string]interface{}
	DirStatus string // status inferred from the issue's directory location
}

// ValidateFieldValue checks a single value against its field definition.
func ValidateFieldValue(fdef FieldDef, value interface{}) error {
	switch fdef.Type {
	case FieldTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case FieldTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
	case FieldTypeInt:
		switch value.(type) {
		case int, int64, float64:
			// accept numeric types
		default:
			return fmt.Errorf("expected int, got %T", value)
		}
	case FieldTypeEnum:
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string (enum), got %T", value)
		}
		for _, v := range fdef.Values {
			if v == s {
				return nil
			}
		}
		return fmt.Errorf("invalid enum value %q (valid: %s)", s, strings.Join(fdef.Values, ", "))
	case FieldTypeList:
		switch v := value.(type) {
		case []interface{}:
			if fdef.ItemType == "string" {
				for i, item := range v {
					if _, ok := item.(string); !ok {
						return fmt.Errorf("list item %d: expected string, got %T", i, item)
					}
				}
			}
		case nil:
			// nil is acceptable for a list (treated as empty)
		default:
			return fmt.Errorf("expected list, got %T", value)
		}
	}
	return nil
}

// areaToPrefix converts an area string to an ID prefix.
// "Combat/Aggro" → "COMBAT-AGGRO"
func areaToPrefix(area string) string {
	parts := strings.Split(area, "/")
	for i, p := range parts {
		parts[i] = strings.ToUpper(p)
	}
	return strings.Join(parts, "-")
}

// ValuesEqual reports whether two interface values are semantically equal,
// handling JSON numeric types (float64) compared against bool and string.
func ValuesEqual(a, b interface{}) bool {
	switch bv := b.(type) {
	case bool:
		av, ok := a.(bool)
		return ok && av == bv
	case string:
		av, ok := a.(string)
		return ok && av == bv
	case float64:
		switch av := a.(type) {
		case float64:
			return av == bv
		case int:
			return float64(av) == bv
		case int64:
			return float64(av) == bv
		}
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// ValidateRecord validates a single issue record against the schema.
// allIDs is the complete set of known issue IDs, used for depends_on cross-reference checks.
func ValidateRecord(s *Schema, rec IssueRecord, allIDs map[string]bool) []ValidationError {
	var errs []ValidationError
	add := func(field, msg string) {
		errs = append(errs, ValidationError{ID: rec.ID, Field: field, Error: msg})
	}

	// Required fields (non-generated)
	for fname, fdef := range s.Fields {
		if fdef.Required && !fdef.Generated {
			if _, ok := rec.Fields[fname]; !ok {
				add(fname, "required field is missing")
			}
		}
	}

	// Field type/value validation
	for fname, val := range rec.Fields {
		fdef, ok := s.Fields[fname]
		if !ok {
			continue // ignore unknown fields
		}
		if err := ValidateFieldValue(fdef, val); err != nil {
			add(fname, err.Error())
		}
	}

	// depends_on cross-references
	if depVal, ok := rec.Fields["depends_on"]; ok && depVal != nil {
		if deps, ok := toStringSlice(depVal); ok {
			for _, dep := range deps {
				if dep != "" && !allIDs[dep] {
					add("depends_on", fmt.Sprintf("references unknown issue ID: %s", dep))
				}
			}
		}
	}

	// Status/directory consistency
	if statusVal, ok := rec.Fields["status"]; ok {
		if status, ok := statusVal.(string); ok && status != rec.DirStatus {
			add("status", fmt.Sprintf("frontmatter says '%s' but file is in %s/ directory", status, rec.DirStatus))
		}
	}

	// Gate condition consistency
	if gate, ok := s.Gates[rec.DirStatus]; ok {
		for field, required := range gate.Requires {
			actual := rec.Fields[field]
			if !ValuesEqual(actual, required) {
				add(field, fmt.Sprintf("issue is in '%s' status but gate requires %s=%v (current: %v)", rec.DirStatus, field, required, actual))
			}
		}
	}

	// ID prefix matches area
	if areaVal, ok := rec.Fields["area"]; ok {
		if area, ok := areaVal.(string); ok && area != "" {
			prefix := areaToPrefix(area)
			if !strings.HasPrefix(rec.ID, prefix+"-") {
				add("id", fmt.Sprintf("ID %s does not match expected prefix %s for area %q", rec.ID, prefix, area))
			}
		}
	}

	return errs
}

// ValidateAll validates a slice of issue records, including duplicate ID detection.
func ValidateAll(s *Schema, records []IssueRecord) []ValidationError {
	// Build all-IDs set and detect duplicates
	allIDs := make(map[string]bool, len(records))
	idCount := make(map[string]int, len(records))
	for _, rec := range records {
		allIDs[rec.ID] = true
		idCount[rec.ID]++
	}

	var errs []ValidationError

	for id, count := range idCount {
		if count > 1 {
			errs = append(errs, ValidationError{
				ID:    id,
				Field: "id",
				Error: fmt.Sprintf("duplicate ID found %d times", count),
			})
		}
	}

	for _, rec := range records {
		errs = append(errs, ValidateRecord(s, rec, allIDs)...)
	}

	return errs
}

func toStringSlice(v interface{}) ([]string, bool) {
	switch s := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			} else {
				return nil, false
			}
		}
		return result, true
	case []string:
		return s, true
	}
	return nil, false
}
