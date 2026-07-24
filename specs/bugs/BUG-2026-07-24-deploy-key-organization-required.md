---
bug_id: BUG-2026-07-24-deploy-key-organization-required
status: in_progress
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

### 5. RED: Deploy key cannot list sites / escalate via org_id
**verify:** `TestSiteKeyOwnership_NoOrgEscalation`

## Acceptance

- [x] Site deploy key can redeploy its own site (not 403 organization required)
- [x] Site deploy key cannot access another site (404)
- [x] Legacy org_id=0 site still accessible via its own key
- [x] JWT cross-org isolation unchanged
- [x] No org_id injection for bb_dep_ in middleware
- [x] MCP ci-templates emit v3 TBR+deploy + `bigbase-deploy@v1`
- [ ] Related GH issues #161 / #162 commented with fix PR

## Resolution

**SAFE fix applied** in `components/sites/sites.go` `requireSiteOwnership`: site-key branch via `kernel.SiteIDFromContext` before org path.

**Tests:** `components/sites/sitekey_ownership_test.go` (own site 201, other site 404, legacy org_id=0, JWT cross-org 404, no org escalation).

**Also:** MCP `ci-templates.json` → v3 TBR+deploy + `bigbase-deploy@v1`; epic seed `specs/epics/e82-artifact-deploy/`.

**Evidence:** `go test ./components/sites/ ./components/auth/ ./components/mcp/ -count=1` green; `./components/deploy/` green except pre-existing `TestResumeSvelteKitStaticDeployment`; `go test ./... -skip 'TestResumeSvelteKitStaticDeployment|TestHealthCheckIntegrationFail'` green.

## Out of Scope

- Multipart artifact upload (e82 — seeded under `specs/epics/e82-artifact-deploy/`)
- Middleware org_id injection for deploy keys
- Consumer workflow migrations
