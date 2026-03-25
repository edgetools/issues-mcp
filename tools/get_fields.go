package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/schema"
)

// RegisterGetFields registers the get_fields tool.
func RegisterGetFields(srv *server.MCPServer, s *schema.Schema) {
	tool := mcp.NewTool("get_fields",
		mcp.WithDescription("Returns all issue fields, their types, whether they are writable, and valid values for enum fields. Call once per session to learn the schema before interacting with issues."),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetFields(s)(ctx, req)
	})
}

func handleGetFields(s *schema.Schema) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		type FieldInfo struct {
			Type     string   `json:"type"`
			Writable bool     `json:"writable"`
			Values   []string `json:"values,omitempty"`
			Note     string   `json:"note,omitempty"`
		}

		fields := make(map[string]FieldInfo, len(s.Fields))
		for name, fdef := range s.Fields {
			info := FieldInfo{
				Type:     string(fdef.Type),
				Writable: !fdef.Generated,
			}
			switch name {
			case "status":
				info.Writable = false
				info.Note = "use move_issue to change status"
				info.Values = fdef.Values
			case "id":
				info.Writable = false
				info.Note = "auto-generated from area"
			default:
				if fdef.Type == schema.FieldTypeEnum {
					info.Values = fdef.Values
				}
			}
			fields[name] = info
		}

		return jsonResult(map[string]interface{}{"fields": fields})
	}
}
