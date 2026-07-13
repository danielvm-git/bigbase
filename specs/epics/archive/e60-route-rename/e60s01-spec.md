# e60s01: Rename /deploy/* → /sites/* Routes

**Story ID:** e60s01 | **Epic:** e60 — Route Rename | **BCPs:** 1 | **Status:** planned

## 1. Type & Context

**type:** chore
**context:** ui
**maturity:** 3 — Countable

## 2. Story Statement

**As a** developer or user of BigBase,
**I want** the admin UI and backend to use `/sites` instead of `/deploy` for site management routes,
**so that** the codebase matches every prototype, IA spec, and external reference that calls this feature "Sites".

## 3. Scope

### Frontend (`ui/src/`)
- `App.tsx` — change `path="deploy"` → `path="sites"`, `path="deploy/new"` → `path="sites/new"`, `path="deploy/:siteId"` → `path="sites/:siteId"`
- `Layout.tsx` — change `to: '/deploy'` → `to: '/sites'`
- All `navigate('/deploy')` / `useNavigate` calls in page components (~16 references total)

### Backend (`main.go`, `components/`)
- `main.go` — add 301 redirect handler: `GET /deploy` → `/sites`, `GET /deploy/*` → `/sites/*`
- No handler logic moves — redirect shim only; all existing `/deploy` Go handlers remain untouched until a future cleanup

### Tests
- Update 11 test files that reference `/deploy` paths to use `/sites`

## 4. Backward Compatibility

The 301 redirect from `/deploy/*` → `/sites/*` in `main.go` ensures:
- Bookmarked URLs continue to work
- Any external scripts or CI jobs using `/deploy` continue to work
- No data migration required (routes are UI-only; no DB references to route strings)

## 5. Acceptance Criteria

```gherkin
Scenario: Sites list accessible at /sites
  Given the user is logged in
  When they navigate to /sites
  Then they see the sites list page

Scenario: /deploy redirects to /sites
  Given an old bookmark to /deploy
  When the user clicks it
  Then they are redirected to /sites with 301

Scenario: Site detail accessible at /sites/:id
  When the user navigates to /sites/abc123
  Then they see the site detail page
```

## 6. Verify

```bash
# UI build passes with renamed routes
cd ui && npm run build

# Go build passes with redirect shim
go build ./...

# Tests pass
go test ./... && cd ui && npm run test
```
