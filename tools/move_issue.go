package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
	"github.com/edgetools/issues-mcp/schema"
)

// RegisterMoveIssue registers the move_issue tool.
// s is accepted for API consistency; transition and gate logic lives in the store.
func RegisterMoveIssue(srv *server.MCPServer, store *issues.Store, _ *schema.Schema) {
	tool := mcp.NewTool("move_issue",
		mcp.WithDescription("Moves an issue to a new status. Enforces transition rules and gate conditions. Updates frontmatter and moves the file atomically."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Issue ID to move"),
		),
		mcp.WithString("status",
			mcp.Required(),
			mcp.Description("Target status (e.g. 'active', 'done')"),
		),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult("id is required"), nil
		}
		newStatus, err := req.RequireString("status")
		if err != nil {
			return errResult("status is required"), nil
		}

		f, err := store.FindByID(id)
		if err != nil {
			return errResult(err.Error()), nil
		}

		if err := store.CheckAndMove(f, newStatus); err != nil {
			return errResult(err.Error()), nil
		}

		return jsonResult(map[string]any{
			"id":     f.ID,
			"status": f.DirStatus,
			"path":   f.RelPath,
			"moved":  true,
		})
	})
}
