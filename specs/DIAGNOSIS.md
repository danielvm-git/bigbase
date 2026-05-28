## Problem

Navigating directly to `http://localhost:9999/repos` (or any other SPA page path) returns the server's home page HTML instead of the admin UI page. The user cannot bookmark or directly type SPA page URLs.

- **Actual**: `GET /repos` returns 200 with the BigBase home page (component status table)
- **Expected**: `GET /repos` redirects to `http://localhost:9999/admin/#/repos` (302 Found)
- **Reproduce**: Open browser at `http://localhost:9999/repos`

## Root Cause Analysis

The admin UI is a React SPA served at `/admin/` using HashRouter. All page URLs are hash-based: `/admin/#/repos`, `/admin/#/deploy`, etc. Users can navigate between pages via sidebar links within the SPA.

However, the Go server has no redirect routes for the SPA page paths. The catch-all root handler `/` matches every path that doesn't have an explicit handler (API routes, `/health`, `/admin/`). `GET /repos` falls through to this catch-all and returns the home page HTML template.

The side navigation links in `Layout.tsx` use `<Link to="/repos">` etc., which produce `href="#/repos"` — these work correctly when the SPA is already loaded. The problem only occurs on direct URL access or page refresh when the browser sends a full HTTP request to the Go server.

**Risk level**: Low — no data loss or corruption. Only affects direct URL access to SPA pages; sidebar navigation within the app works correctly.

## TDD Fix Plan

1. **RED**: Write a test in the proxy test that starts a server with a redirect handler registered for `/repos`, calls `GET /repos`, and asserts the response is 302 with `Location: /admin/#/repos`.
   **GREEN**: Add redirect routes in `main.go` for each SPA page path, registered as `GET /repos`, `GET /deploy`, etc., each redirecting to `/admin/#/<path>`.
   **verify**: `curl -w '%{redirect_url}' http://localhost:9999/repos`

**REFACTOR**: Define the SPA path list as a shared constant to keep main.go DRY.

## Acceptance Criteria

- [ ] `GET /repos` returns 302 → `/admin/#/repos`
- [ ] `GET /deploy` returns 302 → `/admin/#/deploy`
- [ ] `GET /data` returns 302 → `/admin/#/data`
- [ ] `GET /sql` returns 302 → `/admin/#/sql`
- [ ] `GET /users` returns 302 → `/admin/#/users`
- [ ] `GET /messaging` returns 302 → `/admin/#/messaging`
- [ ] `GET /storage` returns 302 → `/admin/#/storage`
- [ ] `GET /functions` returns 302 → `/admin/#/functions`
- [ ] `GET /forge` returns 302 → `/admin/#/forge`
- [ ] `GET /cici` returns 302 → `/admin/#/cici`
- [ ] API endpoints (`/api/auth/login`, `/api/git/repos`, etc.) are not affected
- [ ] `/health` is not affected
- [ ] The root `/` still shows the home page
- [ ] All existing tests pass

## Resolution

Added 11 server-side redirect routes in `main.go` for SPA page paths. Each redirects `GET /<path>` → `302 /admin/#/<path>`:

`/repos`, `/deploy`, `/messaging`, `/storage`, `/functions`, `/forge`, `/cici`, `/data`, `/sql`, `/users`, `/login`

The redirects are registered before the kernel starts, using Go 1.22+ method-pattern routing (`"GET /repos"`). This ensures they take precedence over the catch-all `/` handler without affecting API routes, health check, or the home page.
