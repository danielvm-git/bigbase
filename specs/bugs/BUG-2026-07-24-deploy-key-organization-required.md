---
bug_id: BUG-2026-07-24-deploy-key-organization-required
status: fixed
severity: critical
scope: auth,sites,deploy
title: "Deploy keys (bb_dep_*) return 403 organization required on POST /api/sites/{id}/deploy"
github_issue: 161
created_at: 2026-07-24
related:
  - BUG-136
---

## Problem

Site-scoped deploy keys (`bb_dep_*`) fail with **403 `organization required`** on `POST /api/sites/{id}/deploy`. GitHub Actions using `bigbase-deploy` and REST clients provisioning keys via MCP cannot redeploy.

Consumer reports: bigbase #161, #162.

## Root Cause

Commit introducing BUG-136 org scoping added `requireSiteOwnership()` to all site-by-id handlers including `redeploySite()`. That helper requires `auth.OrgIDFromContext(ctx)`, but deploy-key middleware only sets `kernel.WithSiteID()` — not org_id.

JWT and org API keys work because middleware sets both org context values.

## Safe Fix (chosen)

Extend `requireSiteOwnership()` with a **site-key branch** (mirror `components/deploy/gateway.go` `SiteIDFromContext` match):

- When `kernel.SiteIDFromContext(ctx)` is set, resolve URL `siteID` (id or `git_repo_id` alias) and allow only when it matches the key's site.
- Return **404** on cross-site access (no existence leak).
- **Do NOT** inject `org_id` for `bb_dep_*` in auth middleware — that would expand deploy keys to org-wide access.

## TDD Fix Plan

### 1. RED: Deploy key + own site deploy → passes ownership (not 403)
**verify:** `go test ./components/sites/... -run TestSiteKey -count=1 -v`

### 2. RED: Deploy key + other site → 404
**verify:** same

### 3. RED: Legacy org_id=0 site + deploy key → allowed
**verify:** same

### 4. RED: JWT cross-org → 404 (BUG-136 regression guard)
**verify:** existing org_scoping tests

### 5. RED: Deploy key cannot access unrelated site via GET
**verify:** site-key ownership tests

## Regression Guards

- `components/sites/sitekey_ownership_test.go`

## Out of Scope

- Multipart artifact upload (e82)
- Middleware org_id injection for deploy keys


## Resolution

Landed in #163. SAFE site-key branch in `requireSiteOwnership` via `kernel.SiteIDFromContext`. MCP v3 templates + e82 seed included. Issues #161/#162 closed.

## Certification

**Validated:** 2026-07-27
**Evidence:** All tests pass in `components/sites/...`. Linters pass with 0 issues. Code hardening verified in `requireSiteOwnership` where site existence leak is prevented (returns 404 instead of 403 on mismatch) and `org_id` escalation is correctly blocked.
**Status:** PASS
