# e57s02: Projects Table — Backend CRUD

**Story ID:** e57s02 | **Epic:** e57 — Project Scoping Backend | **BCPs:** 3 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** domain
**maturity:** 3 — Countable

## 2. Story Statement

**As a** BigBase administrator,
**I want** to create and manage projects within organizations,
**so that** I can group sites, deployments, and collections under logical project boundaries for multi-tenant isolation.

## 3. Context

This story creates the `projects` database table and full CRUD operations, implemented on the `Auth` component (which already owns `orgs`, `org_members`, `org_api_keys`). The `projects` table lives under `orgs` — each project belongs to exactly one organization.

The CRUD operations follow the exact same pattern as existing `orgs.go` (CreateOrg, GetOrgByID, ListOrgsByOwner, UpdateOrg, DeleteOrg) for consistency.

### Zoom-Out Summary
- **Auth component purpose:** Authentication, authorization, org/project/user management. Now adds: project CRUD.
- **Callers of auth:** Proxy (middleware), API (scoping), Sites, Deploy (future: e57s03, e57s04)
- **Contracts:** New `Project` struct and CRUD methods. Existing `Auth` struct API preserved.

## 4. Domain Model

```sql
CREATE TABLE projects (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id     INTEGER NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
  name       TEXT    NOT NULL,
  slug       TEXT    NOT NULL,
  created_at TEXT    NOT NULL,
  updated_at TEXT    NOT NULL,
  UNIQUE(org_id, slug)
);
```

```go
type Project struct {
  ID        int64  `json:"id"`
  OrgID     int64  `json:"org_id"`
  Name      string `json:"name"`
  Slug      string `json:"slug"`
  CreatedAt string `json:"created_at"`
  UpdatedAt string `json:"updated_at"`
}
```

## 5. Contract / Interface

```go
// In components/auth/projects.go

func (a *Auth) CreateProject(ctx context.Context, orgID int64, name, slug string) (*Project, error)
func (a *Auth) GetProjectByID(ctx context.Context, id int64) (*Project, error)
func (a *Auth) ListProjectsByOrg(ctx context.Context, orgID int64) ([]Project, error)
func (a *Auth) UpdateProject(ctx context.Context, id int64, name, slug string) (*Project, error)
func (a *Auth) DeleteProject(ctx context.Context, id int64) error
```

API endpoints (added to `ProtectedHandler()`):
- `POST /api/orgs/{orgID}/projects` — create project
- `GET /api/orgs/{orgID}/projects` — list projects
- `GET /api/orgs/{orgID}/projects/{id}` — get project
- `PATCH /api/orgs/{orgID}/projects/{id}` — update project
- `DELETE /api/orgs/{orgID}/projects/{id}` — delete project

## 6. Implementation Strategy

Follow the exact pattern established in `components/auth/orgs.go`:
1. Migration in `Start()` using `a.db.Migrate()`
2. CRUD methods using `a.db.ExecContext` / `a.db.QueryRowContext` / `a.db.QueryContext`
3. HTTP handlers using `mux.HandleFunc` with Go 1.22+ path patterns
4. Authorization: only org owner can create/update/delete projects; org members can list/get

## 7. Data Flow

```
POST /api/orgs/1/projects {name:"MyApp", slug:"myapp"}
  → auth.handleCreateProject
    → Claims.OrgID must match URL orgID (owner check)
    → auth.CreateProject(ctx, 1, "MyApp", "myapp")
      → INSERT INTO projects (org_id, name, slug, created_at, updated_at) VALUES (?, ?, ?, ?, ?)
    → 201 {id: 1, org_id: 1, name: "MyApp", slug: "myapp", ...}
```

## 8. Error Handling

| Condition | HTTP Status | Body |
|-----------|-------------|------|
| Slug already taken in org | 409 | `{"error": "slug already exists"}` |
| Org not found | 404 | `{"error": "organization not found"}` |
| Not org owner | 403 | `{"error": "forbidden"}` |
| Project not found | 404 | `{"error": "project not found"}` |
| Auth missing | 401 | `{"error": "authorization required"}` |

## 9. Testing Strategy

