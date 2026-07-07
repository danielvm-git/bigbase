# e67s01: MCP create_repo Tool

**Story ID:** e67s01 | **Epic:** e67 — MCP Provisioning Tools | **BCPs:** 2 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** domain
**maturity:** 3 — Countable

## 2. Story Statement

**As an** AI coding agent using the BigBase MCP server,
**I want** to register a git repository with BigBase by providing its name and optional description,
**so that** I can immediately proceed to site creation and deployment without opening the BigBase admin UI.

## 3. Context

The Git component (`components/git/git.go`) already exposes `POST /api/git/repos` which creates a bare repo directory and inserts a row into `git_repos`. The MCP component (`components/mcp/mcp.go`) already uses the interface-wiring pattern (`DeployTrigger`, `DBer`) to couple to other components.

This story follows the proven pattern: define a `GitCreator` interface in `mcp.go`, wire `git.Component` as its implementation in `main.go`, and register a `create_repo` MCP tool in `NewMCPServer()`.

### Zoom-Out Summary

- **Module purpose:** `components/mcp` is a Model Context Protocol server that teaches AI agents how to use BigBase.
- **Callers:** AI clients (Claude Desktop, Cursor, VS Code) connecting over SSE or stdio.
- **Contracts preserved:** Existing MCP tools untouched. `Options` gains optional `GitCreator` field; nil = tool responds with "git creator not configured."

## 4. Domain Model

No new tables. Uses existing `git_repos` table (created by `git.Start()`).

```
Repo {
    ID:            string   // hex-encoded 16 bytes, generated server-side
    Name:          string   // unique, user-provided
    OwnerID:       int64    // defaults to 0 (system) until multi-tenant
    Private:       bool     // defaults to true
    DefaultBranch: string   // defaults to "main"
    Description:   string   // optional
    CreatedAt:     string   // RFC3339
}
```

## 5. Contract / Interface

```go
// In mcp/mcp.go — new interface

// GitCreator registers a git repository with BigBase.
type GitCreator interface {
    CreateRepo(ctx context.Context, name, description string, private bool) (id, fullName string, err error)
}

// Options extended:
type Options struct {
    // ... existing fields ...
    GitCreator GitCreator // optional; nil disables create_repo
}

// Component extended:
type Component struct {
    // ... existing fields ...
    gitCreator GitCreator
}
```

MCP tool signature:
```
create_repo(name, description?, private?)
  → { repo_id: "abc...", name: "my-project" }
```

## 6. Implementation Strategy

1. Define `GitCreator` interface (two return values: `id string`, `name string`, `error`).
2. Add `gitCreator` field to `Component`; populate in `New()` from `opts.GitCreator`.
3. Implement `CreateRepo` on `*git.Git` — calls its existing `createRepo` internal logic.
4. Register `create_repo` tool in `NewMCPServer()` — validates name is non-empty, calls `c.gitCreator.CreateRepo()`, returns repo_id and name.
5. Wire `git` component as `GitCreator` in `main.go`:

```go
mcpComp := mcp.New(mcp.Options{
    GitCreator: g,  // *git.Git
    // ...
})
```

6. Write tests in `mcp_test.go` using a mock `GitCreator`.

## 7. Data Flow

```
AI agent: call tool "create_repo" with {"name": "big-library"}
  → mcp.NewMCPServer handler closure
      → c.gitCreator.CreateRepo(ctx, "big-library", "", true)
          → git.Git.createRepo → INSERT INTO git_repos, git init --bare
      → return { repo_id: "abc123...", name: "big-library" }
```

## 8. Error Handling

| Condition | Tool response |
|-----------|---------------|
| `GitCreator` not configured (nil) | "Git tools require a Git component. Start BigBase with the Git component enabled." |
| Name is empty | "name is required" |
| Name already exists (UNIQUE constraint) | "A repo named 'foo' already exists" |
| `CreateRepo` fails | "Failed to create repo: <err>" |

## 9. Testing Strategy

- **Unit — success:** Mock `GitCreator` returns repo ID + name; tool response contains both.
- **Unit — nil creator:** Tool returns "not configured" text.
- **Unit — empty name:** Tool returns validation error.
- **Unit — duplicate name:** Mock returns error with "already exists"; tool propagates it.

## 10. Migration / Rollback

No schema changes. Rollback = remove `GitCreator` interface, field, tool registration, and `main.go` wiring.

## 11. Documentation

Update `specs/tech-architecture/tech-stack.md` MCP section to list `create_repo`.

## 12. Dependencies

- `components/git` already in the codebase — no new imports.
- `github.com/modelcontextprotocol/go-sdk` already in `go.mod` [OK].
- No new external packages.

## 13. Observability

```go
c.logger.Info("mcp tool", "tool", "create_repo", "name", name, "repo_id", id)
```

## 14. Security

**Security level:** none

- MCP server is internal-only (default port 3900, not proxied publicly).
- Repos default to private. The tool accepts a `private` flag.

## 15. Acceptance Criteria

```gherkin
Scenario: create_repo creates a git repo and returns its ID
  Given a running BigBase instance with MCP and Git components
  When an AI agent calls create_repo with {"name": "my-project"}
  Then the response contains a repo_id and name "my-project"
  And the repo is visible via list_repos

Scenario: create_repo validates required name
  When an AI agent calls create_repo without a name
  Then the response says "name is required"

Scenario: create_repo returns helpful error on duplicate
  Given an existing repo named "my-project"
  When an AI agent calls create_repo with {"name": "my-project"}
  Then the response says "already exists"

Scenario: create_repo returns "not configured" without Git component
  Given MCP is started without a GitCreator
  When an AI agent calls create_repo
  Then the response says "Git tools require a Git component"
```

## 16. Out of Scope

- Importing from a remote GitHub URL (currently only creates an empty bare repo). That's a separate `git_import` tool for a future story.
- Setting SSH deploy keys.
- Multi-tenant owner_id mapping (e57 concern).
