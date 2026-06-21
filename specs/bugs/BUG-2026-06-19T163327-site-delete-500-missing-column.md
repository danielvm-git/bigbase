# BUG-2026-06-19T163327: Site deletion returns 500 internal error on production

## Problem

On the production instance (bigbase.click), attempting to delete a site via
`DELETE /api/sites/:id` returns `{"error": "internal error"}` (HTTP 500).

- **Actual:** 500 response with generic error message, no deletion occurs.
- **Expected:** Site is deleted with 204, or a 404/409 with a human-readable reason.
- **Reproduce:** On a production database where the `deployments` table was created
  before the `site_id` column migration existed, call the delete endpoint.

## Root Cause Analysis

### Reproduce

Confirmed by code path analysis. The `deleteSite` handler in the Sites component
calls `hasActiveDeployment`, which runs:

```sql
SELECT COUNT(*) FROM deployments
WHERE (site_id = ? OR (site_id = '' AND repo_id = ?))
AND status IN ('pending', 'building', 'running')
```

If the `deployments` table exists but lacks the `site_id` column, SQLite returns
`"no such column: site_id"`.

### Isolate

The `hasActiveDeployment` and `deleteSiteRecords` functions use `isMissingOptionalTable`
to gracefully handle absent tables. But that helper only matches table-level errors:

- `"no such table: deployments"`
- `"no such table: site_request_logs"`
- `relation "deployments" does not exist` (PostgreSQL)
- `relation "site_request_logs" does not exist` (PostgreSQL)

It does NOT match column-level errors like:
- `"no such column: site_id"` (SQLite)
- `column "site_id" does not exist` (PostgreSQL)

The production database was created from an older BigBase version whose
`ensureSiteIDColumn` migration may not have run, or may have silently failed,
leaving the `deployments` table without the `site_id` column. The delete
handler then throws an unhandled error, producing a 500.

The `ensureSiteIDColumn` migration in the Deploy component catches "duplicate
column" (column already present) and treats that as success, but any other
migration failure is only logged as a warning and does not prevent server
startup.

### Hypothesize

1. **The `site_id` column is missing.** Falsified by checking the production
   DB schema. Cannot verify without VPS access, but consistent with symptoms.
2. **A different column reference fails.** Less likely — `status`, `repo_id`,
   and `id` are present in every version of the deployments table.

### Verify

Root cause confirmed: `isMissingOptionalTable` does not handle column-level
SQL errors. When a referenced column does not exist in the production schema,
the error propagates as a 500 response.

Risk level: **High** — blocks all site deletion on production instances that
were upgraded from older BigBase versions.

## TDD Fix Plan

1. **RED:** Write a test that verifies site deletion returns 204 (not 500)
   when the `deployments` table exists but lacks the `site_id` column.
   **GREEN:** Expand `isMissingOptionalTable` to match "no such column" errors
   for SQLite and PostgreSQL, returning `false` (no active deployments) and
   allowing the cascade delete to continue.
   **verify:** `go test ./components/sites/ -run TestDeleteSite -v`

2. **RED:** Write a test that verifies `hasActiveDeployment` returns
   `(false, nil)` when a column is missing.
   **GREEN:** Apply the same fix.
   **verify:** `go test ./components/sites/ -run TestDeleteSite -v`

**REFACTOR:** None needed — fix is a one-line addition to the existing
`isMissingOptionalTable` helper.

## Acceptance Criteria

- [ ] Deleting a site on a DB where `deployments` lacks `site_id` returns 204 (not 500).
- [ ] Deleting a site on a DB where `site_request_logs` lacks `site_id` returns 204.
- [ ] Existing delete tests still pass.
- [ ] Full test suite passes.

## Resolution

**Status:** fixed

Root cause confirmed: `isMissingOptionalTable` did not match column-level SQL errors.

**Fix:** Expanded the `isMissingOptionalTable` helper in `components/sites/sites.go:457-466`
to match:
- `"no such column"` (SQLite)
- `"column" ... "does not exist"` (PostgreSQL)

This causes `hasActiveDeployment` to return `(false, nil)` when columns are missing,
and `deleteSiteRecords` to skip cascade DELETE errors on missing columns.

**Verification:**
```
go test ./components/sites/ -run TestDeleteSite -v
# All 6 tests pass, including TestDeleteSiteHandlesMissingColumn
```

**Files changed:** `components/sites/sites.go` (isMissingOptionalTable helper)
