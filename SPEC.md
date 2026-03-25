# issues-mcp: Reusable Issue Management MCP Server

## Overview

Go MCP server using `mark3labs/mcp-go`, stdio transport. Single binary, no runtime
dependencies. Schema-driven issue management over markdown files with YAML frontmatter.

Designed for reuse across projects. Each project provides its own `schema.json` that
defines fields, valid values, status transitions, workflow gates, and content locks.
The MCP enforces the schema deterministically so agents never need to interpret rules
by judgment.

Companion to `docs-mcp` (read-only markdown). Shares the same architectural decisions:
Go, `mark3labs/mcp-go`, stdio, single binary, user-global registration.

---

## CLI Interface

### MCP Mode (normal operation)

```
issues-mcp --issues-dir ./issues --schema ./issues/schema.json
```

Starts the MCP server over stdio. Both flags required.

- `--issues-dir`: Root directory containing status subdirectories (`backlog/`, `active/`,
  `done/`, etc.) and the `templates/` directory
- `--schema`: Path to the JSON schema file (can live inside or outside `--issues-dir`)

### Validate Mode (deterministic CLI, no MCP protocol)

```
issues-mcp validate --issues-dir ./issues --schema ./issues/schema.json
```

Runs schema validation across all issues and exits.

- **Exit code 0**: All issues valid
- **Exit code 1**: Validation errors found
- **Stderr**: JSON array of errors (same shape as the `validate` tool output)
- **Stdout**: Nothing (clean for piping)

This is what hooks and `prek` call. No agent in the loop, no MCP protocol overhead.
The validate CLI and the `validate` MCP tool share the same validation code internally.

---

## Directory Layout (per project)

```
issues/
├── schema.json           # field definitions, transitions, gates, locks
├── templates/
│   └── issue.md          # template with all fields at defaults
├── backlog/
│   ├── COMBAT-AGGRO-001.md
│   └── ENTITY-002.md
├── active/
│   └── COMBAT-AGGRO-002.md
└── done/
    └── ENTITY-001.md
```

Status subdirectories are derived from the schema's `statuses` list. The MCP creates
missing directories on startup rather than failing.

### Access Control

The MCP is the **only write interface** to the `issues/` directory. Agent profiles
must deny direct file writes to `issues/` paths. This is enforced through Claude Code
settings profiles (write denies per agent type), not by the MCP itself. The MCP
assumes it is the sole writer and enforces structural rules (transitions, gates, locks)
on that basis.

Without this access control, agents could bypass the MCP by editing issue files
directly, stomping the work log separator, violating gates, or breaking frontmatter
structure.

---

## Issue File Format

An issue file has three sections: YAML frontmatter, the body (problem statement,
hypothesis, notes), and an optional work log. The work log is separated from the
body by a marker and is append-only through the `add_comment` tool.

```markdown
---
id: COMBAT-AGGRO-001
title: "Basic aggro generation from damage"
status: active
area: Combat/Aggro
depends_on: []
source: vault/design/mechanics/aggro.md
spec_approved: true
---

## Problem Statement

When a player character deals damage to a mob, that mob should prioritize
attacking the damage dealer based on cumulative threat generated.

## Hypothesis

We believe a numeric threat table per mob, incremented by damage dealt,
will produce EQ-style aggro behavior where tanks must out-threat DPS.

## Notes

Reference EQ mechanics: 1 damage = 1 threat baseline. Taunt abilities
multiply threat. Healing threat is a separate issue (COMBAT-AGGRO-002).

<!-- work-log -->

### Work Log

**2026-03-24T14:32:00Z | impl-agent**
Implemented basic aggro table with damage-based threat generation.
Tests passing for scenarios 1-3. Scenario 4 (threat decay) deferred
to next session.

**2026-03-24T16:45:00Z | review-semantic**
Review passed. Test coverage matches BDD scenarios. No vacuous assertions.
```

The `<!-- work-log -->` HTML comment is the separator. Everything above it is the
body (editable via `update_body`, subject to locks). Everything below it is the work
log (append-only via `add_comment`, never locked). If no separator exists,
`add_comment` creates one.

