package tools

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
	"github.com/edgetools/issues-mcp/schema"
)

// Register adds all MCP tools to the server.
func Register(srv *server.MCPServer, store *issues.Store, s *schema.Schema) {
	RegisterGetFields(srv, s)
	RegisterListIssues(srv, store)
	RegisterGetIssue(srv, store)
	RegisterGetIssueBody(srv, store)
	RegisterGetWorkLog(srv, store)
	RegisterSearchIssues(srv, store)
	RegisterValidate(srv, store, s)
	RegisterCreateIssue(srv, store, s)
	RegisterUpdateFields(srv, store, s)
	RegisterUpdateBody(srv, store, s)
	RegisterAddComment(srv, store)
	RegisterMoveIssue(srv, store, s)
}
