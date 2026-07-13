# e58s01: Project Scoping Admin UI

## Story ID: e58s01 | Epic: e58 | BCPs: 3 | Status: planned

## 1. Type
**feat** · ui · domain

## 2. Context
BigBase's admin UI (`ui/src/pages/`) has pages for users, sites, deployments, functions,
storage, messaging, and settings — but no page for managing **projects**. With e57
(Project Scoping Backend) adding a `projects` table, CRUD methods, and `project_id`
scoping on sites/deployments, the admin UI needs a **Projects page** to create, view,
edit, and delete projects within an organization.

The existing pattern for CRUD admin pages follows:
- A page component in `ui/src/pages/` (e.g., `UsersPage.tsx`)
- A hook in `ui/src/hooks/` (e.g., `useWorkspace.ts`) for data fetching — many are stubs
- Navigation entry in `ui/src/Layout.tsx` sidebar
- API routes on the Go backend (e.g., `/api/auth/projects`)
- Test file alongside the page (e.g., `UsersPage.test.tsx`)

## 3. Problem / Opportunity

| Problem | Impact |
|---------|--------|
| No admin UI to manage projects | Users can't create or organize projects after e57 backend ships |
| Navigation has no Projects link | Users can't discover the feature |
| No project picker in site/deploy forms | Users can't assign sites to projects |
| No project detail view | Users can't see what belongs to a project |

## 4. Proposed solution

Add three new UI surfaces, a new API file, a new hook, and sidebar navigation:

**Backend API (Go) — `components/auth/projects_api.go`:**
- `GET /api/orgs/{orgId}/projects` — list projects (org-scoped)  
  _Note: If e57s02 is done, this endpoint already exists. If not, this file provides it._
- `POST /api/orgs/{orgId}/projects` — create project
- `GET /api/orgs/{orgId}/projects/{id}` — get project details
- `PUT /api/orgs/{orgId}/projects/{id}` — update project name/slug
- `DELETE /api/orgs/{orgId}/projects/{id}` — delete project (only if no sites/deployments)
- `GET /api/orgs/{orgId}/projects/{id}/sites` — list sites in a project
- `GET /api/orgs/{orgId}/projects/{id}/deployments` — list deployments in a project

**UI — `ui/src/pages/ProjectsPage.tsx`:**
- Table listing all projects: name, slug, site count, created date
- "New Project" button → inline or modal form (name, slug)
- Row click → project detail view or expandable section
- Delete button with confirmation dialog
- Follows the same pattern as `SettingsPage.tsx` (PageHeader, Card, Button, table)

**UI — `ui/src/pages/ProjectDetailPage.tsx`:**
- Shows project name, slug, org, created date
- Lists sites and deployments belonging to this project
- Edit button for name/slug
- Delete button (blocked if sites or deployments exist)

**UI — `ui/src/hooks/useProjects.ts`:**
- `useProjects()` — fetches project list
- `useProject(id)` — fetches single project with sites/deployments
- `useCreateProject()`, `useUpdateProject()`, `useDeleteProject()` — mutations
- Pattern: `useState` + `useEffect` with `fetch()` calls to `/api/orgs/*/projects/*`, matching `useWorkspace.ts` style

**UI — Navigation:**
- Add "Projects" link to the sidebar in `Layout.tsx`, under the "Data" section (before "Users")
- Icon: `folder` or `layers`

## 5. Alternatives considered

- **Single-page with modals** — simpler but harder to show project details (sites, deployments).
  Split into list + detail page like the existing Sites pattern.
- **Embed in Settings page** — projects are organizational, not per-user settings. Deserves
  its own top-level nav entry.
- **Add to UsersPage** — users and projects are different domains. Mixing them would violate
  single-responsibility in the UI.

## 6. Who are the users?
- **BigBase administrators** managing projects for their organization
- **Developers** deploying sites who need to select a project
- **Platform operators** organizing multiple sites and deployments under project boundaries

## 7. Dependencies
- **e51 (Design System)** — UI components (PageHeader, Card, Button, Input, Badge, Icon).
  All already exist and are used by existing pages. No new design system primitives needed.
- **e57s02 (Projects Table CRUD)** — backend `projects` CRUD endpoints. If e57s02 is not done,
  this story provides the API endpoints in `components/auth/projects_api.go`.
- **e57s03 (DB Isolation)** — `project_id` column on sites/deployments. The project detail
  page queries these via `/api/orgs/{orgId}/projects/{id}/sites` and `/api/orgs/{orgId}/projects/{id}/deployments`.

## 8. Assumptions
- The Go API endpoints follow the same auth middleware pattern as existing `orgs` CRUD
  (JWT auth + org scoping via `X-Org-Id` header / `OrgIDFromContext`).