The separator is an implementation detail of how the MCP parses files. Its integrity
is guaranteed by the access control constraint: agents cannot write to issue files
directly. All writes go through MCP tools, which always preserve the separator.

---

## ID Generation

IDs are auto-generated from the `area` field using a deterministic derivation rule.
There is no prefix mapping in the schema. Both `id` and `area` are MCP-intrinsic
fields (not declared in the project schema). The rule is:

1. Take the `area` string (e.g., `"Combat/Aggro"`)
2. Split on `/`
3. Uppercase each segment
4. Join with `-`
5. Append `-` and the next sequential number (zero-padded to 3 digits)

Examples:
- `area: "Combat/Aggro"` produces `COMBAT-AGGRO-001`, `COMBAT-AGGRO-002`, ...
- `area: "Combat/CrowdControl"` produces `COMBAT-CROWDCONTROL-001`, ...
- `area: "Entity"` produces `ENTITY-001`, `ENTITY-002`, ...

To find the next number, the MCP scans all status directories for files matching
`{PREFIX}-*.md`, parses the numeric suffix from each, finds the max, and increments.
If no existing issues match the prefix, the first issue gets `001`.

This means agents can invent new areas freely without anyone updating a config file.
The prefix derivation is mechanical and collision-free across distinct areas.

---

## Example Schema File (`schema.json`)

The following is an **example** `schema.json` file for the game project.

Every field in the
`fields` block is project-defined. Different projects will have different fields
based on their workflow needs. The MCP does not hardcode any assumptions about
which project-defined fields exist or whether they are required.

```json
{
  "statuses": ["backlog", "active", "done"],

  "fields": {
    "title": {
      "type": "string",
      "required": true
    },
    "source": {
      "type": "string",
      "required": false
    },
    "spec_approved": {
      "type": "bool",
      "default": false
    }
  },

  "transitions": {
    "backlog": ["active"],
    "active":  ["done", "backlog"],
    "done":    []
  },

  "gates": {
    "active": {
      "requires": {
        "spec_approved": true
      }
    }
  },

  "locks": {
    "body": {
      "locked_in": ["active", "done"]
    }
  }
}
```

### Schema Sections

**`statuses`**: Ordered list of valid statuses. Each status gets a subdirectory under
`--issues-dir`. This is the source of truth for directory creation.

**`fields`**: Project-defined frontmatter fields, their types, whether they are
required, their default values, and any constraints. Supported types: `string`,
`bool`, `enum`, `list`, `int`.

Four fields are **implicit** and must NOT appear in the schema: `id` (always string,
auto-generated from area), `area` (always string, always required), `status`
(always enum, values sourced from the `statuses` array, default is the first entry),
and `depends_on` (always a list of issue ID strings, default empty). These are
hardwired into the MCP's mechanics. The MCP rejects any schema that declares them
in `fields`.

`depends_on` is validated structurally: every ID in the list must reference an
existing issue. The `validate` tool also checks for circular dependency chains.
These checks run on both the MCP `validate` tool and the CLI `validate` subcommand.

All fields in the `fields` block are fully project-defined. The MCP does not
hardcode any assumptions about which project-defined fields exist, whether they
are required, or what their defaults are. That is entirely determined by whatever
the project puts in its `schema.json`.

**`transitions`**: Which statuses an issue can move to from each status. The MCP
rejects any transition not listed here. Not exposed to agents through `get_fields`.

**`gates`**: Field conditions that must be true before an issue can enter a status.
Evaluated during `move_issue` and during `update_fields` if the update includes a
`status` change. If a gate condition is not met, the MCP returns an error describing
exactly which field failed and what value it needs. Not exposed to agents through
`get_fields`.

**`locks`**: Content sections that become read-only in certain statuses. Currently
supports `body` as the only lockable section. When `update_body` is called on an
issue whose status is in the `locked_in` list, the MCP rejects the update with an
error explaining that the issue must be moved back to an unlocked status first.
Not exposed to agents through `get_fields`. The work log (`add_comment`) is never
locked regardless of status.

