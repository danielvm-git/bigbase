# e52s03: Database Isolation Per Project

**Story ID:** e52s03 | **Epic:** e52 — Project Scoping Backend | **BCPs:** 3 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** infra
**maturity:** 3 — Countable

## 2. Story Statement

**As a** BigBase platform,
**I want** sites and deployments to be scoped by project_id at the database layer,
**so that** querying sites or deployments for one project never leaks data from another project in the same organization.

## 3. Context

This story adds `project_id` columns to the `sites` and `deployments` tables and scopes all queries by the project ID from context. It also creates a default project for every existing organization and backfills all existing sites and deployments to belong to that default project.

The scoping follows the same pattern as existing `org_id` scoping in the API component (`components/api/api.go` — `WithOrgID` / `OrgIDFromContext`).

### Zoom-Out Summary

| Module | Purpose | Callers | Contracts Changed |
|--------|---------|---------|-------------------|
| Sites | Manage sites (CRUD, domains, env vars) | Admin UI, Deploy, Proxy | Sites table gains `project_id` column; all queries add `WHERE project_id = ?` when scoped |
| Deploy | Build, run, and manage deployments | Sites, Proxy, Admin UI | Deployments table gains `project_id` column; all queries add `WHERE project_id = ?` when scoped |
| Auth | User/org management (owns `projects` table) | Proxy middleware, API | No changes — already owns project CRUD from e52s02 |

## 4. Domain Model

```sql
-- sites table updated:
ALTER TABLE sites ADD COLUMN project_id INTEGER DEFAULT NULL;

-- deployments table updated:
ALTER TABLE deployments ADD COLUMN project_id INTEGER DEFAULT NULL;

-- Index for project-scoped queries:
CREATE INDEX IF NOT EXISTS idx_sites_project_id ON sites(project_id);
CREATE INDEX IF NOT EXISTS idx_deployments_project_id ON deployments(project_id);
```

```go
// Scoped queries in sites component:
projectID, ok := kernel.ProjectIDFromContext(ctx)
if ok {
    rows, err = s.db.QueryContext(ctx, "SELECT ... FROM sites WHERE project_id = ?", projectID)
}
```

## 5. Contract / Interface

No new exported APIs. Existing `Sites` and `Deploy` struct methods gain project-scoping internally:

- `sites.listSites()` — scoped by project_id from context
- `sites.getSite()` — scoped by project_id + site_id
- `sites.createSite()` — assigns project_id from context
- `deploy` queries — scoped by project_id from context
- `sites.listSitesFromRepos()` — scoped by project_id

Backfill contract:
- `auth.DefaultProjectForOrg(orgID) → *Project` — returns or creates the default project for an org
- `sites.BackfillProjectIDs(auth *Auth)` — assigns all sites/deployments to default projects

## 6. Implementation Strategy

1. **Migrations:** Add `project_id` columns to `sites` and `deployments` tables via `ALTER TABLE` (idempotent, wrapped in `ensureColumn` pattern)
2. **Backfill:** For each org, create a default project named "Default" with slug "default". Assign all existing sites and deployments in that org to the default project.
3. **Scoping:** Sites queries check `kernel.ProjectIDFromContext(ctx)` and add `WHERE project_id = ?` when present. Unscoped access (legacy, no project context) still works for backward compatibility.
4. **Deploy scoping:** Equivalent scoping in deploy's query methods.

## 7. Data Flow

```
Request → auth.Middleware → ProjectIDFromContext(ctx) → projectID
                                                          ↓
Sites.listSites(ctx) → "SELECT ... FROM sites WHERE project_id = ?" [projectID]
                                                          ↓
Deploy queries → "SELECT ... FROM deployments WHERE project_id = ?" [projectID]
```

Legacy path (no project ID in context):
```
Request (no project context) → Sites.listSites(ctx) → "SELECT ... FROM sites" (unscoped)
```

## 8. Error Handling

