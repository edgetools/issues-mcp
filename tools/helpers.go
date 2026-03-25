package tools

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/edgetools/issues-mcp/issues"
)

// jsonResult serializes v as JSON and returns it as a tool text result.
func jsonResult(v interface{}) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errResult(fmt.Sprintf("internal serialization error: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// errResult returns a structured {"error": msg} JSON tool result.
func errResult(msg string) *mcp.CallToolResult {
	data, _ := json.Marshal(map[string]string{"error": msg})
	return mcp.NewToolResultText(string(data))
}

// issueToMap converts an IssueFile's frontmatter fields to a map, adding the "path" key.
func issueToMap(f *issues.IssueFile) map[string]any {
	result := make(map[string]any, len(f.Fields)+1)
	for k, v := range f.Fields {
		result[k] = v
	}
	result["path"] = f.RelPath
	return result
}

// getString extracts a string argument from the tool call arguments map.
func getString(args map[string]any, key string) (string, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// getBool extracts a bool argument from the tool call arguments map.
func getBool(args map[string]any, key string) (bool, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// getMap extracts a nested object argument from the tool call arguments map.
func getMap(args map[string]any, key string) (map[string]any, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return nil, false
	}
	m, ok := v.(map[string]any)
	return m, ok
}
