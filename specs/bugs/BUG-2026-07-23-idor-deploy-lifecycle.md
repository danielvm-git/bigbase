---
bug_id: BUG-2026-07-23-idor-deploy-lifecycle
status: open
severity: high
scope: security
title: "HIGH: IDOR on Deployment Lifecycle (list/logs/delete/rollback any deployment)"
github_issue: 141
created_at: 2026-07-23
---

## Problem

Beyond the cross-tenant deploy hijack (BUG-134, fixed), every other deployment endpoint lacks ownership verification. Any authenticated user can:

1. **List** all deployments across all orgs (HandleList)
2. **Read** any deployment's details by ID (handleDeployByID GET)
3. **Delete** any deployment by ID (handleDeleteDeployment)
4. **Read logs** of any deployment (handleDeployLogs)
5. **Rollback** any deployment (handleRollback)
6. **View rollback events** for any site (handleRollbackEvents)

The BUG-134 fix added org_id verification only to `HandleCreate`. All other handlers accept a deployment ID and operate on it without checking the caller's org owns that deployment.

## Root Cause Analysis

### Phase 1 — Reproduce

1. Authenticated as org A user, `GET /api/deploy` → returns ALL deployments from all orgs
2. `GET /api/deploy/{org-b-deployment-id}` → 200 with full deployment details
3. `DELETE /api/deploy/{org-b-deployment-id}` → 204, deployment deleted
4. `GET /api/deploy/{org-b-deployment-id}/logs` → 200 with build logs
5. `POST /api/deploy/{org-b-deployment-id}/rollback` → 200, deployment rolled back

### Phase 2 — Isolate

**Module:** `components/deploy/gateway.go` and `components/deploy/rollback.go`

Affected handlers:
- `HandleList` (line 93) — SELECT with no WHERE clause
- `handleDeployByID` GET (line 145) — SELECT by id only
- `handleDeleteDeployment` (line 170) — DELETE by id only
- `handleDeployLogs` (line 212) — SELECT by id only
- `handleRollback` (line 150 of rollback.go) — operates on id only
- `handleRollbackEvents` (line 234 of rollback.go) — SELECT by site_id only

The ownership chain: `deployments.site_id → sites.id → sites.org_id`. The caller's org_id is available via `auth.OrgIDFromContext(r.Context())`.

### Phase 3 — Hypothesize

| # | Hypothesis | Falsification Test |
|---|-----------|-------------------|
| H1 | All non-create handlers skip org_id check | Grep for OrgIDFromContext in gateway.go — only appears in HandleCreate ✅ |
| H2 | Database lacks org_id linkage | deployments has site_id, sites has org_id — chain exists but unused ✅ |

### Phase 4 — Verify

**Confirmed root cause:** Only `HandleCreate` (gateway.go:63-75) performs org_id ownership verification. All other handlers operate on deployment IDs without verifying the caller's org owns the deployment (via its site_id → sites.org_id chain).

## TDD Fix Plan

### 1. RED: HandleList returns only deployments belonging to caller's org
**GREEN:** When org_id is present in context, JOIN deployments→sites and filter by `sites.org_id = ?`. When no org_id (e.g., site deploy key), return all (or scope by site_id from context).
**verify:** `go test ./components/deploy/... -run TestIDOR -count=1 -v`

### 2. RED: handleDeployByID (GET) returns 403 for deployments in other orgs
**GREEN:** After fetching deployment, look up site's org_id and compare to caller's org_id. Return 403 if mismatch.
**verify:** `go test ./components/deploy/... -run TestIDOR -count=1 -v`

### 3. RED: handleDeleteDeployment returns 403 for deployments in other orgs
**GREEN:** Before delete, verify the deployment's site belongs to the caller's org.
**verify:** `go test ./components/deploy/... -run TestIDOR -count=1 -v`

### 4. RED: handleDeployLogs returns 403 for deployments in other orgs
**GREEN:** Before fetching logs, verify the deployment's site belongs to the caller's org.
**verify:** `go test ./components/deploy/... -run TestIDOR -count=1 -v`

### 5. RED: handleRollback returns 403 for deployments in other orgs
**GREEN:** Before rollback, verify the deployment's site belongs to the caller's org.
**verify:** `go test ./components/deploy/... -run TestIDOR -count=1 -v`

### 6. RED: handleRollbackEvents returns 403 for sites in other orgs
**GREEN:** Before returning rollback events, verify the site belongs to the caller's org.
**verify:** `go test ./components/deploy/... -run TestIDOR -count=1 -v`

### 7. RED: Stats endpoint filtered by org (optional — lower priority)
**GREEN:** Filter deploy stats by caller's org_id.
**verify:** `go test ./components/deploy/... -run TestIDOR -count=1 -v`

**REFACTOR:** Extract shared `verifyDeploymentOwnership(ctx, db, deploymentID)` helper to avoid code duplication across handlers.

## Security Impact

**HIGH** — Any authenticated user can read, delete, or rollback any deployment across all organizations. This is a data confidentiality, integrity, and availability issue.

## Exploit Path

1. Attacker authenticates with valid JWT/API-key for their org
2. Enumerates deployment IDs (via HandleList which returns all)
3. Reads build logs (may contain secrets, env vars, source code)
4. Deletes victim deployments (DoS)
5. Triggers rollbacks on victim deployments (availability impact)

## Acceptance Criteria

- [ ] HandleList scoped by caller's org_id
- [ ] handleDeployByID returns 403 for cross-org access
- [ ] handleDeleteDeployment returns 403 for cross-org access
- [ ] handleDeployLogs returns 403 for cross-org access
- [ ] handleRollback returns 403 for cross-org access
- [ ] handleRollbackEvents returns 403 for cross-org access
- [ ] All IDOR tests pass
- [ ] Existing tests still pass

## Resolution

**Status:** Fixed
**Date:** 2026-07-23
**Commit:** pending

### Changes Made

1. **`components/auth/middleware.go`** — Added `WithOrgID(ctx, orgID)` helper for test injection
2. **`components/deploy/gateway.go`** — Added:
   - `verifyDeploymentOwnership()` — checks deployment's site belongs to caller's org
   - `verifySiteOwnership()` — checks site belongs to caller's org
   - `HandleList` — scoped by org_id via JOIN to sites table
   - `handleDeployByID` GET — ownership check before returning deployment
   - `handleDeleteDeployment` — ownership check before delete
   - `handleDeployLogs` — ownership check before returning logs
3. **`components/deploy/rollback.go`** — Added:
   - `handleRollback` — ownership check before rollback
   - `handleRollbackEvents` — site ownership check before returning events
4. **`components/deploy/idor_test.go`** — 11 new tests covering all IDOR scenarios

### Test Results

- All 11 IDOR tests pass
- All existing rollback tests pass
- All existing sitekey auth tests pass
- No regressions
