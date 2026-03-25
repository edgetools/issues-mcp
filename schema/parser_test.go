package schema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/edgetools/issues-mcp/schema"
)

// baseSchemaData returns a minimal valid schema as a map for easy modification.
func baseSchemaData() map[string]any {
	return map[string]any{
		"statuses": []string{"todo", "done"},
		"fields":   map[string]any{},
		"transitions": map[string]any{
			"todo": []string{"done"},
			"done": []string{},
		},
		"gates": map[string]any{},
		"locks": map[string]any{},
	}
}

func writeSchemaFile(t *testing.T, data map[string]any) string {
	t.Helper()
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	path := filepath.Join(t.TempDir(), "schema.json")
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	return path
}

func TestLoad_IntrinsicFieldRejections(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		fieldDef  map[string]any
	}{
		{
			name:      "status in fields",
			fieldName: "status",
			fieldDef:  map[string]any{"type": "enum", "values": []string{"todo", "done"}, "default": "todo"},
		},
		{
			name:      "depends_on in fields",
			fieldName: "depends_on",
			fieldDef:  map[string]any{"type": "list", "item_type": "string"},
		},
		{
			name:      "id in fields",
			fieldName: "id",
			fieldDef:  map[string]any{"type": "string"},
		},
		{
			name:      "area in fields",
			fieldName: "area",
			fieldDef:  map[string]any{"type": "string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := baseSchemaData()
			data["fields"] = map[string]any{tt.fieldName: tt.fieldDef}
			path := writeSchemaFile(t, data)

			_, err := schema.Load(path)
			if err == nil {
				t.Errorf("expected error for schema declaring %q in fields, got nil", tt.fieldName)
			}
		})
	}
}

func TestLoad_StatusesValidation(t *testing.T) {
	t.Run("empty statuses array", func(t *testing.T) {
		data := baseSchemaData()
		data["statuses"] = []string{}
		path := writeSchemaFile(t, data)

		_, err := schema.Load(path)
		if err == nil {
			t.Error("expected error for empty statuses array, got nil")
		}
	})

	t.Run("missing statuses key", func(t *testing.T) {
		data := baseSchemaData()
		delete(data, "statuses")
		path := writeSchemaFile(t, data)

		_, err := schema.Load(path)
		if err == nil {
			t.Error("expected error for missing statuses key, got nil")
		}
	})
}

func TestLoad_ValidSchema(t *testing.T) {
	t.Run("minimal schema with no project fields", func(t *testing.T) {
		path := writeSchemaFile(t, baseSchemaData())
		s, err := schema.Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s.Statuses) != 2 {
			t.Errorf("expected 2 statuses, got %d", len(s.Statuses))
		}
	})

	t.Run("schema with project fields", func(t *testing.T) {
		data := baseSchemaData()
		data["fields"] = map[string]any{
			"title": map[string]any{"type": "string", "required": true},
		}
		path := writeSchemaFile(t, data)
		s, err := schema.Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := s.Fields["title"]; !ok {
			t.Error("expected 'title' field in loaded schema")
		}
	})
}
