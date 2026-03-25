package tools_test

import (
	"testing"

	"github.com/edgetools/issues-mcp/tools"
)


func TestUpdateFields_DependsOnValidIDs(t *testing.T) {
	s := testSchema(t, map[string]any{})
	store := testStore(t, s)
	srv := testServer()
	tools.RegisterCreateIssue(srv, store, s)
	tools.RegisterUpdateFields(srv, store, s)

	// Create two issues
	resA := callTool(t, srv, "create_issue", map[string]any{"area": "Test"})
	if errMsg := toolError(resA); errMsg != "" {
		t.Fatalf("create issue A: %s", errMsg)
	}
	idA := resA["id"].(string)

	resB := callTool(t, srv, "create_issue", map[string]any{"area": "Test"})
	if errMsg := toolError(resB); errMsg != "" {
		t.Fatalf("create issue B: %s", errMsg)
	}
	idB := resB["id"].(string)

	// Set A's depends_on to point at B
	result := callTool(t, srv, "update_fields", map[string]any{
		"id": idA,
		"fields": map[string]any{
			"depends_on": []any{idB},
		},
	})
	if errMsg := toolError(result); errMsg != "" {
		t.Errorf("expected success updating depends_on to valid ID, got error: %s", errMsg)
	}
}

func TestUpdateFields_DependsOnNonexistentID(t *testing.T) {
	s := testSchema(t, map[string]any{})
	store := testStore(t, s)
	srv := testServer()
	tools.RegisterCreateIssue(srv, store, s)
	tools.RegisterUpdateFields(srv, store, s)

	resA := callTool(t, srv, "create_issue", map[string]any{"area": "Test"})
	idA := resA["id"].(string)

	result := callTool(t, srv, "update_fields", map[string]any{
		"id": idA,
		"fields": map[string]any{
			"depends_on": []any{"NONEXISTENT-999"},
		},
	})
	if errMsg := toolError(result); errMsg == "" {
		t.Error("expected error for depends_on with nonexistent ID, got success")
	}
}

func TestUpdateFields_StatusRejected(t *testing.T) {
	s := testSchema(t, map[string]any{})
	store := testStore(t, s)
	srv := testServer()
	tools.RegisterCreateIssue(srv, store, s)
	tools.RegisterUpdateFields(srv, store, s)

	res := callTool(t, srv, "create_issue", map[string]any{"area": "Test"})
	id := res["id"].(string)

	result := callTool(t, srv, "update_fields", map[string]any{
		"id": id,
		"fields": map[string]any{
			"status": "done",
		},
	})
	if errMsg := toolError(result); errMsg == "" {
		t.Error("expected error when setting status via update_fields, got success")
	}
}

func TestUpdateFields_IDRejected(t *testing.T) {
	s := testSchema(t, map[string]any{})
	store := testStore(t, s)
	srv := testServer()
	tools.RegisterCreateIssue(srv, store, s)
	tools.RegisterUpdateFields(srv, store, s)

	res := callTool(t, srv, "create_issue", map[string]any{"area": "Test"})
	id := res["id"].(string)

	result := callTool(t, srv, "update_fields", map[string]any{
		"id": id,
		"fields": map[string]any{
			"id": "SOMETHING-ELSE-001",
		},
	})
	if errMsg := toolError(result); errMsg == "" {
		t.Error("expected error when updating 'id' field, got success")
	}
}

func TestUpdateFields_AreaRejected(t *testing.T) {
	s := testSchema(t, map[string]any{})
	store := testStore(t, s)
	srv := testServer()
	tools.RegisterCreateIssue(srv, store, s)
	tools.RegisterUpdateFields(srv, store, s)

	res := callTool(t, srv, "create_issue", map[string]any{"area": "Test"})
	id := res["id"].(string)

	result := callTool(t, srv, "update_fields", map[string]any{
		"id": id,
		"fields": map[string]any{
			"area": "OtherArea",
		},
	})
	if errMsg := toolError(result); errMsg == "" {
		t.Error("expected error when updating 'area' field, got success")
	}
}