---

## Tools

### Read Tools

#### `get_fields`

Returns the fields an agent can interact with, their types, whether they are
writable, and valid values for enum types. This is a filtered view combining
the MCP's implicit fields (`id`, `area`, `status`, `depends_on`) with the project's
schema-defined fields. It deliberately excludes transitions, gates, locks, validation
rules, and other server-side enforcement details.

**Input:** none

**Output:**
```json
{
  "fields": {
    "title":         { "type": "string",  "writable": true },
    "area":          { "type": "string",  "writable": true },
    "depends_on":    { "type": "list",    "writable": true,
                       "note": "list of issue IDs, validated for existence" },
    "source":        { "type": "string",  "writable": true },
    "spec_approved": { "type": "bool",    "writable": true },
    "status":        { "type": "enum",    "writable": false,
                       "values": ["backlog", "active", "done"],
                       "note": "use move_issue to change status" },
    "id":            { "type": "string",  "writable": false,
                       "note": "auto-generated from area" }
  }
}
```

`id`, `area`, `status`, and `depends_on` always appear in this output regardless of
the project schema. They are MCP-intrinsic fields. All other fields come from the
schema.

Agents call this once per session to learn what fields exist and which they can
write to. If they attempt something invalid, the write tool returns a descriptive
error. The agent never sees the enforcement rules themselves.

---

#### `list_issues`

Lists all issues with their frontmatter (no body, no work log). Supports filtering.

**Input:**
```json
{
  "status": "active",
  "area": "Combat"
}
```

Both fields optional. When omitted, returns all issues across all statuses.
`area` matches as a prefix, so `"Combat"` matches `"Combat/Aggro"`,
`"Combat/CrowdControl"`, etc.

**Output:**
```json
{
  "issues": [
    {
      "id": "COMBAT-AGGRO-002",
      "title": "Healing generates threat",
      "status": "active",
      "area": "Combat/Aggro",
      "depends_on": ["COMBAT-AGGRO-001"],
      "source": "vault/design/mechanics/aggro.md",
      "spec_approved": true,
      "path": "active/COMBAT-AGGRO-002.md"
    }
  ]
}
```

The `path` field is relative to `--issues-dir`. Included so callers know where
the file lives without guessing.

---

#### `get_issue`

Returns a single issue by ID. Frontmatter only, never includes the body or
work log.

**Input:**
```json
{
  "id": "COMBAT-AGGRO-002"
}
```

**Output:**
```json
{
  "id": "COMBAT-AGGRO-002",
  "title": "Healing generates threat",
  "status": "active",
  "area": "Combat/Aggro",
  "depends_on": ["COMBAT-AGGRO-001"],
  "source": "vault/design/mechanics/aggro.md",
  "spec_approved": true,
  "path": "active/COMBAT-AGGRO-002.md"
}
```

---

#### `get_issue_body`

Returns only the markdown body of an issue (everything between the frontmatter
and the work log separator). Does not include frontmatter or the work log.

**Input:**
```json
{
  "id": "COMBAT-AGGRO-002"
}
```

**Output:**
```json
{
  "id": "COMBAT-AGGRO-002",
  "body": "## Problem Statement\n\nWhen a player character deals damage..."
}
```

If the issue has no body content (just frontmatter), `body` is an empty string.

---

#### `get_work_log`

Returns only the work log entries for an issue. Each entry is a structured object,
not raw markdown.

**Input:**
```json
{
  "id": "COMBAT-AGGRO-001"
}
```

**Output:**
```json
{
  "id": "COMBAT-AGGRO-001",
  "entries": [
    {
      "timestamp": "2026-03-24T14:32:00Z",
      "author": "impl-agent",
      "content": "Implemented basic aggro table with damage-based threat generation.\nTests passing for scenarios 1-3. Scenario 4 (threat decay) deferred\nto next session."
    },
    {
      "timestamp": "2026-03-24T16:45:00Z",
      "author": "review-semantic",
      "content": "Review passed. Test coverage matches BDD scenarios. No vacuous assertions."
    }
  ]
}
```

