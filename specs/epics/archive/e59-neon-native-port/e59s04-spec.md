# e59s04: MCP Database Migration Tools

**Story ID:** e59s04 | **Epic:** e59 — Native Feature Port | **BCPs:** 3 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** domain
**maturity:** 3 — Countable

## 2. Story Statement

**As an** AI coding agent using the BigBase MCP server,
**I want** tools to inspect and manage database migration state,
**so that** I can guide users through schema changes, diagnose migration drift, and apply or roll back migrations without leaving the MCP tool conversation.

## 3. Context

The BigBase MCP component (`components/mcp/mcp.go`) already exposes deploy workflow tools (`list_repos`, `deploy_site`, `get_deploy_status`, `get_deploy_logs`) via the `modelcontextprotocol/go-sdk` using `mcpsdk.AddTool`. The pattern is established: add a tool struct, register a handler closure, wire dependencies through `Options`.

The `db` component has a fully-implemented versioned `Migrator` (`components/db/migrator.go`) with `Up()`, `Down(n int)`, and `Status()`. It reads from a `schema_migrations` table.

This story adds three new MCP tools:
- `list_migrations` — shows all known migrations and their applied status.
- `run_migrations` — applies all pending migrations (calls `Migrator.Up()`).
- `rollback_migration` — rolls back the last N applied migrations (calls `Migrator.Down(n)`).

