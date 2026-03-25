package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
)

// RegisterListIssues registers the list_issues tool.
func RegisterListIssues(srv *server.MCPServer, store *issues.Store) {
	tool := mcp.NewTool("list_issues",
		mcp.WithDescription("Lists all issues with their frontmatter. Supports optional filtering by status and area prefix."),
		mcp.WithString("status",
			mcp.Description("Filter by status (e.g. 'backlog', 'active', 'done'). Omit to return all statuses."),
		),
		mcp.WithString("area",
			mcp.Description("Filter by area prefix (e.g. 'Combat' matches 'Combat/Aggro' and 'Combat/CrowdControl'). Omit to return all areas."),
		),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		statusFilter, _ := getString(args, "status")
		areaFilter, _ := getString(args, "area")

		files, err := store.ListAll(statusFilter, areaFilter)
		if err != nil {
			return errResult(err.Error()), nil
		}

		issueList := make([]map[string]interface{}, 0, len(files))
		for _, f := range files {
			issueList = append(issueList, issueToMap(f))
		}
		return jsonResult(map[string]interface{}{"issues": issueList})
	})
}