If no work log exists, `entries` is an empty array.

---

#### `search_issues`

Case-insensitive keyword search across frontmatter, body, and work log text.
Returns matching issues with surrounding context snippets.

**Input:**
```json
{
  "query": "aggro healing threat",
  "status": "active"
}
```

`status` is optional. When omitted, searches all issues across all directories.
Multiple keywords are matched individually (same behavior as `docs-mcp` search).

**Output:**
```json
{
  "results": [
    {
      "id": "COMBAT-AGGRO-002",
      "title": "Healing generates threat",
      "status": "active",
      "area": "Combat/Aggro",
      "path": "active/COMBAT-AGGRO-002.md",
      "matches": [
        {
          "line": 8,
          "context": "Healing spells generate threat on the aggro table proportional to effective healing done."
        },
        {
          "line": 14,
          "context": "Overhealing should generate reduced threat compared to effective healing."
        }
      ]
    }
  ]
}
```

Each result includes the issue's frontmatter summary (no full body) plus an array
of match contexts showing the line number and surrounding text for each hit.

---

#### `validate`

Validates one issue or all issues against the schema. Returns structured pass/fail.

**Input:**
```json
{
  "id": "COMBAT-AGGRO-002"
}
```

Or validate everything:
```json
{
  "all": true
}
```

**Output:**
```json
{
  "valid": false,
  "errors": [
    {
      "id": "COMBAT-AGGRO-002",
      "field": "depends_on",
      "error": "references unknown issue ID: COMBAT-AGGRO-099"
    },
    {
      "id": "ENTITY-003",
      "field": "status",
      "error": "frontmatter says 'active' but file is in backlog/ directory"
    }
  ]
}
```

Validation checks:
- All required fields present
- Field values match their type and constraints (enum values, patterns)
- `depends_on` references point to existing issue IDs
- `depends_on` chains contain no circular dependencies
- Frontmatter `status` matches the subdirectory the file lives in
- Gate conditions are consistent (e.g., issue in `active/` has `spec_approved: true`)
- No duplicate IDs across directories
- ID matches the expected prefix for the issue's `area` field

---

### Write Tools

#### `create_issue`

Creates a new issue file. The MCP auto-generates the `id` from the `area` field
using the derivation rule documented above.

**Input:**
```json
{
  "title": "Mez spell freezes target for duration",
  "area": "Combat/CrowdControl",
  "source": "vault/design/mechanics/crowd-control.md",
  "body": "## Problem Statement\n\nMez (mesmerize) is a core CC ability..."
}
```

Fields not provided use their schema defaults (`status: "backlog"`,
`spec_approved: false`, `depends_on: []`). The `body` field is optional; when
omitted the issue is created with an empty body.

**Behavior:**
1. Derive the ID prefix from `area` using the standard rule.
2. Scan all status directories for files matching `{PREFIX}-*.md`.
3. Parse the numeric suffix from each, find the max, increment by one.
4. Validate all provided fields against the schema.
5. Write the file to `backlog/{ID}.md` (new issues always start in `backlog`).
6. Return the created issue with its generated `id` and `path`.

**Output:**
```json
{
  "id": "COMBAT-CROWDCONTROL-001",
  "path": "backlog/COMBAT-CROWDCONTROL-001.md",
  "created": true
}
```

---

#### `update_fields`

Updates one or more frontmatter fields on an existing issue.

**Input:**
```json
{
  "id": "COMBAT-AGGRO-001",
  "fields": {
    "spec_approved": true,
    "title": "Basic aggro generation from damage (revised)"
  }
}
```

**Behavior:**
1. Find the issue file by scanning all status directories for `{ID}.md`.
2. Validate each field name exists in the schema.
3. Validate each value against the field's type and constraints.
4. If `status` is included in `fields`, delegate to `move_issue` logic
   (transition check + gate check + file move). This keeps the atomic
   guarantee that status and filesystem always agree.
5. Write updated frontmatter. Preserve body and work log unchanged.
6. Return the updated issue frontmatter.

