package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
)

// RegisterAddComment registers the add_comment tool.
func RegisterAddComment(srv *server.MCPServer, store *issues.Store) {
	tool := mcp.NewTool("add_comment",
		mcp.WithDescription("Appends a timestamped entry to the issue's work log. The work log is append-only and is never locked regardless of issue status."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Issue ID to comment on"),
		),
		mcp.WithString("author",
			mcp.Required(),
			mcp.Description("Author identifier (e.g. 'impl-agent', 'review-semantic')"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Comment content (plain text or markdown)"),
		),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult("id is required"), nil
		}
		author, err := req.RequireString("author")
		if err != nil {
			return errResult("author is required"), nil
		}
		content, err := req.RequireString("content")
		if err != nil {
			return errResult("content is required"), nil
		}

		f, err := store.FindByID(id)
		if err != nil {
			return errResult(err.Error()), nil
		}

		ts := f.AppendWorkLogEntry(author, content)

		if err := store.Write(f); err != nil {
			return errResult("writing issue: " + err.Error()), nil
		}

		return jsonResult(map[string]any{
			"id":        f.ID,
			"timestamp": ts,
			"added":     true,
		})
	})
}
