package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
)

// RegisterGetIssueBody registers the get_issue_body tool.
func RegisterGetIssueBody(srv *server.MCPServer, store *issues.Store) {
	tool := mcp.NewTool("get_issue_body",
		mcp.WithDescription("Returns the markdown body of an issue (between the frontmatter and the work log separator). Does not include frontmatter or work log."),
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

		return jsonResult(map[string]any{
			"id":   f.ID,
			"body": f.Body,
		})
	})
}