**Rejected updates return errors, not silent failures:**
```json
{
  "error": "field 'priority' is not defined in schema"
}
```

**Cannot update generated fields:**
```json
{
  "error": "field 'id' is generated and cannot be updated"
}
```

---

#### `update_body`

Replaces the body content of an issue. Only affects the content between the
frontmatter and the work log separator. The work log is never modified by
this tool.

**Subject to locks.** If the issue's current status is in the schema's
`locks.body.locked_in` list, the update is rejected.

**Input:**
```json
{
  "id": "COMBAT-AGGRO-001",
  "body": "## Problem Statement\n\n(revised problem statement here...)\n\n## Hypothesis\n\n(revised hypothesis...)"
}
```

**Behavior:**
1. Find the issue file.
2. Check the `locks.body.locked_in` list against the issue's current status.
   If locked, reject with error.
3. Parse the file into three sections: frontmatter, body, work log.
4. Replace the body section with the provided content.
5. Preserve frontmatter and work log exactly as they are.
6. Write the file back.

**Lock rejection example:**
```json
{
  "error": "cannot update body of COMBAT-AGGRO-001: body is locked while status is 'active' (move to 'backlog' to edit)"
}
```

**Output (success):**
```json
{
  "id": "COMBAT-AGGRO-001",
  "updated": true
}
```

---

#### `add_comment`

Appends a timestamped entry to the issue's work log. The work log is append-only;
existing entries are never modified. **Not subject to locks.** Comments can be added
in any status because the work log records progress during active work.

**Input:**
```json
{
  "id": "COMBAT-AGGRO-001",
  "author": "impl-agent",
  "content": "Implemented basic aggro table with damage-based threat generation.\nTests passing for scenarios 1-3."
}
```

**Behavior:**
1. Find the issue file.
2. If no work log separator (`<!-- work-log -->`) exists, append one after the body,
   followed by a `### Work Log` heading.
3. Append a new entry with the current UTC timestamp, the author, and the content.
4. Write the file back.

The timestamp is generated server-side (not provided by the caller) to ensure
chronological accuracy.

**Output:**
```json
{
  "id": "COMBAT-AGGRO-001",
  "timestamp": "2026-03-24T14:32:00Z",
  "added": true
}
```

---

#### `move_issue`

Moves an issue to a new status. Updates frontmatter and moves the file atomically.

**Input:**
```json
{
  "id": "COMBAT-AGGRO-001",
  "status": "active"
}
```

**Behavior:**
1. Find the issue file.
2. Check `transitions`: is `current_status -> target_status` allowed?
3. Check `gates`: does the issue meet all field conditions for `target_status`?
4. Update `status` in frontmatter.
5. Move the file from `{old_status}/{ID}.md` to `{new_status}/{ID}.md`.
6. Return the result.

**Gate failure example:**
```json
{
  "error": "cannot move COMBAT-AGGRO-001 to 'active': gate requires spec_approved=true but current value is false"
}
```

**Transition failure example:**
```json
{
  "error": "cannot move COMBAT-AGGRO-001 from 'done' to 'active': transition not allowed (done allows: [])"
}
```

---

## Error Handling Contract

Every tool returns structured JSON. Success responses vary by tool (documented above).
All error responses use the same shape:

```json
{
  "error": "human-readable description of what went wrong and why"
}
```

Errors are descriptive enough that the calling agent can decide what to do next
without needing to re-read the schema. For example, a gate failure names the
specific field, its current value, and its required value. A lock rejection names
the locked section, the current status, and which status to move to for editing.

The MCP never partially applies a write operation. If validation fails, nothing
is written. If a file move fails, the frontmatter is not updated.

---

## Startup Behavior

On startup, the MCP:

1. Reads and parses `--schema`. Exits with error if missing or malformed.
2. Validates schema structure:
   - `statuses` array must exist and contain at least one entry.
   - `id`, `area`, `status`, and `depends_on` must NOT be declared in `fields`
     (they are MCP-intrinsic). If present, exit with error to prevent confusion.
   - `transitions` keys must all be valid statuses.
   - `gates` keys must all be valid statuses.
   - `locks.body.locked_in` values must all be valid statuses.