- Missing project_id in context → return all records (backward compatible; no project context = unscoped)
- Site not found in project → 404 with `{"error": "site not found"}`
- Cross-project deployment access → 404 (don't leak existence)

## 9. Testing Strategy

- **Migration test:** `TestProjectIDColumnExists` — verify `project_id` column exists in sites and deployments
- **Backfill test:** `TestBackfillProjectIDs` — create org, create sites, run backfill, verify all sites have project_id set
- **Scoping test:** `TestSitesScopedByProject` — create 2 projects, create sites in each, query scoped, verify isolation
- **Deploy scoping test:** `TestDeploymentsScopedByProject` — equivalent for deployments
- **Backward compat test:** `TestSitesWithoutProjectContext` — no project context returns all sites
- **Default project test:** `TestDefaultProjectCreated` — verify default project is created per org

## 10. Migration / Rollback

```sql
-- Up:
ALTER TABLE sites ADD COLUMN project_id INTEGER DEFAULT NULL;
ALTER TABLE deployments ADD COLUMN project_id INTEGER DEFAULT NULL;
CREATE INDEX IF NOT EXISTS idx_sites_project_id ON sites(project_id);
CREATE INDEX IF NOT EXISTS idx_deployments_project_id ON deployments(project_id);

-- Down (NOT provided — migrations are forward-only):
-- Data loss would occur on drop.
```

The `ensureColumn` pattern is used (idempotent ALTER TABLE wrapped in try/catch). Follows existing patterns from `deploy/ensureSiteIDColumn`, `deploy/ensurePassthroughPathsColumn`.

## 11. Documentation

Update `tech-stack.md` — add `project_id` to sites and deployments table schemas.

## 12. Dependencies

- **Depends on:** e52s01 (kernel project ID helpers), e52s02 (projects table exists)
- **Depended on by:** e52s04 (auth namespacing references project_id in queries)

## 13. Observability

- Backfill logs: `"backfill project" key=value` with org_id, project_id, affected_row_count
- Site create logs: include `project_id` key=value pair

## 14. Security

- Project-scoped queries prevent cross-project data leakage
- No authorization bypass — project_id is extracted from JWT/API key context (set by auth middleware)
- `project_id` column cannot be set via user input — always from context

## 15. Performance

- `idx_sites_project_id` and `idx_deployments_project_id` indexes ensure scoped queries are fast
- Backfill is O(orgs × sites/deployments) — runs once at startup in `Start()`, bounded by backfill timeout

## 16. Alternatives Considered

| Option | Decision |
|--------|----------|
| Per-project SQLite files | Rejected — breaks joins, complex backup, doesn't scale |
| Views with project_id filter | Rejected — SQLite views are read-only for complex queries |
| Row-level security (Postgres only) | Rejected — must work with SQLite for single-binary distribution |
| project_id ON DELETE CASCADE | Deferred — adds complexity; hard-delete of projects not yet implemented |

## 17. Acceptance Criteria

```gherkin
Scenario: Sites scoped by project
  Given project 1 has site A and project 2 has site B (in same org)
  When listing sites with project 1 context
  Then only site A is returned

Scenario: Deployments scoped by project
  Given project 1 has deployment X and project 2 has deployment Y
  When querying deployments with project 2 context
  Then only deployment Y is returned

Scenario: Backfill assigns default project
  Given an org with 3 sites and no project_id assigned
  When the backfill runs
  Then all 3 sites have project_id set to the org's default project

Scenario: Backward compatible unscoped access
  Given a request with no project context
  When listing sites
  Then all sites are returned (existing behavior preserved)

Scenario: Default project created per org
  Given an org with no projects
  When DefaultProjectForOrg() is called
  Then a project with slug "default" is created and returned
```

## 18. Out of Scope

- API collections/items project scoping (those use org_id from context already; adding project_id is future work)
- Functions project scoping
- Storage files project scoping
- Forge (issues/labels/kanban/wiki) project scoping
- UI for project-scoped views (e57)

## 19. Risks

- **Risk:** Backfill could fail if orgs table is empty or malformed
- **Mitigation:** Backfill runs best-effort, logs errors, doesn't block component start
- **Risk:** `ALTER TABLE ADD COLUMN` could fail on very large tables
- **Mitigation:** SQLite's ALTER TABLE is fast (schema change only, no data copy). Monitored via startup logs.

## 20. Verification Script

1. `go test ./components/sites/ -run TestProjectID -v -count=1` — sites scoping tests pass
2. `go test ./components/deploy/ -run TestProjectID -v -count=1` — deploy scoping tests pass
3. `go test ./... -count=1` — full suite green
4. Start server: `go run . serve --port 9999 --db :memory:`
5. Register user, create org, verify default project is auto-created (e52s04 will wire this)
6. Create site, verify project_id is assigned
7. List sites with project context, verify scoping