- **Unit:** `TestCreateProject`, `TestCreateProjectDuplicateSlug`, `TestGetProjectByID`, `TestGetProjectByIDNotFound`, `TestListProjectsByOrg`, `TestUpdateProject`, `TestDeleteProject`
- **Integration:** `TestCreateProjectEndpoint`, `TestListProjectsEndpoint`, `TestCrossOrgProjectAccess` (org 1 cannot see org 2's projects)
- Follow existing test patterns from `components/auth/orgs.go` tests

## 10. Migration / Rollback

```sql
-- Up:
CREATE TABLE IF NOT EXISTS projects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(org_id, slug)
);

-- Down:
DROP TABLE IF EXISTS projects;
```

Migration runs in `Auth.Start()` before any handler is registered. Idempotent via `CREATE TABLE IF NOT EXISTS`.

## 11. Documentation

Update `tech-stack.md` Component Catalog — Auth row: `orgs, projects, invites, API keys`

## 12. Dependencies

- **Depends on:** e57s01 (kernel project ID context helpers — used for authorization checks)
- **Depended on by:** e57s03 (DB isolation needs projects table), e57s04 (auth namespacing needs projects)

## 13. Observability

- Auth component logs: `"project created"` / `"project deleted"` with `project_id` and `org_id` key=value pairs
- Error logs for duplicate slugs and not-found cases

## 14. Security

- Authorization: only org owner (user with `OwnerID == claims.UserID`) can create/update/delete projects
- Slug validation: alphanumeric + hyphens only, 1-64 characters
- No cross-org project access — all queries scoped by `org_id`

## 15. Performance

- `UNIQUE(org_id, slug)` is indexed — slug lookups are O(log n)
- List query scoped by `org_id` only
- No N+1 queries — no child relations yet

## 16. Alternatives Considered

| Option | Decision |
|--------|----------|
| Projects in separate component | Rejected — follows orgs pattern; auth already owns org/user management |
| Soft-delete with `deleted_at` | Rejected for v1 — simpler to hard-delete; add later if needed |
| UUID primary key | Rejected — follows existing INTEGER AUTOINCREMENT pattern (orgs, users) |

## 17. Acceptance Criteria

```gherkin
Scenario: Create a project in an org
  Given an authenticated user who owns org 1
  When POST /api/orgs/1/projects with {"name": "My App", "slug": "my-app"}
  Then response is 201 with project ID, name, slug, org_id

Scenario: Duplicate slug rejected
  Given a project with slug "my-app" exists in org 1
  When POST /api/orgs/1/projects with {"name": "Other", "slug": "my-app"}
  Then response is 409 Conflict

Scenario: Cross-org slug allowed
  Given a project with slug "my-app" exists in org 1
  When POST /api/orgs/2/projects with {"name": "Other", "slug": "my-app"}
  Then response is 201 (slug is unique per org, not globally)

Scenario: Non-owner cannot create
  Given a user who is NOT the owner of org 1
  When POST /api/orgs/1/projects with {"name": "X", "slug": "x"}
  Then response is 403 Forbidden

Scenario: List projects returns org-only projects
  Given org 1 has 2 projects and org 2 has 1 project
  When GET /api/orgs/1/projects as org 1 owner
  Then response contains exactly 2 projects

Scenario: Delete project
  Given a project exists in org 1
  When DELETE /api/orgs/1/projects/1 as org 1 owner
  Then response is 200 and subsequent GET returns 404
```

## 18. Out of Scope

- Project-level API keys (deferred to e61 Secrets)
- Project members/invites (e57s04)
- Project → site/deployment assignment (e57s03, e57s04)
- Project settings/env vars (future)
- Default project auto-creation on org create (e57s04)

## 19. Risks

- **Risk:** Migration `UNIQUE(org_id, slug)` constraint could fail if projects table already has duplicates (not possible on first run, but migration re-runs could be tricky).
- **Mitigation:** `CREATE TABLE IF NOT EXISTS` is idempotent. Migration only runs once. Manual cleanup would be needed if schema changes.

## 20. Verification Script

1. `go test ./components/auth/ -run TestProject -v -count=1` — all project CRUD tests pass
2. `go test ./... -count=1` — full suite passes, no regressions
3. Start server: `go run . serve --port 9999 --db :memory:`
4. Create org: `curl -s -X POST http://localhost:9999/api/auth/register -d '{"email":"a@b.com","password":"test123"}' | jq .`
5. Create project: `curl -s -X POST http://localhost:9999/api/orgs/1/projects -H "Authorization: Bearer <TOKEN>" -d '{"name":"My App","slug":"my-app"}' | jq .`
6. List projects: `curl -s http://localhost:9999/api/orgs/1/projects -H "Authorization: Bearer <TOKEN>" | jq .data`
7. Verify slug uniqueness: second create with same slug returns 409