3. Creates any missing status subdirectories under `--issues-dir`.
4. Runs a full validation pass across all existing issues. Logs warnings for
   any inconsistencies (status/directory mismatch, missing required fields)
   but does NOT refuse to start. This allows the MCP to help fix problems
   rather than being blocked by them.

---

## What the MCP Does NOT Do

- **No ordering or prioritization.** The MCP doesn't rank issues or decide what
  to work on next. Agents use `list_issues` with filters and make their own
  decisions (or follow instructions from the orchestrator).
- **No notifications or triggers.** The MCP is a passive tool. It doesn't watch
  for changes or trigger pipelines. The `just` recipes and (later) the file
  watcher handle orchestration.
- **No auth or multi-user.** Single user, local filesystem. This is a developer
  tool, not a team collaboration platform.
- **No delete.** Deletion is a human action. If you want to remove an issue,
  delete the file directly. The MCP will stop returning it on the next call.
- **No work log editing.** The work log is append-only through `add_comment`.
  If an entry needs correction, a new comment is added clarifying the previous one.
  This preserves the audit trail.
- **No direct file access enforcement.** The MCP assumes it is the sole writer
  to the `issues/` directory. Preventing agents from bypassing the MCP is the
  responsibility of Claude Code settings profiles (write denies on issue paths).

---

## Registration

User-global Claude settings (same as `docs-mcp`):

```json
{
  "mcpServers": {
    "issues": {
      "command": "/path/to/issues-mcp",
      "args": [
        "--issues-dir", "./issues",
        "--schema", "./issues/schema.json"
      ]
    }
  }
}
```

The `--issues-dir` path is relative to the project root. Since the MCP is registered
globally but invoked per-project, the working directory at invocation time determines
which project's issues are managed.

---

## Repo Structure (`issues-mcp` repository)

```
issues-mcp/
├── main.go                 # CLI flags, server setup, stdio serve, validate subcommand
├── go.mod
├── go.sum
├── schema/
│   ├── types.go            # Schema struct definitions
│   ├── parser.go           # Reads + validates schema.json
│   └── validator.go        # Validates issues against schema (shared by MCP + CLI)
├── issues/
│   ├── store.go            # Filesystem operations (scan, read, write, move)
│   ├── id.go               # ID generation (area derivation, max scan, increment)
│   ├── frontmatter.go      # YAML frontmatter parsing + serialization
│   ├── body.go             # Body + work log parsing and manipulation
│   └── search.go           # Keyword search with context extraction
├── tools/
│   ├── get_fields.go
│   ├── list_issues.go
│   ├── get_issue.go
│   ├── get_issue_body.go
│   ├── get_work_log.go
│   ├── search_issues.go
│   ├── validate.go
│   ├── create_issue.go
│   ├── update_fields.go
│   ├── update_body.go
│   ├── add_comment.go
│   └── move_issue.go
├── testdata/
│   ├── valid_schema.json
│   ├── issues/             # fixture issues for tests
│   └── ...
└── README.md
```

---

## Tool Summary

| Tool             | Read/Write | Lockable | Purpose                                          |
|------------------|------------|----------|--------------------------------------------------|
| `get_fields`     | Read       | —        | What fields exist and which are writable          |
| `list_issues`    | Read       | —        | Board view: frontmatter for all/filtered issues   |
| `get_issue`      | Read       | —        | Single issue frontmatter                          |
| `get_issue_body` | Read       | —        | Single issue body (no frontmatter, no work log)   |
| `get_work_log`   | Read       | —        | Single issue work log as structured entries        |
| `search_issues`  | Read       | —        | Keyword search across all issue content           |
| `validate`       | Read       | —        | Schema validation (also available as CLI command)  |
| `create_issue`   | Write      | No       | New issue with auto-generated ID                   |
| `update_fields`  | Write      | No       | Modify frontmatter fields (validated)              |
| `update_body`    | Write      | Yes      | Replace body content (preserves work log)          |
| `add_comment`    | Write      | No       | Append to work log (never locked)                  |
| `move_issue`     | Write      | No       | Change status with transition + gate enforcement   |

