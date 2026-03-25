package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
	"github.com/edgetools/issues-mcp/schema"
)

// RegisterUpdateBody registers the update_body tool.
func RegisterUpdateBody(srv *server.MCPServer, store *issues.Store, s *schema.Schema) {
	tool := mcp.NewTool("update_body",
		mcp.WithDescription("Replaces the body content of an issue (between frontmatter and work log). Subject to locks: if the issue's status is in locks.body.locked_in, the update is rejected."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Issue ID to update"),
		),
		mcp.WithString("body",
			mcp.Required(),
			mcp.Description("New markdown body content"),
		),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult("id is required"), nil
		}
		body, err := req.RequireString("body")
		if err != nil {
			return errResult("body is required"), nil
		}

		f, err := store.FindByID(id)
		if err != nil {
			return errResult(err.Error()), nil
		}

		// Check body lock
		if s.Locks.Body != nil {
			for _, lockedStatus := range s.Locks.Body.LockedIn {
				if f.DirStatus == lockedStatus {
					// Find an unlocked status to suggest
					unlocked := unlockSuggestion(s)
					return errResult(fmt.Sprintf(
						"cannot update body of %s: body is locked while status is '%s' (move to '%s' to edit)",
						id, f.DirStatus, unlocked,
					)), nil
				}
			}
		}

		f.Body = strings.TrimSpace(body)
		if err := store.Write(f); err != nil {
			return errResult(fmt.Sprintf("writing issue: %v", err)), nil
		}

		return jsonResult(map[string]any{
			"id":      f.ID,
			"updated": true,
		})
	})
}

// unlockSuggestion returns the first status not in the locked_in list, to suggest in error messages.
func unlockSuggestion(s *schema.Schema) string {
	locked := make(map[string]bool)
	if s.Locks.Body != nil {
		for _, st := range s.Locks.Body.LockedIn {
			locked[st] = true
		}
	}
	for _, st := range s.Statuses {
		if !locked[st] {
			return st
		}
	}
	return "an unlocked status"
}
