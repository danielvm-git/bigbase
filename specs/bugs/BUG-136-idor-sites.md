---
bug_id: BUG-136
status: fixed
severity: critical
scope: security
title: "IDOR: Sites API lacks org_id multi-tenant isolation"
github_issue: 136
created_at: 2026-07-23
fixed_at: 2026-07-23
---

## Problem

All sites HTTP API endpoints and MCP tools query the `sites` table without filtering by `org_id`. Any authenticated user can:
- **List** all sites across all organizations (`GET /api/sites`)
- **Read** any site's details (`GET /api/sites/{id}`)
- **Delete** any site (`DELETE /api/sites/{id}`)
- **Trigger redeployment** of any site (`POST /api/sites/{id}/deploy`)
- **Read/write** any site's manifest, auth policy, env vars, domains, logs
- **Manage** deploy keys for any site

The `sites` table already has an `org_id` column (added by migration) and `insertSite` correctly stamps it on creation, but no read/update/delete operation uses it.

## Root Cause Analysis

### Reproduce
1. Create two orgs (Org A, Org B) with separate users
2. User A creates a site (gets org_id = A)
3. User B calls `GET /api/sites` — sees User A's site
4. User B calls `DELETE /api/sites/{userA_site_id}` — deletes User A's site

### Isolate
The vulnerability is in `components/sites/sites.go`. Every query function uses bare `WHERE id = ?` without an `AND org_id = ?` clause:

- `ListSites()` — `SELECT ... FROM sites ORDER BY s.name` (no WHERE at all)
- `GetSite()` — `WHERE s.id = ? OR s.git_repo_id = ?`
- `siteDeleteTarget()` — `WHERE id = ? OR git_repo_id = ?`
- `getSiteAuthPolicy()` — `WHERE id = ?`
- `setSiteAuthPolicy()` — `UPDATE ... WHERE id = ?`
- `resolveRepoBranch()` — `WHERE id = ?`
- `listSiteRequestLogs()` — `WHERE site_id = ?`

The MCP `list_sites`, `get_site`, `provision_ci_credentials`, `list_site_keys`, `revoke_site_key`, and `set_site_auth_policy` tools delegate to the same unscoped functions or perform their own unscoped queries.

### Hypothesize
The org_id column was added to `sites` during the BUG-134 fix (cross-tenant deploy hijack), but only `insertSite` was updated to populate it. The existing read/update/delete paths were not retrofitted with org_id filtering.

### Verify
Confirmed by reading `components/sites/sites.go` — no `org_id` filter in any query. The `auth.OrgIDFromContext(ctx)` helper exists and works correctly (used in `insertSite`), but is not called in any read path.

## Affected Endpoints

| Endpoint | Method | Vulnerability |
|----------|--------|---------------|
| `/api/sites` | GET | Lists all orgs' sites |
| `/api/sites/{id}` | GET | Reads any org's site |
| `/api/sites/{id}` | DELETE | Deletes any org's site |
| `/api/sites/{id}/deploy` | POST | Redeploys any org's site |
| `/api/sites/{id}/manifest` | GET/POST | Read/write any site's manifest |
| `/api/sites/{id}/auth-policy` | GET/POST | Read/write any site's auth policy |
| `/api/sites/{id}/domains` | GET/POST/DELETE | Manage any site's domains |
| `/api/sites/{id}/env-vars` | GET/POST/DELETE | Manage any site's env vars |
| `/api/sites/{id}/logs` | GET | Read any site's request logs |
| MCP `list_sites` | tool | Lists all orgs' sites |
| MCP `get_site` | tool | Reads any org's site |
| MCP `provision_ci_credentials` | tool | Creates keys for any site |
| MCP `list_site_keys` | tool | Lists keys for any site |
| MCP `revoke_site_key` | tool | Revokes keys for any site |
| MCP `set_site_auth_policy` | tool | Modifies any site's policy |

## TDD Fix Plan

1. **RED**: Write test `TestListSites_IsolatedByOrg` — create sites for two orgs, verify each org only sees its own
   **GREEN**: Add `WHERE org_id = ?` to `ListSites()` query, parameterized from `auth.OrgIDFromContext(ctx)`
   **verify**: `go test ./components/sites/ -run TestListSites_IsolatedByOrg`

