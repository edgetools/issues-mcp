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
		mcp.WithDescription("Updates one or more frontmatter fields on an existing issue. Cannot update id, area, or status (use move_issue for status). depends_on is writable and validated for existence."),
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

		// Validate and apply fields
		for fname, val := range fieldsArg {
			switch fname {
			case "id":
				return errResult("field 'id' is auto-generated and cannot be updated"), nil
			case "area":
				return errResult("field 'area' cannot be changed after creation"), nil
			case "status":
				return errResult("field 'status' cannot be set directly (use move_issue)"), nil
			case "depends_on":
				deps, err := toDependsOnList(val)
				if err != nil {
					return errResult(fmt.Sprintf("field 'depends_on': %s", err.Error())), nil
				}
				if err := validateDependsOn(store, deps); err != nil {
					return errResult(err.Error()), nil
				}
				f.Fields["depends_on"] = deps
			default:
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
		}

		if err := store.Write(f); err != nil {
			return errResult(fmt.Sprintf("writing issue: %v", err)), nil
		}

		return jsonResult(issueToMap(f))
	})
}

// toDependsOnList converts an interface value to a []interface{} of strings.
func toDependsOnList(val interface{}) ([]interface{}, error) {
	switch v := val.(type) {
	case []interface{}:
		for i, item := range v {
			if _, ok := item.(string); !ok {
				return nil, fmt.Errorf("item %d: expected string, got %T", i, item)
			}
		}
		return v, nil
	case nil:
		return []interface{}{}, nil
	default:
		return nil, fmt.Errorf("expected list of strings, got %T", val)
	}
}

// validateDependsOn checks that every ID in deps references an existing issue.
func validateDependsOn(store *issues.Store, deps []interface{}) error {
	if len(deps) == 0 {
		return nil
	}
	allFiles, err := store.ListAll("", "")
	if err != nil {
		return fmt.Errorf("listing issues for depends_on validation: %v", err)
	}
	allIDs := make(map[string]bool, len(allFiles))
	for _, f := range allFiles {
		allIDs[f.ID] = true
	}
	for _, dep := range deps {
		depID := dep.(string) // safe after toDependsOnList
		if depID != "" && !allIDs[depID] {
			return fmt.Errorf("field 'depends_on': references unknown issue ID: %s", depID)
		}
	}
	return nil
}
