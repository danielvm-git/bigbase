# e67s02: MCP create_site Tool

**Story ID:** e67s02 | **Epic:** e67 — MCP Provisioning Tools | **BCPs:** 2 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** domain
**maturity:** 3 — Countable

## 2. Story Statement

**As an** AI coding agent using the BigBase MCP server,
**I want** to provision a new site for a git repo and get back the critical `site_id`,
**so that** I can wire it into CI/CD secrets (`BIGBASE_SITE_ID`) without opening the admin UI.

The domain (e.g. `my-project.bigbase.click`) is NOT returned by `create_site` — it is a **deployment property** computed at deploy time via `deploymentURL()` in `components/deploy/urls.go`. The agent gets the URL from `deploy_site` or `get_deploy_status` after the first deploy.

## 3. Context

The Sites component (`components/sites/sites.go`) already exposes `POST /api/sites` which creates a site record in the `sites` table and optionally triggers an auto-deploy when a `DeployTrigger` is wired. The `site_id` is a critical piece of infrastructure — used by the deploy API, environment variables UI, and CI/CD pipeline scripts.

The domain logic lives in `components/deploy` (not in `components/sites`). `sites.createSite()` only writes to the `sites` table — it has no `PublicDomain` field and does not compute `my-project.bigbase.click`. This is architecturally correct: the domain is a deployment concern, not a site concern. The MCP tool reflects this by returning only `site_id`.

This story: define a `SiteCreator` interface, wire `sites.Sites`, register `create_site` tool.

### Zoom-Out Summary

Same as e67s01 — follows the proven MCP tool registration pattern.

## 4. Domain Model

No new tables. Uses existing `sites` table (created by `sites.Start()`). No `Domain` field — domains are deployment properties.

```
Site {
    ID:               string   // hex-encoded 16 bytes
    Name:             string   // user-provided, or defaults from repo name
    FullName:         string   // display name (repo name fallback)
    GitRepoID:        string   // FK to git_repos.id
    ProductionBranch: string   // default "main"
    RootPath:         string   // default "./"
}
```

## 5. Contract / Interface

```go
// In mcp/mcp.go — new interface

// SiteCreator provisions a new site on BigBase.
// Returns site_id and name (name is defaulted from repo when empty input).
type SiteCreator interface {
    CreateSite(ctx context.Context, gitRepoID, name, branch string) (id, name string, err error)
}

// Options extended:
type Options struct {
    // ... existing fields ...
    SiteCreator SiteCreator // optional; nil disables create_site
}

// Component extended:
type Component struct {
    // ... existing fields ...
    siteCreator SiteCreator
}
```

MCP tool signature:
```
create_site(git_repo_id, name?, branch?, auto_deploy?)
  → { site_id: "04c58b9...", name: "my-project" }
```

When `name` is omitted, it defaults to the repo name (queried from `git_repos`). The domain is NOT returned — use `deploy_site` or `get_deploy_status` to get the URL after the first deploy.

## 6. Implementation Strategy

1. Define `SiteCreator` interface (returns `id string`, `name string`, `error`).
2. Add `siteCreator` field to `Component`; populate in `New()` from `opts.SiteCreator`.
3. Implement `CreateSite` on `*sites.Sites` — calls existing `createSite` internal logic, validates `git_repo_id` exists, defaults `name` from repo name when omitted. Returns `id` and `name`. This is the single owner of name defaulting — the MCP tool is a passthrough.
4. Register `create_site` tool in `NewMCPServer()` — validates `git_repo_id`, passes through to `c.siteCreator.CreateSite()`, returns `site_id` + `name`. No name defaulting in the MCP layer — Sites is the single owner. If `auto_deploy=true` and `Deployer` is wired, triggers deployment and includes deploy ID + URL in response.
5. Wire `sites` component as `SiteCreator` in `main.go`.
6. Write tests in `mcp_test.go` using a mock `SiteCreator`.

## 7. Data Flow

```
AI agent: create_site({"git_repo_id": "abc123", "name": "my-project"})
  → handler validates git_repo_id non-empty
  → c.siteCreator.CreateSite(ctx, "abc123", "my-project", "main")
      → sites.Sites.createSite: validates repo exists, defaults name from repo if omitted, INSERT INTO sites
      → returns site_id "04c58b9..."
  → return { site_id: "04c58b9...", name: "my-project" }
  → (optional) if auto_deploy=true, calls c.deployer.Trigger() → deploy ID + URL in response
```