2. **RED**: Write test `TestGetSite_CrossOrg404` — Org A's site returns 404 for Org B
   **GREEN**: Add `AND org_id = ?` to `GetSite()` query
   **verify**: `go test ./components/sites/ -run TestGetSite_CrossOrg404`

3. **RED**: Write test `TestDeleteSite_CrossOrg404` — Org B cannot delete Org A's site
   **GREEN**: Add `AND org_id = ?` to `siteDeleteTarget()` query
   **verify**: `go test ./components/sites/ -run TestDeleteSite_CrossOrg404`

4. **RED**: Write test `TestRedeploySite_CrossOrg404` — Org B cannot redeploy Org A's site
   **GREEN**: Add `AND org_id = ?` to `redeploySite()` site lookup query
   **verify**: `go test ./components/sites/ -run TestRedeploySite_CrossOrg404`

5. **RED**: Write test `TestSiteAuthPolicy_CrossOrg404` — Org B cannot read/write Org A's auth policy
   **GREEN**: Add `AND org_id = ?` to `getSiteAuthPolicy()` and `setSiteAuthPolicy()` queries
   **verify**: `go test ./components/sites/ -run TestSiteAuthPolicy_CrossOrg404`

6. **RED**: Write test `TestSiteManifest_CrossOrg404` — Org B cannot read Org A's manifest
   **GREEN**: Add `AND org_id = ?` to `resolveRepoBranch()` query
   **verify**: `go test ./components/sites/ -run TestSiteManifest_CrossOrg404`

7. **RED**: Write test `TestSiteRequestLogs_CrossOrg403` — Org B cannot read Org A's logs
   **GREEN**: Add org_id check to `listSiteRequestLogs()` before querying
   **verify**: `go test ./components/sites/ -run TestSiteRequestLogs_CrossOrg403`

8. **RED**: Write test `TestMCPListSites_IsolatedByOrg` — MCP list_sites only returns caller's org sites
   **GREEN**: Pass org_id from MCP context to `ListSites()` or add org_id filter to MCP site queries
   **verify**: `go test ./components/mcp/ -run TestMCPListSites_IsolatedByOrg`

**REFACTOR**: Extract `requireSiteOwnership(ctx, siteID)` helper that queries `SELECT org_id FROM sites WHERE id = ?` and returns 403/404 if mismatch. Use in all handlers.

## Acceptance Criteria

- [ ] `GET /api/sites` returns only caller's org sites
- [ ] `GET /api/sites/{id}` returns 404 for cross-org site
- [ ] `DELETE /api/sites/{id}` returns 404 for cross-org site
- [ ] `POST /api/sites/{id}/deploy` returns 404 for cross-org site
- [ ] Auth policy endpoints return 404 for cross-org site
- [ ] Manifest endpoints return 404 for cross-org site
- [ ] Request logs return 403 for cross-org site
- [ ] MCP tools scoped by org_id
- [ ] All existing tests pass
- [ ] All new isolation tests pass

## Security Impact

**CRITICAL** — Full cross-tenant data access. Any authenticated user can enumerate, read, modify, delete, or redeploy any site across all organizations. Exploit path: `GET /api/sites` returns all site IDs, then any write endpoint accepts arbitrary site_id.

## Resolution

**Fixed:** 2026-07-23
**Root cause confirmed:** All site HTTP endpoints and MCP tools queried the `sites` table without filtering by `org_id`, allowing any authenticated user to access/modify/delete sites belonging to other organizations.
**Fix applied:** Added `requireSiteOwnership()` helper function that verifies the caller's org owns the site. Added `org_id` filtering to `ListSites()`. Applied ownership checks to all site endpoints: getSite, deleteSite, redeploySite, getSiteManifest, saveSiteManifest, getSiteAuthPolicy, setSiteAuthPolicy, listSiteRequestLogs, handleDomains, handleEnvVarsRoute.
**Hardening added:** Type guard via `requireSiteOwnership()` that checks org_id before any site operation. Returns 404 (not 403) to avoid leaking site existence across orgs.
**Evidence:** All 27 packages pass (`go test ./...`). 8 new IDOR isolation tests verify cross-org access is denied.
**Commit:** `fix(security): add org_id multi-tenant isolation to all site endpoints`
