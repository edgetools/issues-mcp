package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
)

// RegisterGetWorkLog registers the get_work_log tool.
func RegisterGetWorkLog(srv *server.MCPServer, store *issues.Store) {
	tool := mcp.NewTool("get_work_log",
		mcp.WithDescription("Returns the structured work log entries for an issue. Each entry includes timestamp, author, and content."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Issue ID (e.g. 'COMBAT-AGGRO-001')"),
		),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult("id is required"), nil
		}

		f, err := store.FindByID(id)
		if err != nil {
			return errResult(err.Error()), nil
		}

		entries := f.WorkLog
		if entries == nil {
			entries = []issues.WorkLogEntry{}
		}

		return jsonResult(map[string]any{
			"id":      f.ID,
			"entries": entries,
		})
	})
}
