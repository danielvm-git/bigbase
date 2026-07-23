---
bug_id: BUG-2026-07-13T150000
status: fixed
severity: critical
scope: functions
title: "IDOR via Function Runtime DB Access (injectDB)"
github_issue: 132
---

## Problem

`db.collection(name)` in user function code issues raw queries with **no `org_id` filter**.

Any tenant can read/write/delete another tenant's data via:
```js
db.collection("orders").list()  // returns ALL tenants' rows
```

## Root Cause

`RunContext` has no `OrgID` field. `injectDB` receives only `kernel.DBer` — no tenant context. All queries run unscoped.

## Affected File

`components/functions/runtime.go:200-296` (`injectDB`)

## Fix

1. Add `OrgID int64` to `RunContext`
2. Pass `OrgID` to `injectDB(vm, dber, orgID)`
3. In `injectDB`:
   - Add `org_id INTEGER NOT NULL DEFAULT 0` to CREATE TABLE
   - Add `AND org_id = ?` to all SELECT/UPDATE/DELETE WHERE clauses
   - Add `org_id` to all INSERT statements
4. Update call sites in `functions.go:218` and `handlers.go:239` to extract OrgID from request context via `auth.OrgIDFromContext`

## Verify

- [x] `db.collection("x").list()` only returns caller's org rows
- [x] `db.collection("x").create({})` stamps org_id automatically
- [x] `db.collection("x").get(id)` scoped to org
- [x] `db.collection("x").update(id, {})` scoped to org
- [x] `db.collection("x").delete(id)` scoped to org
- [x] Existing tests pass (26/26)
- [x] New test: cross-tenant isolation (TestDBCrossTenantIsolation)
