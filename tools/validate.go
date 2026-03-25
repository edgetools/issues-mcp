package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
	"github.com/edgetools/issues-mcp/schema"
)

// RegisterValidate registers the validate tool.
func RegisterValidate(srv *server.MCPServer, store *issues.Store, s *schema.Schema) {
	tool := mcp.NewTool("validate",
		mcp.WithDescription("Validates one issue or all issues against the schema. Returns structured pass/fail results."),
		mcp.WithString("id",
			mcp.Description("Validate a single issue by ID. Omit if using 'all'."),
		),
		mcp.WithBoolean("all",
			mcp.Description("Set to true to validate all issues."),
		),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		id, hasID := getString(args, "id")
		all, _ := getBool(args, "all")

		if !hasID && !all {
			return errResult("provide either 'id' or 'all: true'"), nil
		}

		var records []schema.IssueRecord

		if hasID && id != "" {
			f, err := store.FindByID(id)
			if err != nil {
				return errResult(err.Error()), nil
			}
			// Build allIDs for cross-reference checking
			allRecords, err := store.AllRecords()
			if err != nil {
				return errResult(err.Error()), nil
			}
			allIDs := make(map[string]bool, len(allRecords))
			for _, r := range allRecords {
				allIDs[r.ID] = true
			}

			errs := schema.ValidateRecord(s, schema.IssueRecord{
				ID:        f.ID,
				Fields:    f.Fields,
				DirStatus: f.DirStatus,
			}, allIDs)

			return jsonResult(map[string]any{
				"valid":  len(errs) == 0,
				"errors": nilIfEmpty(errs),
			})
		}

		// Validate all
		var err error
		records, err = store.AllRecords()
		if err != nil {
			return errResult(err.Error()), nil
		}

		errs := schema.ValidateAll(s, records)
		return jsonResult(map[string]any{
			"valid":  len(errs) == 0,
			"errors": nilIfEmpty(errs),
		})
	})
}

// nilIfEmpty returns nil if the slice has no elements, otherwise the slice itself.
// This ensures JSON serializes as null instead of [] when there are no errors.
func nilIfEmpty[T any](s []T) any {
	if len(s) == 0 {
		return nil
	}
	return s
}
