# e57s04: Auth Namespacing + Site→Project Backfill

**Story ID:** e57s04 | **Epic:** e57 — Project Scoping Backend | **BCPs:** 7 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** domain
**maturity:** 3 — Countable

## 2. Story Statement

**As a** BigBase platform,
**I want** JWT tokens to carry a `project_id` claim, auth middleware to inject it into context, and all existing sites to be backfilled to their org's default project,
**so that** every authenticated request is project-namespaced and legacy data is migrated without downtime.

## 3. Context

This is the capstone story of e57. It wires together all previous stories:
- e57s01 provided kernel project ID context helpers
- e57s02 created the `projects` table with CRUD
- e57s03 added `project_id` columns to sites and deployments

This story:
1. Adds `ProjectID` to the JWT `Claims` struct
2. Updates `createJWT` to accept and emit `project_id`
3. Updates auth middleware to extract `project_id` from JWT/API key into context
4. Creates a `project_members` table for user → project membership
5. Auto-creates a default project when an org is created
6. Runs the full Site → Project backfill (assigns all existing sites/deployments to default projects)
7. Updates all auth endpoints (login, register, OAuth) to include project_id in issued tokens

### Zoom-Out Summary

| Module | Purpose | Callers | Contracts Changed |
|--------|---------|---------|-------------------|
| Auth/jwt.go | JWT creation and verification | Auth middleware, login/register handlers | `Claims` gains `ProjectID` field; `createJWT` signature gains `projectID` param |
| Auth/auth.go | Registration, login, OAuth, middleware | Proxy | All token-issuing paths pass projectID; middleware injects `kernel.WithProjectID()` |
| Auth/middleware | Context injection | All protected routes | Adds `ctx = kernel.WithProjectID(ctx, claims.ProjectID)` |
| Auth/orgs.go | Org management | Admin UI, API | `CreateOrg` auto-creates default project |
| Auth/passwords | Backfill flow | Startup | `BackfillSitesToProjects` migrates existing data |

## 4. Domain Model

```go
// JWT Claims — UPDATED
type Claims struct {
    UserID    int64  `json:"user_id"`
    Email     string `json:"email"`
    Role      string `json:"role"`
    OrgID     int64  `json:"org_id"`
    ProjectID int64  `json:"project_id"` // NEW — zero means no project selected
    jwt.RegisteredClaims
}
```

```sql
-- NEW table: project membership
CREATE TABLE IF NOT EXISTS project_members (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT    NOT NULL DEFAULT 'member',
    created_at TEXT    NOT NULL,
    UNIQUE(project_id, user_id)
);
```

```go
// Project member
type ProjectMember struct {
    ID        int64  `json:"id"`
    ProjectID int64  `json:"project_id"`
    UserID    int64  `json:"user_id"`
    Role      string `json:"role"`
    CreatedAt string `json:"created_at"`
}
```

## 5. Contract / Interface

### Updated JWT Functions

```go
// createJWT signature UPDATED:
func createJWT(userID int64, email, role string, orgID, projectID int64, secret []byte) (string, error)

// verifyJWT unchanged — returns Claims which now includes ProjectID
func verifyJWT(tokenStr string, secret []byte) (*Claims, error)
```

### Updated Middleware

```go
// Auth.Middleware — UPDATED to inject project_id:
claims, err := verifyJWT(token, a.secret)
// ...
ctx = context.WithValue(ctx, ctxOrgID, claims.OrgID)
ctx = kernel.WithProjectID(ctx, claims.ProjectID)  // NEW
```

### Project Membership Methods

```go
func (a *Auth) AddProjectMember(ctx context.Context, projectID, userID int64, role string) (*ProjectMember, error)
func (a *Auth) ListProjectMembers(ctx context.Context, projectID int64) ([]ProjectMember, error)
func (a *Auth) RemoveProjectMember(ctx context.Context, projectID, userID int64) error
func (a *Auth) GetDefaultProject(ctx context.Context, orgID int64) (*Project, error)
```

### Backfill Hook

```go
// Called in Auth.Start() after projects table is ready
func (a *Auth) BackfillSitesToProjects(ctx context.Context) error
```

## 6. Implementation Strategy

### Phase 1: JWT Claims Update
1. Add `ProjectID int64 \`json:"project_id"\`` to `Claims` struct
2. Update `createJWT` to accept `projectID` parameter
3. Update all callers of `createJWT` (login, register, OAuth, anonymous, phone) to pass `projectID`
4. Set `projectID` to the user's default project (or 0 if none)

### Phase 2: Middleware Injection
1. In `Auth.Middleware`, after JWT verification: add `ctx = kernel.WithProjectID(ctx, claims.ProjectID)`
2. For API key auth path: resolve project ID from API key metadata (or default project of the org)
3. Fallback: if `claims.ProjectID == 0`, get the org's default project and set it

