---
bug_id: BUG-2026-07-07T223200
status: fixed
severity: medium
scope: mcp,sites,auth
title: MCP references list_sites but tool does not exist (e69 / issue #57)
github_issue: 57
---

# BUG-2026-07-07T223200: MCP list_sites referenced but not implemented

## Problem

**Actual:** `provision_ci_credentials` error text tells agents to use `list_sites`, but no such MCP tool is registered. Site-scoped deploy keys (`bb_dep_*`) can be created but not listed or revoked via MCP.

**Expected:** Agents can discover sites (`list_sites`, `get_site`), inspect deploy metadata, list site keys, and revoke compromised keys (`revoke_site_key`) without admin UI access.

**Reproduce:** Call MCP tool `list_sites` — tool not found. Call `provision_ci_credentials` without `site_id` — message references nonexistent `list_sites`.

**Security impact:** MEDIUM — leaked `bb_dep_*` tokens have no agent-accessible revocation path; rotation requires direct DB or admin UI.

## Root Cause Analysis

e67 shipped MCP provisioning (`create_repo` → `create_site` → `provision_ci_credentials`) but stopped before day-2 operations. The credential tool's help text assumed `list_sites` would exist. Auth stores site keys with a `revoked` column and `ResolveSiteKey` honors it, but no MCP surface exposes list/revoke.

**Risk level:** Medium — operational gap, not data corruption; blocks big-library CI key rotation.

## TDD Fix Plan

1. **RED:** MCP test — `list_sites` returns JSON array with site_id, name, git_repo_id when SiteLister wired.
   **GREEN:** Add `SiteLister` interface; implement `ListSites`/`GetSite` on Sites; register `list_sites`/`get_site` tools.
   **verify:** `go test ./components/mcp/... -run TestListSites`

2. **RED:** MCP test — `get_site` returns production_branch and last deploy status.
   **GREEN:** Wire `get_site` to `SiteLister.GetSite`.
   **verify:** `go test ./components/mcp/... -run TestGetSite`

3. **RED:** Auth test — `RevokeSiteKey` causes `ResolveSiteKey` to fail; `ListSiteKeys` returns metadata.
   **GREEN:** Add `ListSiteKeys` and `RevokeSiteKey` on Auth; extend `SiteKeyCreator` interface.
   **verify:** `go test ./components/auth/... -run TestSiteKeyRevoke`

4. **RED:** MCP test — `list_site_keys` and `revoke_site_key` tools work end-to-end.
   **GREEN:** Register tools; wire `SiteKeyCreator` in main.go (same auth component).
   **verify:** `go test ./components/mcp/... -run 'SiteKey|list_site'`

**REFACTOR:** Extract shared site-query logic from HTTP handlers into `ListSites`/`GetSite` methods.

## Acceptance Criteria

- [x] `list_sites` MCP tool registered and returns site catalog
- [x] `get_site` returns url, production_branch, last deployment status
- [x] `list_site_keys` returns key metadata for a site
- [x] `revoke_site_key` sets revoked=1; old token fails `ResolveSiteKey`
- [x] All new tests pass
- [x] Existing tests still pass

## Resolution

Fixed on branch `fix/e69-mcp-site-discovery`.

- Added `SiteLister` interface and MCP tools `list_sites`, `get_site`
- Extended `SiteKeyCreator` with `ListSiteKeys`, `RevokeSiteKey`; MCP tools `list_site_keys`, `revoke_site_key`
- `Sites.ListSites` / `Sites.GetSite` extracted from HTTP handlers
- `Auth.ListSiteKeys` / `Auth.RevokeSiteKey` on `org_api_keys`
- Wired `SiteLister: st` in `main.go`

→ verify: `go test ./components/mcp/... ./components/auth/... -run 'Site|Key|list_sites'` (21 passed)
