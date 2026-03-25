package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
	"github.com/edgetools/issues-mcp/schema"
)

// RegisterUpdateFields registers the update_fields tool.
func RegisterUpdateFields(srv *server.MCPServer, store *issues.Store, s *schema.Schema) {
	tool := mcp.NewTool("update_fields",
		mcp.WithDescription("Updates one or more frontmatter fields on an existing issue. If 'status' is included, applies transition and gate checks before moving the file."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Issue ID to update"),
		),
		mcp.WithObject("fields",
			mcp.Required(),
			mcp.Description("Map of field names to new values"),
		),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return errResult("id is required"), nil
		}

		args := req.GetArguments()
		fieldsArg, ok := getMap(args, "fields")
		if !ok || len(fieldsArg) == 0 {
			return errResult("fields is required and must be a non-empty object"), nil
		}

		f, err := store.FindByID(id)
		if err != nil {
			return errResult(err.Error()), nil
		}

		// Separate status from other fields
		var newStatus string
		hasStatus := false
		if sv, ok := fieldsArg["status"]; ok {
			if s, ok := sv.(string); ok {
				newStatus = s
				hasStatus = true
			} else {
				return errResult("field 'status': expected string"), nil
			}
		}

		// Validate and apply non-status fields
		for fname, val := range fieldsArg {
			if fname == "status" {
				continue
			}
			fdef, ok := s.Fields[fname]
			if !ok {
				return errResult(fmt.Sprintf("field '%s' is not defined in schema", fname)), nil
			}
			if fdef.Generated {
				return errResult(fmt.Sprintf("field '%s' is generated and cannot be updated", fname)), nil
			}
			if err := schema.ValidateFieldValue(fdef, val); err != nil {
				return errResult(fmt.Sprintf("field '%s': %s", fname, err.Error())), nil
			}
			f.Fields[fname] = val
		}

		if hasStatus {
			// Delegate to move logic (checks transition + gate + moves file)
			if err := store.CheckAndMove(f, newStatus); err != nil {
				return errResult(err.Error()), nil
			}
		} else {
			if err := store.Write(f); err != nil {
				return errResult(fmt.Sprintf("writing issue: %v", err)), nil
			}
		}

		return jsonResult(issueToMap(f))
	})
}