These tools require a `Migrator` dependency wired into `mcp.Options`. The `Migrator` interface in the MCP package is kept minimal (only the three methods BigBase's Migrator already has) to avoid coupling to the concrete type.

### Zoom-Out Summary
- **Module purpose:** `components/mcp` is a Model Context Protocol server that teaches AI agents how to use BigBase — deploy sites, discover services, and now manage database schema.
- **Callers:** AI clients (Claude Desktop, IDE extensions) connecting over SSE or stdio; no other BigBase component calls into `mcp` directly.
- **Contracts preserved:** Existing MCP tools (`ping`, `list_services`, `list_repos`, etc.) untouched. `Options` struct gains an optional `Migrator` field; nil = tools respond with "migrator not configured." `Component` struct gains a `migrator` field.

## 4. Domain Model

No new tables. `schema_migrations` table already exists (created by `db.Migrator.ensureTable()`).

```
MigrationStatus {
    Version   string
    Applied   bool
    AppliedAt string   // RFC3339 or "" if not applied
}
```

## 5. Contract / Interface

```go
// components/mcp/mcp.go additions

// MigratorI is the minimal interface the MCP component needs.
type MigratorI interface {
    Status() ([]db.MigrationRecord, error)
    Up() error
    Down(n int) error
}

// Options extended:
type Options struct {
    // ... existing fields ...
    Migrator MigratorI // optional; nil disables migration tools
}

// Component extended:
type Component struct {
    // ... existing fields ...
    migrator MigratorI
}
```

MCP tool names and descriptions:
```
list_migrations    — "List all database migrations and their applied status."
run_migrations     — "Apply all pending database migrations. Returns the number applied."
rollback_migration — "Roll back the last N applied migrations (default N=1). Use with caution."
```

Wire in `main.go`:
```go
mcpComp := mcp.New(mcp.Options{
    // ... existing ...
    Migrator: db.NewMigrator(dbComp.DB(), allMigrations),
})
```

`allMigrations` is a `[]db.Migration` slice that must be assembled in `main.go` from all component migrations. This story defines the pattern; the actual slice is populated here.

## 6. Implementation Strategy

1. Add `MigratorI` interface and `migrator` field to `Component` in `mcp.go`.
2. Populate `migrator` in `New()` from `opts.Migrator`.
3. Register the three new tools in `NewMCPServer()` using `mcpsdk.AddTool`, following the exact pattern of existing tools.
4. `list_migrations` handler: call `c.migrator.Status()`, format as markdown table (version, applied ✓/✗, applied_at).
5. `run_migrations` handler: call `c.migrator.Up()`, report count of newly applied migrations (infer from Status before/after).
6. `rollback_migration` handler: parse optional `n` arg (default 1), call `c.migrator.Down(n)`, confirm.
7. In `main.go`: define `allMigrations` as the union of all component migration definitions (auth, api, etc.) and wire `db.NewMigrator(dbComp.DB(), allMigrations)` into `mcp.Options`.
8. Write tests in `mcp_test.go`.

## 7. Data Flow

```
AI agent: call tool "list_migrations"
  → mcp.Component.NewMCPServer (handler closure)
      → c.migrator.Status()
      → format markdown table
      → return TextContent

AI agent: call tool "run_migrations"
  → handler closure
      → c.migrator.Up()
      → return "N migrations applied" or "all up to date"

AI agent: call tool "rollback_migration" with {"n": 2}
  → handler closure
      → c.migrator.Down(2)
      → return "Rolled back 2 migrations"
```

## 8. Error Handling

| Condition | Tool response |
|-----------|---------------|
| `Migrator` not configured (`nil`) | "Migration tools require a Migrator. Start BigBase with a configured db to enable." |
| `Status()` fails | "Error fetching migration status: <err>" |
| `Up()` fails | "Migration failed: <err>" |
| `Down(n)` fails | "Rollback failed: <err>" |
| Invalid `n` arg | "n must be a positive integer" |

Errors are returned as MCP `TextContent` (not Go errors) — consistent with the existing pattern in `mcp.go`.

## 9. Testing Strategy

- **Unit — `list_migrations` with mock migrator:** Status returns 2 applied, 1 pending → formatted table with ✓/✗ markers.
- **Unit — `run_migrations`:** Up() succeeds → "applied" message; Up() errors → error text result.
- **Unit — `rollback_migration`:** Down(1) succeeds → confirmation; Down(0) → "n must be a positive integer"; nil migrator → "not configured" message.
- All tests use a mock `MigratorI` implementation (defined in `mcp_test.go`).

## 10. Migration / Rollback

No schema changes in this story. Rollback = remove the three tool registrations and the `Migrator`/`migrator` fields from Options/Component.

## 11. Documentation

Update `specs/tech-architecture/tech-stack.md` MCP section to list the three new migration tools. Update MCP knowledge base file if it exists under `components/mcp/knowledge/`.

## 12. Dependencies

- `db.MigrationRecord` and `db.NewMigrator` from `components/db` — already in the codebase.
- `github.com/modelcontextprotocol/go-sdk` already in `go.mod` [OK].
- No new external packages.
- Requires that `main.go` assembles `allMigrations`. This story defines the slice; it is seeded with any currently-inline migration SQL from `auth.Start()`, `api.Start()`, etc.

## 13. Observability

- `logger.Info("mcp migration tool", "tool", toolName, "result", summary)` on each tool invocation.

## 14. Security

**Security level:** none

- MCP server is internal-only (default port 3900, not proxied publicly).
- `run_migrations` and `rollback_migration` are potentially destructive; document that the MCP server should not be exposed without authentication in production.

## 15. Performance

`Status()` queries `schema_migrations` — a small table, sub-millisecond. `Up()` and `Down()` are admin operations, not hot path.

## 16. Alternatives Considered

- **Import `db.Migrator` directly (concrete type):** creates an import cycle (`mcp` → `db` is already allowed since `mcp.Options.DB` is `DBer`). Concrete type import is fine, but the interface is cleaner for testing.
- **Expose migration HTTP endpoints instead of MCP tools:** useful but separate concern. MCP tools are the e59 scope.

## 17. Acceptance Criteria

```gherkin
Scenario: list_migrations returns status table
  Given a BigBase instance with 3 migrations (2 applied, 1 pending)
  When an AI agent calls list_migrations via MCP
  Then the response is a markdown table with version, applied ✓/✗, and applied_at

Scenario: run_migrations applies pending migrations
  Given 1 pending migration
  When an AI agent calls run_migrations
  Then the response confirms "1 migration applied"
  And the migration is recorded in schema_migrations

Scenario: rollback_migration rolls back last migration
  Given 2 applied migrations
  When an AI agent calls rollback_migration with {"n": 1}
  Then the response confirms rollback
  And schema_migrations has 1 row

Scenario: tools respond gracefully when Migrator not configured
  Given mcp.Options.Migrator is nil
  When an AI agent calls list_migrations
  Then the response text says "Migration tools require a Migrator"
```

## 18. Out of Scope

- HTTP API endpoints for migration management.
- Per-component migration isolation.
- Migration file discovery from the filesystem.
- Automatic migration on startup (that is already each component's `Start()` responsibility).

## 19. Risks

- Assembling `allMigrations` in `main.go` requires extracting inline SQL from each component's `Start()` method into exported slices — moderate refactor. If this proves too invasive, scope `run_migrations` to a no-op ("use the CLI or restart BigBase to apply migrations") and only implement `list_migrations` (read-only from `schema_migrations`).

## 20. Verification Script

```bash
go build ./components/mcp/ && go test ./components/mcp/ -run TestMigrationTool -v -count=1
```
