package tools_test

import (
	"testing"

	"github.com/edgetools/issues-mcp/tools"
)

func TestCreateIssue_RequiredProjectFieldsEnforced(t *testing.T) {
	t.Run("title required by schema: missing title is rejected", func(t *testing.T) {
		s := testSchema(t, map[string]any{
			"title": map[string]any{"type": "string", "required": true},
		})
		store := testStore(t, s)
		srv := testServer()
		tools.RegisterCreateIssue(srv, store, s)

		result := callTool(t, srv, "create_issue", map[string]any{
			"area": "Test",
		})
		if errMsg := toolError(result); errMsg == "" {
			t.Error("expected error when required 'title' is missing, got success")
		}
	})

	t.Run("title required by schema: providing title succeeds", func(t *testing.T) {
		s := testSchema(t, map[string]any{
			"title": map[string]any{"type": "string", "required": true},
		})
		store := testStore(t, s)
		srv := testServer()
		tools.RegisterCreateIssue(srv, store, s)

		result := callTool(t, srv, "create_issue", map[string]any{
			"area":  "Test",
			"title": "My issue",
		})
		if errMsg := toolError(result); errMsg != "" {
			t.Errorf("unexpected error: %s", errMsg)
		}
		if result["id"] == "" {
			t.Error("expected non-empty id in result")
		}
	})

	t.Run("title not in schema: omitting title succeeds", func(t *testing.T) {
		s := testSchema(t, map[string]any{}) // no title field
		store := testStore(t, s)
		srv := testServer()
		tools.RegisterCreateIssue(srv, store, s)

		result := callTool(t, srv, "create_issue", map[string]any{
			"area": "Test",
		})
		if errMsg := toolError(result); errMsg != "" {
			t.Errorf("unexpected error when title not in schema: %s", errMsg)
		}
		if result["created"] != true {
			t.Error("expected created=true in result")
		}
	})
}

func TestCreateIssue_DependsOnDefaultsToEmpty(t *testing.T) {
	s := testSchema(t, map[string]any{})
	store := testStore(t, s)
	srv := testServer()
	tools.RegisterCreateIssue(srv, store, s)
	tools.RegisterGetIssue(srv, store)

	createResult := callTool(t, srv, "create_issue", map[string]any{"area": "Test"})
	if errMsg := toolError(createResult); errMsg != "" {
		t.Fatalf("create_issue failed: %s", errMsg)
	}
	id := createResult["id"].(string)

	getResult := callTool(t, srv, "get_issue", map[string]any{"id": id})
	depVal, ok := getResult["depends_on"]
	if !ok {
		t.Fatal("expected 'depends_on' field in issue, not present")
	}
	deps, ok := depVal.([]any)
	if !ok {
		t.Fatalf("expected depends_on to be a list, got %T", depVal)
	}
	if len(deps) != 0 {
		t.Errorf("expected empty depends_on, got %v", deps)
	}
}

func TestCreateIssue_IntrinsicFieldsSet(t *testing.T) {
	s := testSchema(t, map[string]any{})
	store := testStore(t, s)
	srv := testServer()
	tools.RegisterCreateIssue(srv, store, s)
	tools.RegisterGetIssue(srv, store)

	createResult := callTool(t, srv, "create_issue", map[string]any{"area": "Combat/Aggro"})
	if errMsg := toolError(createResult); errMsg != "" {
		t.Fatalf("create_issue failed: %s", errMsg)
	}
	id := createResult["id"].(string)

	issue := callTool(t, srv, "get_issue", map[string]any{"id": id})
	if issue["status"] != s.Statuses[0] {
		t.Errorf("status = %q, want %q", issue["status"], s.Statuses[0])
	}
	if issue["area"] != "Combat/Aggro" {
		t.Errorf("area = %q, want %q", issue["area"], "Combat/Aggro")
	}
}
