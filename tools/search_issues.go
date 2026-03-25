package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
)

// RegisterSearchIssues registers the search_issues tool.
func RegisterSearchIssues(srv *server.MCPServer, store *issues.Store) {
	tool := mcp.NewTool("search_issues",
		mcp.WithDescription("Case-insensitive keyword search across frontmatter, body, and work log. Multiple keywords are matched individually. Returns matching issues with line-level context snippets."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Space-separated keywords to search for"),
		),
		mcp.WithString("status",
			mcp.Description("Restrict search to a specific status directory. Omit to search all statuses."),
		),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return errResult("query is required"), nil
		}

		args := req.GetArguments()
		statusFilter, _ := getString(args, "status")

		results, err := store.Search(query, statusFilter)
		if err != nil {
			return errResult(err.Error()), nil
		}

		if results == nil {
			results = []issues.SearchResult{}
		}

		return jsonResult(map[string]any{"results": results})
	})
}