- The project ID is available in the UI after login (via `/api/auth/me` or similar).
- UI components (PageHeader, Card, Button, Input, Badge, Icon, Modal/ConfirmDialog) are
  already available from e51 or from the existing component library.
- Delete is blocked if `sites_count > 0` or `deployments_count > 0` (enforced by backend).

## 9. Risks
- **e57 not done yet**: The backend API endpoints may not exist. Mitigation: this story
  includes the full API endpoints in `components/auth/projects_api.go`. If e57s02 is
  already deployed, this file is not needed. Mark the file as `// e58s01: project CRUD API`.
- **e51 not done yet**: UI components may be missing. Mitigation: the existing component
  library already has PageHeader, Card, CardHeader, Button, Input, Badge, Icon — used
  extensively by existing pages. Verify availability before implementation.
- **UI test gap**: Only 1 UI test file exists. Mitigation: this story adds `.test.tsx`
  alongside the new pages, following the `SettingsPage.test.tsx` pattern.
- **Route scoping technical debt**: (Issue #43) These routes follow the newer `/api/orgs/...` convention. When resolving Issue #43, this API boundary might be rationalized further.

## 10. Non-goals
- This story does NOT add project-scoped data filtering on existing pages (e.g., showing
  only sites for the selected project). That's a follow-up enhancement.
- This story does NOT add a project picker/selector to the global navigation header.
- This story does NOT modify the existing org management UI.
- This story does NOT add real-time project event subscriptions.

## 11. Migration plan
1. Add `components/auth/projects_api.go` with project CRUD + sites/deployments listing endpoints
   (if e57s02 not yet deployed — otherwise skip, endpoints already exist).
2. Add API route registration in `auth.Start()` for the new endpoints.
3. Add `ui/src/hooks/useProjects.ts` with `useProjects`, `useProject`, `useCreateProject`,
   `useUpdateProject`, `useDeleteProject` hooks.
4. Add `ui/src/pages/ProjectsPage.tsx` with project list table and create/delete functionality.
5. Add `ui/src/pages/ProjectDetailPage.tsx` with project details, sites list, deployments list.
6. Add navigation entry in `ui/src/Layout.tsx` sidebar.
7. Add tests: `ProjectsPage.test.tsx`, `ProjectDetailPage.test.tsx`, `useProjects.test.ts`.

## 12. Wireframes / Diagrams

```
Projects page (list view):
┌────────────────────────────────────────────────────────────┐
│  Projects                                     [+ New]      │
│  Manage your organization's projects                       │
├────────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────────┐│
│  │ Name          Slug      Sites    Deployments   Created ││
│  │────────────── ────────  ─────── ──────────── ───────── ││
│  │ My App        my-app    3        12            Jan 15  ││
│  │ Landing Page  landing   1        4             Feb 3   ││
│  │ API Service   api-svc   2        8             Mar 20  ││
│  └────────────────────────────────────────────────────────┘│
└────────────────────────────────────────────────────────────┘

Project detail view:
┌────────────────────────────────────────────────────────────┐
│  ← Projects   /   My App                       [Edit] [×] │
├────────────────────────────────────────────────────────────┤
│  Name: My App    Slug: my-app                              │
│  Created: Jan 15, 2026                                     │
│                                                            │
│  Sites (3)                        Deployments (12)         │
│  ┌────────────── ────────┐        ┌────────── ────────┐   │
│  │ landing       Active  │        │ v1.2.3    Live    │   │
│  │ dashboard     Active  │        │ v1.2.2    Live    │   │
│  │ api           Active  │        │ v1.2.1    Live    │   │
│  └────────────── ────────┘        └────────── ────────┘   │
└────────────────────────────────────────────────────────────┘
```

## 13. API / Data Model

**New endpoints (if e57s02 not yet deployed):**
```
GET    /api/orgs/{orgId}/projects               → {data: [{id, name, slug, org_id, sites_count, deployments_count, created_at}]}
POST   /api/orgs/{orgId}/projects               → {data: {id, name, slug, org_id, created_at}}
GET    /api/orgs/{orgId}/projects/{id}          → {data: {id, name, slug, org_id, created_at}}
PUT    /api/orgs/{orgId}/projects/{id}          → {data: {id, name, slug}}
DELETE /api/orgs/{orgId}/projects/{id}          → {ok: true}  (409 if has sites/deployments)
GET    /api/orgs/{orgId}/projects/{id}/sites    → {data: [{id, name, domain, status, created_at}]}
GET    /api/orgs/{orgId}/projects/{id}/deployments → {data: [{id, version, status, created_at}]}
```

**All endpoints** use existing auth middleware (`Authorization: Bearer <JWT>`)
and org scoping (`OrgIDFromContext`).

**Response shape:** Matches existing conventions: `{"data": ...}` for success,
`{"error": "message"}` for errors.

## 14. Affected code

| File | Change |
|------|--------|
| `components/auth/projects_api.go` | **NEW** (if e57s02 not done) — project CRUD + sites/deployments listing |
| `components/auth/auth.go` | Register new routes in `Start()` if projects_api.go is new |
| `ui/src/hooks/useProjects.ts` | **NEW** — data fetching hooks for projects |
| `ui/src/pages/ProjectsPage.tsx` | **NEW** — project list page |
| `ui/src/pages/ProjectDetailPage.tsx` | **NEW** — project detail page |
| `ui/src/Layout.tsx` | Add "Projects" nav link to sidebar |
| `ui/src/App.tsx` | Add routes for `/projects` and `/projects/:id` |

## 15. Testing strategy

| Test | What it verifies |
|------|-----------------|
| `TestProjectCRUDAPI` (Go) | Create, list, get, update, delete project via HTTP — if new API added |
| `TestProjectDeleteBlocked` (Go) | Delete returns 409 when project has sites/deployments — if new API added |
| `ProjectsPage.test.tsx` (Vitest) | Renders project list, create button, delete confirmation |
| `ProjectDetailPage.test.tsx` (Vitest) | Renders project details, sites list, deployments list |
| `useProjects.test.ts` (Vitest) | Hooks return correct data shapes, handle errors |
| Full UI build | `cd ui && npm run build` succeeds |

## 16. Rollback plan
1. Revert `Layout.tsx` and `App.tsx` route changes.
2. Delete `ProjectsPage.tsx`, `ProjectDetailPage.tsx`, `useProjects.ts`.
3. Delete `projects_api.go` if new (or revert route registration if it exists from e57).
4. Rebuild UI: `cd ui && npm run build`.
5. Deploy. All pages still render.

## 17. Acceptance Criteria

- [ ] Go backend exposes project CRUD endpoints (either from e57s02 or new in this story):
  - [ ] `GET /api/orgs/{orgId}/projects` returns list (org-scoped)
  - [ ] `POST /api/orgs/{orgId}/projects` creates a project
  - [ ] `GET /api/orgs/{orgId}/projects/{id}` returns project details
  - [ ] `PUT /api/orgs/{orgId}/projects/{id}` updates name/slug
  - [ ] `DELETE /api/orgs/{orgId}/projects/{id}` deletes (409 if has sites/deployments)
  - [ ] `GET /api/orgs/{orgId}/projects/{id}/sites` returns sites list
  - [ ] `GET /api/orgs/{orgId}/projects/{id}/deployments` returns deployments list
- [ ] UI hook `useProjects.ts` with `useProjects`, `useProject`, `useCreateProject`, `useUpdateProject`, `useDeleteProject`
- [ ] `ProjectsPage.tsx` renders in `/projects` route with:
  - [ ] Project list table (name, slug, sites count, deployments count, created date)
  - [ ] "New Project" button with inline/modal form
  - [ ] Delete button with confirmation dialog
- [ ] `ProjectDetailPage.tsx` renders in `/projects/:id` route with:
  - [ ] Project name, slug, org, created date
  - [ ] Sites list (name, status)
  - [ ] Deployments list (version, status)
  - [ ] Edit and Delete buttons
- [ ] "Projects" link appears in sidebar navigation
- [ ] Tests pass:
  - [ ] `go test ./components/auth/ -run TestProjectCRUD -v` (if new API)
  - [ ] `cd ui && npx vitest run ProjectsPage.test.tsx`
  - [ ] `cd ui && npx vitest run useProjects.test.ts`
- [ ] `cd ui && npm run build` succeeds
- [ ] `go vet ./components/auth/` clean

## 18. Implementation Steps (see e58s01-tasks.yaml)

## 19. Verification Script (for manual UAT)

```bash
# 1. Start BigBase
go run . serve --port 9999 &
PID=$!
sleep 2

# 2. Login and get a token
TOKEN=$(curl -s -X POST http://localhost:9999/api/auth/register \
  -d '{"email":"admin@test.com","password":"secret123"}' | jq -r '.token')

# 3. Create a project
curl -s -H "Authorization: Bearer $TOKEN" \
  -X POST http://localhost:9999/api/orgs/1/projects \
  -d '{"name":"My App","slug":"my-app"}' | jq .

# 4. List projects
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:9999/api/orgs/1/projects | jq .

# 5. Open browser to http://localhost:9999
#    - Navigate to Projects page from sidebar
#    - Verify project list renders
#    - Click a project → detail view shows sites and deployments
#    - Try deleting a project with sites → should be blocked
#    - Create a new project via "New Project" button

kill $PID 2>/dev/null
```

## 20. Out of scope
- Project picker in global navigation header
- Project-scoped filtering on existing pages (Sites, Deployments)
- Realtime project event subscriptions
- Project member management (beyond what org membership provides)
- Usage/quotas per project
