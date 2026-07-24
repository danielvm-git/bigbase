---
bug_id: BUG-135
status: fixed
severity: critical
scope: auth
title: "IDOR on Site Deploy Keys — any user can mint/revoke keys for any site"
---

## Summary

`handleCreateSiteKey`, `handleListSiteKeys`, and `handleRevokeSiteKey` in `components/auth/org_http.go` take `siteID` from the URL path with no ownership verification. Any authenticated user can mint, list, or revoke deploy keys for any site, enabling:

1. **Unauthorized deployments** — mint a `bb_dep_` key for any site, then trigger deployments
2. **CI pipeline DoS** — revoke keys belonging to other users' CI pipelines
3. **Information disclosure** — list all deploy keys for any site

## Root Cause

The three handlers only check authentication (`UserIDFromContext`) but never verify that the caller owns the target site. The `sites` table has an `org_id` column, but the handlers don't join against it.

### Affected Code

- `components/auth/org_http.go` — `handleCreateSiteKey` (line ~302), `handleListSiteKeys` (line ~340), `handleRevokeSiteKey` (line ~370)
- `components/auth/apikeys.go` — `CreateSiteKey`, `ListSiteKeys`, `RevokeSiteKey` methods (no org_id parameter)

## Fix Approach

1. Add `lookupSiteOrgID` helper to query `SELECT org_id FROM sites WHERE id = ?`
2. In each handler, after extracting `siteID` from the URL:
   - Call `lookupSiteOrgID(siteID)` to get the owning org_id
   - Call `lookupOwnedOrg(ctx, orgID, userID)` to verify ownership
   - Return 403 if ownership check fails
3. Add cross-tenant IDOR test: register two users with separate orgs, verify user B cannot access user A's site keys

## Verify Steps

1. `go test ./components/auth/ -run TestSiteKey -v` — all site key tests pass
2. `go test ./components/auth/ -run TestIDOR -v` — cross-tenant isolation test passes
3. `go test ./components/auth/ -v` — full auth package tests pass
4. `go vet ./...` — no lint issues
