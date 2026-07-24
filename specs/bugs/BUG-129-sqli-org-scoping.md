---
bug_id: BUG-129
status: fixed
severity: critical
scope: components/api
title: "SQL Injection / Tenant-Isolation Bypass via unparenthesized WHERE clause in scopeQueryForOrg"
github_issue: 129
---

# BUG-129: SQL Injection / Tenant-Isolation Bypass in scopeQueryForOrg

## Summary

`scopeQueryForOrg` in `components/api/api.go:678-706` appends `AND org_id = ?` to existing WHERE clauses without wrapping the original predicate in parentheses. SQL operator precedence (AND binds tighter than OR) allows a crafted query to bypass tenant isolation.

## Root Cause

When a query already contains `WHERE`, the function appends `AND org_id = ?` at the insertion point:

```go
// Line 700-701
return normalized[:insertPoint] + " AND org_id = ? " + normalized[insertPoint:], []any{orgID}
```

A user submitting:
```sql
SELECT id, data FROM items WHERE id > 0 OR 1=1
```

Gets transformed to:
```sql
SELECT id, data FROM items WHERE id > 0 OR 1=1 AND org_id = ?
```

By SQL precedence: `(id > 0) OR (1=1 AND org_id = ?)` → always true → full table leak across all tenants.

## Exploit

```bash
curl -X POST /api/sql \
  -H "Authorization: Bearer <admin-token>" \
  -H "X-Org-Id: 1" \
  -d '{"query":"SELECT id, data FROM items WHERE id > 0 OR 1=1"}'
# Returns ALL rows from ALL orgs, not just org 1
```

## Fix

Wrap the original WHERE clause in parentheses:

```go
// Before: WHERE <original> AND org_id = ?
// After:  WHERE (<original>) AND org_id = ?
```

Specifically:
1. Find the WHERE keyword position
2. Extract everything between WHERE and the insertion point (GROUP BY / HAVING / ORDER BY / LIMIT / end)
3. Wrap that content in parentheses
4. Append `AND org_id = ?`

## Verify

1. `go test ./components/api/ -run TestScopeQueryForOrgParens -v` — new test verifying parenthesization ✅
2. `go test ./components/api/ -run TestSQLOrgScoped -v` — existing org-scoped SQL test still passes ✅
3. `go test ./components/api/ -v` — full API test suite passes (48/48) ✅
4. `go vet ./...` — clean ✅

## Fix Applied

`scopeQueryForOrg` now wraps the original WHERE clause in parentheses:

```go
// Before (BUG): WHERE id > 0 OR 1=1 AND org_id = ?
// After (FIX):  WHERE (id > 0 OR 1=1) AND org_id = ?
```

The fix extracts the content between WHERE and the next clause keyword, wraps it in `()`, and appends `AND org_id = ?`. This ensures the org_id filter is always AND-ed with the entire original predicate, not just the last term.
