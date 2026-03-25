# issues-mcp

Schema-driven issue management MCP server. Issues are markdown files with YAML frontmatter, organized into status subdirectories. The server enforces field types, status transitions, gate conditions, and body locks defined in a per-project `schema.json`.

## Quick Start

### 1. Build

```sh
just build   # output: build/issues-mcp
just install # installs to $GOPATH/bin
```

### 2. Create a schema

```json
{
  "statuses": ["backlog", "active", "done"],
  "fields": {
    "title":         { "type": "string", "required": true },
    "spec_approved": { "type": "bool", "default": false }
  },
  "transitions": {
    "backlog": ["active"],
    "active":  ["done", "backlog"],
    "done":    []
  },
  "gates": {
    "active": { "requires": { "spec_approved": true } }
  },
  "locks": {
    "body": { "locked_in": ["active", "done"] }
  }
}
```

Save it as `issues/schema.json` in your project.

### 3. Register with Claude Code

In your Claude Code MCP settings (`~/.claude/settings.json`):

```json
{
  "mcpServers": {
    "issues": {
      "command": "/path/to/issues-mcp",
      "args": ["--issues-dir", "./issues", "--schema", "./issues/schema.json"]
    }
  }
}
```

The `--issues-dir` path is relative to the project root where Claude Code is invoked.

## CLI Usage

**MCP mode** (normal — called by Claude Code):
```sh
issues-mcp --issues-dir ./issues --schema ./issues/schema.json
```

**Validate mode** (for hooks/CI, no MCP protocol):
```sh
issues-mcp validate --issues-dir ./issues --schema ./issues/schema.json
# exit 0 = all valid
# exit 1 = errors (JSON array written to stderr)
```

## Issue File Format

```markdown
---
id: COMBAT-AGGRO-001
title: "Basic aggro from damage"
status: backlog
area: Combat/Aggro
depends_on: []
spec_approved: false
---

## Problem Statement

...

<!-- work-log -->

### Work Log

**2026-03-24T14:32:00Z | impl-agent**
Work done here.
```

## ID Generation

IDs are derived from the `area` field: split on `/`, uppercase each segment, join with `-`, append zero-padded sequence number.

- `Combat/Aggro` → `COMBAT-AGGRO-001`, `COMBAT-AGGRO-002`, …
- `Entity` → `ENTITY-001`, …

## Available Tools

| Tool | Purpose |
|------|---------|
| `get_fields` | Schema fields and which are writable |
| `list_issues` | All issues with optional status/area filter |
| `get_issue` | Single issue frontmatter |
| `get_issue_body` | Issue body only |
| `get_work_log` | Structured work log entries |
| `search_issues` | Keyword search across all content |
| `validate` | Schema validation (single issue or all) |
| `create_issue` | New issue with auto-generated ID |
| `update_fields` | Modify frontmatter fields |
| `update_body` | Replace body (respects locks) |
| `add_comment` | Append to work log (never locked) |
| `move_issue` | Change status with transition/gate enforcement |

## Schema Reference

- **`statuses`** — ordered list of valid statuses; each gets a subdirectory
- **`fields`** — project-defined field declarations with `type`, `required`, `values`, `default`
- **`transitions`** — allowed status moves per source status
- **`gates`** — field conditions required to enter a status
- **`locks`** — content sections locked in certain statuses (`body` is the only lockable section)

Field types: `string`, `bool`, `enum`, `list`, `int`

### Field tiers

Fields come from two sources:

| Tier | Fields | Declared in schema? |
|------|--------|---------------------|
| MCP-intrinsic | `id`, `area`, `status`, `depends_on` | No — hardwired; declaring any of them in `fields` is a startup error |
| Project-defined | everything else | Yes — add whatever fields the project needs |

- `id` — auto-generated from `area`; never writable
- `area` — set at creation; cannot be changed
- `status` — driven by the `statuses` array; change via `move_issue` only
- `depends_on` — list of issue IDs; writable through `update_fields`, validated for existence