### Phase 3: Project Membership
1. Migration for `project_members` table
2. CRUD methods: `AddProjectMember`, `ListProjectMembers`, `RemoveProjectMember`
3. API endpoints: `POST/GET/DELETE /api/projects/{id}/members`

### Phase 4: Default Project on Org Create
1. In `CreateOrg`: after inserting org, auto-create a default project (name="Default", slug="default")
2. In registration flow: after creating personal org, set user's default project

### Phase 5: Site → Project Backfill
1. For each org that has sites but no projects assigned:
   a. Get or create default project
   b. UPDATE sites SET project_id = ? WHERE git_repo_id IN (SELECT git_repo_id FROM sites WHERE project_id IS NULL)
   c. UPDATE deployments SET project_id = ? WHERE site_id IN (SELECT id FROM sites WHERE project_id = ?)

## 7. Data Flow

```
User registers → createJWT(..., projectID=defaultProjectID) → token with project_id claim
                                                                  ↓
Request with Bearer token → auth.Middleware → verifyJWT → Claims.ProjectID
                                                                  ↓
                                            kernel.WithProjectID(ctx, projectID)
                                                                  ↓
Sites handler → kernel.ProjectIDFromContext(ctx) → scoped query
Deploy handler → kernel.ProjectIDFromContext(ctx) → scoped query
API handler → kernel.ProjectIDFromContext(ctx) → scoped query
```

## 8. Error Handling

| Condition | HTTP Status | Body |
|-----------|-------------|------|
| Token has no project_id (and no default project) | 403 | `{"error": "no project selected"}` |
| Project membership duplicate | 409 | `{"error": "user already a member"}` |
| Project not found | 404 | `{"error": "project not found"}` |
| Not project owner | 403 | `{"error": "forbidden"}` |
| Backfill DB error | (logged, component starts) | N/A — best-effort at startup |

## 9. Testing Strategy

- **JWT Claims test:** `TestClaimsWithProjectID` — verify token carries project_id
- **Middleware test:** `TestMiddlewareInjectsProjectID` — verify context has project_id after auth
- **Backward compat test:** `TestTokenWithoutProjectID` — old tokens (no project_id) still work, projectID defaults to 0
- **Project member test:** `TestAddListRemoveProjectMember`
- **Default project test:** `TestCreateOrgCreatesDefaultProject`
- **Backfill test:** `TestBackfillSitesToProjects` — 3 orgs, 5 sites, verify all assigned
- **Integration test:** `TestFullAuthFlowWithProject` — register → org → default project → site creation scoped

## 10. Migration / Rollback

```sql
-- project_members table:
CREATE TABLE IF NOT EXISTS project_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    created_at TEXT NOT NULL,
    UNIQUE(project_id, user_id)
);

-- Backfill SQL (run once in Auth.Start):
-- For each org without a default project:
INSERT INTO projects (org_id, name, slug, created_at, updated_at)
SELECT id, 'Default', 'default', datetime('now'), datetime('now')
FROM orgs WHERE id NOT IN (SELECT org_id FROM projects WHERE slug = 'default');

-- Assign sites to default projects:
UPDATE sites SET project_id = (
    SELECT p.id FROM projects p
    JOIN orgs o ON o.id = p.org_id
    WHERE p.slug = 'default'
    -- Match site's org via git_repo
    LIMIT 1
) WHERE project_id IS NULL;
```

Rollback: Not provided (forward-only migration). Existing `project_id` columns would need to be NULLed.

## 11. Documentation

Update:
- `tech-stack.md` — JWT Claims structure, middleware flow, backfill process
- Auth component godoc — `createJWT` signature change
- `specs/adr/` — new ADR for project scoping decision

## 12. Dependencies

- **Depends on:** e57s01 (kernel project ID helpers), e57s02 (projects table), e57s03 (project_id columns on sites/deployments)
- **Depended on by:** e56 (OTP session hardening needs project scope), e58 (UI), e59 (Neon port), e61 (Secrets), e62 (CSP), e63 (Dashboard), e64 (Schema Designer), e65 (Preview envs), e66 (Multi-user)

## 13. Observability

- Token creation logs: `"jwt created" user_id=X org_id=Y project_id=Z`
- Backfill logs: `"backfill assigned sites" org_id=X project_id=Y site_count=N`
- Middleware debug: `"auth context set" project_id=X` (at debug level)

## 14. Security

- `project_id` in JWT is signed (HS256) — cannot be tampered
- API key path: project_id resolved from API key → org → default project (no user-supplied project_id)
- Project membership is org-scoped (only org members can be project members)
- Backfill runs with system privileges (no user context needed)