---

## Design Decisions

**Why lock the body in active/done statuses?**
The body is the contract the implementation agent works from. If it changes while
an agent is mid-session, the agent is working against a spec that no longer matches
the file. Locking the body when active creates intentional friction: you must move
the issue back to backlog to revise the spec, which signals to the pipeline that
work has been paused. This is the same philosophy as requiring `spec_approved`
before moving to active. You can't quietly change the terms while work is in
progress.

**Why is the work log never locked?**
The work log records what happened during each phase. Locking it during active work
would defeat its purpose. `add_comment` is the one write tool that always works
regardless of status.

**Why `get_fields` instead of exposing the full schema?**
If agents see the transitions, gates, locks, and validation rules, they may try to
reason about or work around them. The MCP enforces those rules server-side. Agents
only need to know what fields exist and which are writable. If they try something
invalid, they get a descriptive error. The enforcement rules are invisible to the
agent.

**Why separate `get_issue`, `get_issue_body`, and `get_work_log`?**
Each returns a distinct section of the issue file. Combining them behind flags
invites agents to always request everything "to be safe," wasting tokens. Separate
tools make the cost visible and the intent explicit. An agent must make a deliberate
call to read the body. If agents are calling `get_issue_body` on every issue in a
loop, that pattern is visible in logs and fixable in agent instructions.

**Why `update_body` and `add_comment` as separate tools?**
The body (problem statement, hypothesis, notes) is collaborative and editable
during design. The work log is an append-only audit trail. Different access patterns,
different guarantees, different lock behavior. `update_body` replaces and is subject
to locks; `add_comment` appends and is never locked. The work log separator ensures
`update_body` can never accidentally erase history.

**Why server-generated timestamps on `add_comment`?**
Agents might provide inaccurate timestamps. The work log's chronological order
is more trustworthy when the server controls the clock.

**Why auto-derived prefixes instead of a mapping table?**
Design agents create new areas as part of their work. A manual prefix mapping
would require someone to update schema.json every time a new area appears, which
is a token-wasting failure mode when forgotten. The derivation rule (split on `/`,
uppercase, join with `-`) is deterministic and requires zero maintenance.

**Why a separate MCP instead of `just` recipes with shell scripts?**
Shell scripts parsing YAML frontmatter is fragile. The MCP gives agents a typed,
validated interface. It also gives deterministic enforcement of transitions and
gates, which is the core requirement for the agentic pipeline.

**Why JSON schema instead of YAML?**
The schema is machine-read configuration, not human-edited prose. JSON avoids
YAML parsing ambiguities and lets us reuse Go's `encoding/json` directly.

**Why are `id`, `area`, `status`, and `depends_on` implicit instead of schema-declared?**
These fields are hardwired into the MCP's mechanics: `id` is derived from `area`,
`area` drives prefix filtering in `list_issues`, `status` determines directory
placement, transition enforcement, gate evaluation, and lock evaluation, and
`depends_on` requires cross-issue reference validation that a generic list field
cannot express. These are the fields that any issue management system needs
regardless of project. `status` values come from the `statuses` array, which is
already a top-level schema key. Keeping all four out of `fields` prevents projects
from accidentally redefining them and prevents agents from thinking they are
configurable.

**Why startup validation is warn-only?**
If the MCP refuses to start when issues are invalid, you can't use the MCP to
fix them. Warn on startup, let agents call `validate` explicitly, fix issues
through `update_fields`.

**Why a CLI validate mode in addition to the MCP tool?**
The MCP `validate` tool is for agents. The CLI `validate` subcommand is for hooks
and `prek`, where we need a deterministic exit code with no agent in the loop.
Same validation code, two interfaces.

**Why is access control external to the MCP?**
The MCP is a passive tool that enforces structural rules on its own writes. It
cannot prevent other processes from writing to the filesystem. Agent write denies
are configured in Claude Code settings profiles. This separates concerns: the MCP
owns schema enforcement, the IDE/agent platform owns file access control.