## 8. Error Handling

| Condition | Tool response |
|-----------|---------------|
| `SiteCreator` not configured (nil) | "Site tools require a Sites component." |
| `git_repo_id` is empty | "git_repo_id is required. Use list_repos to find available repositories." |
| Git repo not found | "Repository 'abc123' not found. Use list_repos to see available repositories." |
| `CreateSite` fails | "Failed to create site: <err>" |

## 9. Testing Strategy

- **Unit — success:** Mock returns site_id; tool response contains site_id + name.
- **Unit — nil creator:** Returns "not configured."
- **Unit — missing git_repo_id:** Returns validation error.
- **Unit — name defaulting:** Mock verifies that omitted name → repo name queried from git_repos.
- **Unit — with auto_deploy=true:** Verifies deployer.Trigger is called when both SiteCreator and Deployer are wired.

## 10. Migration / Rollback

No schema changes. Rollback = remove `SiteCreator` interface, field, tool registration, and wiring.

## 11. Documentation

Update `specs/tech-architecture/tech-stack.md` MCP section to list `create_site`.

## 12. Dependencies

- `components/sites` already in the codebase.
- `github.com/modelcontextprotocol/go-sdk` already in `go.mod` [OK].
- No new external packages.

## 13. Observability

```go
c.logger.Info("mcp tool", "tool", "create_site", "name", name, "repo_id", gitRepoID, "site_id", id)
```

## 14. Security

**Security level:** none — MCP server is internal-only.

## 15. Acceptance Criteria

```gherkin
Scenario: create_site provisions a site and returns site_id
  Given a git repo with id "repo-1" exists
  When an AI agent calls create_site with {"git_repo_id": "repo-1", "name": "my-app"}
  Then the response contains a site_id
  And the site is visible via the admin UI

Scenario: create_site defaults name from repo when omitted
  Given a git repo named "my-repo" exists
  When an AI agent calls create_site with {"git_repo_id": "repo-1"}
  Then the response contains name "my-repo"
  And a site_id is returned

Scenario: create_site validates git_repo_id
  When an AI agent calls create_site without git_repo_id
  Then the response says "git_repo_id is required"

Scenario: create_site returns error for unknown repo
  When an AI agent calls create_site with {"git_repo_id": "nonexistent"}
  Then the response says "not found"

Scenario: create_site triggers auto-deploy when requested
  Given a Deployer is wired
  When an AI agent calls create_site with {"git_repo_id": "repo-1", "auto_deploy": true}
  Then a deployment is triggered and the response includes deploy ID and URL
```

## 16. Out of Scope

- Custom domains (that's the existing `/api/sites/:id/domains` endpoint).
- Domain computation (lives in deploy, not sites).
- Environment variable provisioning during site creation.
- Site deletion (existing `delete_site` tool from a future story).
- Changing REST API to allow empty `name` on `POST /api/sites` (MCP-only defaulting via `CreateSite`).

## 17. Requirements

#### ADDED: MCP `create_site` tool provisions a site and returns `site_id`
Agents call `create_site(git_repo_id, name?, branch?, auto_deploy?)` and receive `{ site_id, name }`. Domain URL is intentionally omitted — agents use `deploy_site` / `get_deploy_status` after first deploy.

#### ADDED: `Sites.CreateSite` exported method with name defaulting
`CreateSite(ctx, gitRepoID, name, branch string) (id, name string, err error)` validates repo exists, defaults `name` from `git_repos.name` when empty, inserts into `sites`. Single owner of name defaulting — MCP layer is passthrough.

#### ADDED: Optional auto-deploy on site creation
When `auto_deploy=true` and `Deployer` is wired, tool triggers deployment and includes deploy ID + URL in response.

## 18. Risks

- **Current HTTP gap:** `createSite` HTTP handler today requires non-empty `name` (line 408). `CreateSite` adds defaulting for MCP; HTTP behavior unchanged unless explicitly MODIFIED later.
- **e57 blocker:** Sites table may gain `project_id` after e57 — provisioning tools need project context post-e57.

## 19. Verification Script

1. `go test ./components/mcp/ -run TestCreateSite -v -count=1` — MCP tool tests pass
2. `go test ./components/sites/ -v -count=1` — sites component tests pass
3. `go test ./... -count=1` — full suite green
4. Confirm `main.go` wires `SiteCreator: st` in `mcp.Options`
