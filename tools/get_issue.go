package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
)

// RegisterGetIssue registers the get_issue tool.
func RegisterGetIssue(srv *server.MCPServer, store *issues.Store) {
	tool := mcp.NewTool("get_issue",
		mcp.WithDescription("Returns the frontmatter of a single issue by ID. Does not include the body or work log."),
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

		return jsonResult(issueToMap(f))
	})
}
