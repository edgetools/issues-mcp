package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
	"github.com/edgetools/issues-mcp/schema"
)

// testSchema builds and loads a schema from a fields map, using a standard two-status setup.
func testSchema(t *testing.T, fields map[string]any) *schema.Schema {
	t.Helper()
	data := map[string]any{
		"statuses": []string{"backlog", "done"},
		"fields":   fields,
		"transitions": map[string]any{
			"backlog": []string{"done"},
			"done":    []string{},
		},
		"gates": map[string]any{},
		"locks": map[string]any{},
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	path := filepath.Join(t.TempDir(), "schema.json")
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	s, err := schema.Load(path)
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	return s
}

// testStore creates a Store backed by a fresh temp directory.
func testStore(t *testing.T, s *schema.Schema) *issues.Store {
	t.Helper()
	store, err := issues.NewStore(t.TempDir(), s)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return store
}

// testServer creates a new MCP server for tool registration in tests.
func testServer() *server.MCPServer {
	return server.NewMCPServer("test", "1.0")
}

// callTool invokes a registered tool handler by name and returns the parsed JSON response.
func callTool(t *testing.T, srv *server.MCPServer, toolName string, args map[string]any) map[string]any {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := srv.GetTool(toolName).Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("tool %q returned unexpected Go error: %v", toolName, err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	var m map[string]any
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		t.Fatalf("unmarshal tool result: %v (raw: %s)", err, text)
	}
	return m
}

// toolError extracts the error string from a parsed tool response, or "" if none.
func toolError(m map[string]any) string {
	if e, ok := m["error"].(string); ok {
		return e
	}
	return ""
}
