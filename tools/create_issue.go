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
		mcp.WithDescription("Creates a new issue. The ID is auto-generated from the area field. New issues always start in the first status. Call get_fields first to learn which fields are required."),
		mcp.WithString("area",
			mcp.Required(),
			mcp.Description("Issue area (e.g. 'Combat/Aggro'). Determines the ID prefix."),
		),
		mcp.WithString("body",
			mcp.Description("Markdown body content (problem statement, hypothesis, notes). Optional."),
		),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		// Apply caller-provided schema-defined fields
		for fname, fdef := range s.Fields {
			if fdef.Generated {
				continue
			}
			if val, ok := args[fname]; ok && val != nil {
				fields[fname] = val
			}
		}

		// Validate field values and enforce required fields
		for fname, fdef := range s.Fields {
			if fdef.Generated {
				continue
			}
			if fdef.Required {
				if _, ok := fields[fname]; !ok {
					return errResult(fmt.Sprintf("field '%s' is required", fname)), nil
				}
			}
			if val, ok := fields[fname]; ok {
				if err := schema.ValidateFieldValue(fdef, val); err != nil {
					return errResult(fmt.Sprintf("field '%s': %s", fname, err.Error())), nil
				}
			}
		}

		// Generate ID
		id, err := issues.NextID(store.IssuesDir, s.Statuses, area)
		if err != nil {
			return errResult(fmt.Sprintf("generating ID: %v", err)), nil
		}
		fields["id"] = id
		fields["area"] = area

		// Determine initial status (always first status for new issues)
		initialStatus := s.Statuses[0]
		fields["status"] = initialStatus

		// depends_on defaults to empty list
		if _, ok := fields["depends_on"]; !ok {
			fields["depends_on"] = []interface{}{}
		}

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