## 15. Performance

- JWT Claims addition: negligible size increase (~10 bytes)
- `project_members` table: `UNIQUE(project_id, user_id)` indexed
- Backfill: O(orgs + sites + deployments) — runs once at startup, bounded by 30s timeout
- Middleware: one additional context.WithValue call — O(1)

## 16. Alternatives Considered

| Option | Decision |
|--------|----------|
| project_id as query param instead of JWT claim | Rejected — insecure, user-controllable |
| Separate project-scoped tokens | Rejected — complexity; single token with multiple scopes is simpler |
| project_id in cookie instead of JWT | Rejected — breaks API key auth, SPA flows |
| Require project_id on every request | Rejected for v1 — backward compat needed for legacy clients; unscoped access still works |

## 17. Acceptance Criteria

```gherkin
Scenario: JWT carries project_id
  Given a user registers with email "a@b.com"
  When the JWT token is issued
  Then the decoded token contains a "project_id" claim with the default project's ID

Scenario: Middleware injects project_id into context
  Given a valid JWT with project_id=5
  When a request passes through auth middleware
  Then kernel.ProjectIDFromContext(ctx) returns (5, true)

Scenario: API key auth also resolves project_id
  Given a valid API key
  When a request uses X-API-Key header
  Then the context contains the org's default project_id

Scenario: Default project created on org creation
  Given a new org is created
  When checking projects for that org
  Then a project with slug "default" exists

Scenario: Site → project backfill
  Given an existing org with 3 sites and no project assignments
  When the backfill runs at startup
  Then all 3 sites have a project_id assigned to the default project

Scenario: Legacy token (no project_id) still works
  Given a JWT token issued before this migration (no project_id claim)
  When the token is verified
  Then ProjectID defaults to 0 and middleware sets it to the org's default project

Scenario: Project member CRUD
  Given a project in org 1
  When an org member is added to the project
  Then they appear in the project members list

Scenario: Cross-org member rejection
  Given a user in org 1
  When they try to add a user from org 2 to org 1's project
  Then the operation is rejected with 403
```

## 18. Out of Scope

- Multiple project selection in a single session (user picks one project at a time)
- Project-scoped API keys (e61)
- Project settings/env vars (future)
- Project transfer between orgs
- Role-based access within projects (beyond owner/member)
- Project delete cascading to sites/deployments (hard-delete not implemented)

## 19. Risks

- **Risk:** `createJWT` signature change breaks all callers
- **Mitigation:** Compile-time safety — Go won't compile with mismatched arguments. All callers updated in same commit.
- **Risk:** Backfill could fail on large production DBs
- **Mitigation:** Best-effort, logged, non-blocking. Manual backfill script available.
- **Risk:** Old tokens (no project_id) become invalid
- **Mitigation:** `ProjectID` defaults to `0` (zero value) in Claims. Middleware handles `projectID == 0` by resolving default project. Backward compatible.
- **Risk:** Legacy zero-value could leak unbackfilled data.
- **Mitigation:** Callers receiving `(0, false)` from `ProjectIDFromContext` must run queries **unscoped** — no `WHERE project_id` clause — not `WHERE project_id = 0`.
- **Risk:** Backfill could be interrupted, causing duplicate attempts on restart.
- **Mitigation:** `BackfillSitesToProjects` is idempotent — it calls `GetProjectBySlug(orgID, "default")` before `CreateProject`; a UNIQUE conflict is treated as success.
- **Risk:** This story adds ~120 lines to `auth.go` (project_members, 5 token-issuing paths, middleware).
- **Mitigation:** Accepted debt per issue #44 — no forcing function. Reverts cleanly if a Verifier interface is extracted later.

## 20. Verification Script

1. `go test ./components/auth/ -run TestClaimsWithProjectID -v -count=1` — JWT claims carry project_id
2. `go test ./components/auth/ -run TestMiddlewareInjectsProjectID -v -count=1` — context has project_id
3. `go test ./components/auth/ -run TestProjectMember -v -count=1` — membership CRUD works
4. `go test ./components/auth/ -run TestDefaultProject -v -count=1` — default project auto-created
5. `go test ./components/auth/ -run TestBackfill -v -count=1` — backfill completes successfully
6. `go test ./... -count=1` — full suite green
7. Manual smoke test:
   a. `go run . serve --port 9999 --db :memory:`
   b. Register: `curl -X POST localhost:9999/api/auth/register -d '{"email":"t@t.com","password":"test123"}'`
   c. Verify token has project_id: decode JWT payload (base66url), confirm `"project_id"` exists
   d. Create site, verify project_id is assigned
   e. List sites, verify scoped to default project
