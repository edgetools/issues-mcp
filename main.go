package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/edgetools/issues-mcp/issues"
	"github.com/edgetools/issues-mcp/schema"
	"github.com/edgetools/issues-mcp/tools"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "validate" {
		runValidateCLI()
		return
	}

	issuesDir := flag.String("issues-dir", "", "root issues directory")
	schemaPath := flag.String("schema", "", "path to schema.json")
	flag.Parse()

	if *issuesDir == "" || *schemaPath == "" {
		fmt.Fprintln(os.Stderr, "usage: issues-mcp --issues-dir DIR --schema FILE")
		os.Exit(1)
	}

	s, err := schema.Load(*schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading schema: %v\n", err)
		os.Exit(1)
	}

	store, err := issues.NewStore(*issuesDir, s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing store: %v\n", err)
		os.Exit(1)
	}

	// Startup validation — warn-only so the MCP can help fix problems.
	records, err := store.AllRecords()
	if err == nil && len(records) > 0 {
		errs := schema.ValidateAll(s, records)
		if len(errs) > 0 {
			log.Printf("startup: %d validation warning(s)", len(errs))
			for _, e := range errs {
				log.Printf("  [%s] %s: %s", e.ID, e.Field, e.Error)
			}
		}
	}

	srv := server.NewMCPServer("issues-mcp", "1.0.0",
		server.WithToolCapabilities(false),
	)
	tools.Register(srv, store, s)

	if err := server.ServeStdio(srv); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

// runValidateCLI implements the `issues-mcp validate` subcommand.
// Exits 0 if all issues are valid, 1 if errors are found.
// Errors are written as a JSON array to stderr; stdout is silent.
func runValidateCLI() {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	issuesDir := fs.String("issues-dir", "", "root issues directory")
	schemaPath := fs.String("schema", "", "path to schema.json")
	fs.Parse(os.Args[2:])

	if *issuesDir == "" || *schemaPath == "" {
		fmt.Fprintln(os.Stderr, "usage: issues-mcp validate --issues-dir DIR --schema FILE")
		os.Exit(1)
	}

	s, err := schema.Load(*schemaPath)
	if err != nil {
		writeJSONErr(schema.ValidationError{Error: err.Error()})
		os.Exit(1)
	}

	store, err := issues.NewStore(*issuesDir, s)
	if err != nil {
		writeJSONErr(schema.ValidationError{Error: err.Error()})
		os.Exit(1)
	}

	records, err := store.AllRecords()
	if err != nil {
		writeJSONErr(schema.ValidationError{Error: err.Error()})
		os.Exit(1)
	}

	errs := schema.ValidateAll(s, records)
	if len(errs) > 0 {
		data, _ := json.Marshal(errs)
		fmt.Fprintln(os.Stderr, string(data))
		os.Exit(1)
	}
	os.Exit(0)
}

func writeJSONErr(e schema.ValidationError) {
	data, _ := json.Marshal([]schema.ValidationError{e})
	fmt.Fprintln(os.Stderr, string(data))
}
