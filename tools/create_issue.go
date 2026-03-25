package tools

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
	"github.com/edgetools/issues-mcp/schema"
)

// RegisterCreateIssue registers the create_issue tool.
func RegisterCreateIssue(srv *server.MCPServer, store *issues.Store, s *schema.Schema) {
	tool := mcp.NewTool("create_issue",
		mcp.WithDescription("Creates a new issue. The ID is auto-generated from the area field. New issues always start in the backlog status."),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Issue title"),
		),
		mcp.WithString("area",
			mcp.Required(),
			mcp.Description("Issue area (e.g. 'Combat/Aggro'). Determines the ID prefix."),
		),
		mcp.WithString("body",
			mcp.Description("Markdown body content (problem statement, hypothesis, notes). Optional."),
		),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title, err := req.RequireString("title")
		if err != nil {
			return errResult("title is required"), nil
		}
		area, err := req.RequireString("area")
		if err != nil {
			return errResult("area is required"), nil
		}

		args := req.GetArguments()
		body, _ := getString(args, "body")

		// Build initial fields from schema defaults
		fields := make(map[string]any)
		for fname, fdef := range s.Fields {
			if fdef.Generated {
				continue
			}
			if def := schema.DefaultValue(fdef); def != nil {
				fields[fname] = def
			}
		}

		// Apply caller-provided fields
		fields["title"] = title
		fields["area"] = area

		// Apply any additional schema-defined fields provided in args (beyond title/area/body)
		for fname, fdef := range s.Fields {
			if fdef.Generated || fname == "title" || fname == "area" || fname == "status" {
				continue
			}
			if val, ok := args[fname]; ok && val != nil {
				fields[fname] = val
			}
		}

		// Validate provided field values
		for fname, val := range fields {
			fdef, ok := s.Fields[fname]
			if !ok {
				continue
			}
			if fdef.Generated {
				return errResult(fmt.Sprintf("field '%s' is generated and cannot be set", fname)), nil
			}
			if err := schema.ValidateFieldValue(fdef, val); err != nil {
				return errResult(fmt.Sprintf("field '%s': %s", fname, err.Error())), nil
			}
		}

		// Generate ID
		id, err := issues.NextID(store.IssuesDir, s.Statuses, area)
		if err != nil {
			return errResult(fmt.Sprintf("generating ID: %v", err)), nil
		}
		fields["id"] = id

		// Determine initial status (always backlog for new issues)
		initialStatus := s.Statuses[0] // first status is always backlog per spec
		fields["status"] = initialStatus

		// Build the issue file
		f := &issues.IssueFile{
			Fields:    fields,
			Body:      body,
			DirStatus: initialStatus,
			ID:        id,
		}
		absPath := filepath.Join(store.IssuesDir, initialStatus, id+".md")
		f.AbsPath = absPath
		f.RelPath = initialStatus + "/" + id + ".md"

		if err := store.Write(f); err != nil {
			return errResult(fmt.Sprintf("writing issue: %v", err)), nil
		}

		return jsonResult(map[string]any{
			"id":      id,
			"path":    f.RelPath,
			"created": true,
		})
	})
}
