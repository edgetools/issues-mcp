package tools_test

import (
	"testing"

	"github.com/edgetools/issues-mcp/tools"
)

func TestGetFields_StatusIsIntrinsic(t *testing.T) {
	s := testSchema(t, map[string]any{})
	store := testStore(t, s)
	srv := testServer()
	tools.RegisterGetFields(srv, s)
	tools.RegisterCreateIssue(srv, store, s) // satisfy store dependency

	result := callTool(t, srv, "get_fields", map[string]any{})
	if errMsg := toolError(result); errMsg != "" {
		t.Fatalf("get_fields returned error: %s", errMsg)
	}

	fields, ok := result["fields"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'fields' object in result, got: %T", result["fields"])
	}

	statusField, ok := fields["status"].(map[string]any)
	if !ok {
		t.Fatal("expected 'status' field in get_fields output")
	}

	if statusField["type"] != "enum" {
		t.Errorf("status type = %q, want %q", statusField["type"], "enum")
	}
	if statusField["writable"] != false {
		t.Errorf("status writable = %v, want false", statusField["writable"])
	}

	// Values should come from the statuses array
	values, ok := statusField["values"].([]any)
	if !ok {
		t.Fatalf("expected status.values to be a list, got %T", statusField["values"])
	}
	if len(values) != len(s.Statuses) {
		t.Errorf("status.values len = %d, want %d", len(values), len(s.Statuses))
	}
	for i, want := range s.Statuses {
		if values[i] != want {
			t.Errorf("status.values[%d] = %q, want %q", i, values[i], want)
		}
	}
}

func TestGetFields_DependsOnIsIntrinsic(t *testing.T) {
	s := testSchema(t, map[string]any{})
	srv := testServer()
	tools.RegisterGetFields(srv, s)

	result := callTool(t, srv, "get_fields", map[string]any{})
	fields := result["fields"].(map[string]any)

	depField, ok := fields["depends_on"].(map[string]any)
	if !ok {
		t.Fatal("expected 'depends_on' field in get_fields output")
	}

	if depField["type"] != "list" {
		t.Errorf("depends_on type = %q, want %q", depField["type"], "list")
	}
	if depField["writable"] != true {
		t.Errorf("depends_on writable = %v, want true", depField["writable"])
	}
}

func TestGetFields_ProjectFieldsIncluded(t *testing.T) {
	s := testSchema(t, map[string]any{
		"title": map[string]any{"type": "string", "required": true},
	})
	srv := testServer()
	tools.RegisterGetFields(srv, s)

	result := callTool(t, srv, "get_fields", map[string]any{})
	fields := result["fields"].(map[string]any)

	if _, ok := fields["title"]; !ok {
		t.Error("expected 'title' project field in get_fields output")
	}
	// Intrinsic fields must not be lost
	for _, intrinsic := range []string{"id", "area", "status", "depends_on"} {
		if _, ok := fields[intrinsic]; !ok {
			t.Errorf("expected intrinsic field %q in get_fields output", intrinsic)
		}
	}
}
