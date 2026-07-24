---
bug_id: BUG-2026-07-23-idor-functions-handlers
status: fixed
severity: critical
scope: functions
title: "CRITICAL: IDOR on Functions — read/execute/delete any org's functions"
github_issue: 131
---

## Problem

The `functions` table has no `org_id` column; no handler checks ownership — only authentication.

## Exploit

Any authenticated user can:
- **Read** another org's function source (may embed secrets in Env map)
- **Execute** it with attacker input
- **Overwrite/delete** it
- **Read execution logs** (may leak sensitive output)

## Affected Handlers

| Handler | Line | Vulnerability |
|---------|------|--------------|
| `handleCreate` | handlers.go:87 | No `org_id` in INSERT |
| `handleList` | handlers.go:131 | Returns ALL functions, no org filter |
| `handleGet` | handlers.go:155 | Fetches by id, no ownership check |
| `handleUpdate` | handlers.go:160 | Updates by id, no ownership check |
| `handleDelete` | handlers.go:188 | Deletes by id, no ownership check |
| `handleRun` | handlers.go:204 | Runs by id, no ownership check |
| `handleFunctionLogs` | handlers.go:247 | Reads logs, no ownership check |
| `saveExecution` | handlers.go:235 | No `org_id` in execution record |

## Root Cause

`functions` table schema has no `org_id` column. All SQL queries use bare `WHERE id = ?` with no tenant scoping.

## Fix

1. Add `org_id INTEGER NOT NULL DEFAULT 0` to `functions` table
2. Add `org_id INTEGER NOT NULL DEFAULT 0` to `function_executions` table
3. In `handleCreate`: extract `orgID` via `auth.OrgIDFromContext(r.Context())`, include in INSERT
4. In `handleList`: add `WHERE org_id = ?` filter
5. In `handleGet/handleUpdate/handleDelete/handleRun/handleFunctionLogs`: verify function's `org_id` matches caller's org (404 if mismatch — don't leak existence)
6. In `saveExecution`: include `org_id` in INSERT

## Pattern Reference

Follow the same pattern as:
- `components/sites/sites.go:532-540` — org_id stamp on create
- `components/deploy/gateway.go:60-80` — org_id ownership check on access
- `components/functions/runtime.go:166-296` — already has org_id scoping for db.collection() (BUG-132 fix)

## Verify

- [x] Functions created with correct org_id
- [x] `GET /api/functions` returns only caller's org functions
- [x] `GET /api/functions/:id` returns 404 for cross-org access
- [x] `PUT /api/functions/:id` returns 404 for cross-org access
- [x] `DELETE /api/functions/:id` returns 404 for cross-org access
- [x] `POST /api/functions/:id/run` returns 404 for cross-org access
- [x] `GET /api/functions/:id/logs` returns 404 for cross-org access
- [x] Execution records have correct org_id
- [x] Existing tests pass (29/29)
- [x] New test: cross-tenant function isolation (TestFunctionsCrossTenantIsolation)
- [x] go vet clean
- [x] go build clean
