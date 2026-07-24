---
bug_id: BUG-2026-07-24T165040
status: resolved
severity: high
scope: deploy
title: CI red on main — TestDeployContract 500s because GET /api/deploy org-scoped query joins sites table the test never creates
---

# BUG-2026-07-24T165040: TestDeployContract fails 500 — deploy list org-scope query assumes `sites` table exists

## Problem

**Actual:** `TestDeployContract` (tests/contract/contract_test.go) fails on both sqlite and postgres CI matrix jobs:
```
contract_test.go:212: status code: got 500, want 200
contract_test.go:216: response missing field "data"
```
CI/CD on `main` has been red since commit `60fb5066e` (fix: cross-tenant deployment hijack via missing org_id check), confirmed via `gh run view 30114291380 --log-failed` (workflow CI/CD, triggered by merge of PR #152).

**Expected:** `GET /api/deploy` returns 200 with `{"data": [...]}`.

**Reproduce:** `go test ./tests/contract/... -run TestDeployContract -v -count=1`

**Security impact:** NONE — production topology (main.go) always registers the `sites` component before serving requests, so this never manifests outside of test isolation.

## Root Cause Analysis

### Reproduce
Confirmed locally with `go test ./tests/contract/...`. Same failure as CI.

### Isolate
Added temporary debug logging to `components/deploy/gateway.go: HandleList`. Output:
```
DEBUG: took org branch, orgID= 1 ok= true
DEBUG list deployments error: SQL logic error: no such table: sites (1)
```
`tests/contract/contract_test.go`'s `doRequest` helper injects `auth.WithOrgID(ctx, 1)` into every request (added for BUG-136 site multi-tenant isolation tests). This routes `HandleList` into its org-scoped branch, which runs:
```sql
SELECT ... FROM deployments d LEFT JOIN sites s ON d.site_id = s.id WHERE s.org_id = ? OR ...
```
`TestDeployContract` registers only `db`, `git`, `deploy` (matching `deploy.Dependencies() = ["db", "git"]`) — it never registers the `sites` component, so the `sites` table doesn't exist and the JOIN fails.

### Hypothesize
The org-scoped deploy IDOR fix (`c69ccf4ef`, BUG-2026-07-23-idor-deploy-lifecycle) added a hard dependency on the `sites` table without declaring it, and without a corresponding test fixture update in `tests/contract/contract_test.go` (unlike `components/deploy/idor_test.go`, which already creates a `sites` table via `setupIDORTest` for exactly this reason).

### Verify
- `main.go` registers the `sites` component (`s`) and `deploy` component (`depComp`) together; `kernel.Start()` runs every registered component's `Start()` to completion, sequentially, before the HTTP server accepts requests — so `sites` is always migrated before any request reaches `HandleList` in production. Confirmed no production risk.
- `components/deploy/idor_test.go` already establishes the convention: any deploy test exercising the org-scoped path must provide a `sites` table/component.
- `tests/contract/contract_test.go`'s `TestSitesContract` shows the correct pattern: construct the real `sites.New(...)` component and register it with the kernel.

**Risk level:** Low (test-only gap; no production code path affected)

## TDD Fix Plan

1. **RED:** `TestDeployContract` fails 500 on `GET /api/deploy` (already failing in CI/main — no new test needed, existing contract test is the regression signal)
2. **GREEN:** Register the real `sites` component in `TestDeployContract`'s kernel, matching `TestSitesContract`'s pattern and production's component topology
   **verify:** `go test ./tests/contract/... -run TestDeployContract -v -count=1`

## Acceptance Criteria

- [x] `TestDeployContract` passes locally and mirrors production component topology
- [x] Full suite green: `go test ./... -count=1`
- [x] `go vet ./...` clean
- [x] `go build ./...` clean

## Resolution

**Fixed:** 2026-07-24

**Root cause:** `tests/contract/contract_test.go: TestDeployContract` registered only `db`, `git`, `deploy` — omitting the `sites` component that the deploy IDOR fix's org-scoped `HandleList` query depends on via a direct SQL JOIN. Every contract-test request carries `org_id=1` (via `doRequest`), so the org-scoped branch always runs and always hit the missing table.

**Fix applied:** Register `sites.New(sites.Options{DB: dd, Logger: testLogger{}})` alongside `deploy` in `TestDeployContract`, matching production wiring (`main.go`) and the existing `TestSitesContract`/`idor_test.go` conventions.

**Evidence:**
- `go test ./tests/contract/... -run TestDeployContract -v -count=1` — 1 passed
- `go test ./... -count=1` — 1159 passed, 0 failed
- `go vet ./...` — clean
- `go build ./...` — clean
