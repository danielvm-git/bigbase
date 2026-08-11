# e86s03 — Org-scoped logs (org_id migration + isolation)

**type:** fix
**risk:** P0
**context:** domain
**BCPs:** 3

## Summary

Close the cross-tenant log leak: `monitoring_logs` has no `org_id`, so any authenticated user can read every tenant's logs via `GET /api/monitoring/logs`. Add `org_id`, scope writes/reads/byID by the authenticated org — the same isolation alerts already have (BUG-143 pattern). This is the foundational story: e86s04 needs org_id at write time.

## Context

Monitoring component purpose: metrics, logs, alerts, health, SSE events. Callers: admin UI (HTTP), proxy middleware, event-bus hooks (`wireObservabilityHooks`). Contracts preserved: all `/api/monitoring/*` routes, `WithEventBus`, `Middleware`, `RecordOrgRequest`. The BUG-143 idempotent-migration pattern is already established in `initObservability` (`ALTER TABLE ... ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0`, tolerate "duplicate column") — copy it verbatim.

## Requirements

#### MODIFIED: Logs are isolated by organization
**Before:** `monitoring_logs` has no `org_id`; any authenticated user queries all tenants' logs (`SELECT ... ORDER BY created_at DESC LIMIT 100` with no org filter). Cross-tenant leak.
**After:** `monitoring_logs.org_id` is populated on insert from `kernel.OrgIDFromContext` (401 `org_id required` when absent, BUG-143 pattern); search/byID filter `WHERE org_id = ?`; rows with NULL org_id (pre-migration backfill) are platform-internal and never returned to tenant orgs.

## Implementation Steps

1. Schema — add `org_id INTEGER` to `monitoring_logs`: include in the `CREATE TABLE IF NOT EXISTS` block (new installs) AND add the idempotent `ALTER TABLE monitoring_logs ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0` guarded by "duplicate column" tolerance, mirroring `initObservability` (monitoring/observability.go ~L85-95). → verify: `go test ./components/monitoring/... -run TestMigrate -v && go build ./...`
2. `handleLogCreate` (monitoring.go:536) — resolve `orgID, ok := kernel.OrgIDFromContext(r.Context())`; 401 `{"error":"org_id required"}` when missing (mirror handleAlerts at monitoring.go:636-640); INSERT `org_id`. → verify: `go test ./components/monitoring/... -run TestLogCreateOrg -v`
3. `handleLogSearch` (monitoring.go:564) — append `AND org_id = ?` to both the plain and `?q=` queries; `handleLogByID` (monitoring.go:599) — scope `WHERE id = ? AND org_id = ?`. → verify: `go test ./components/monitoring/... -run 'TestLogSearch|TestLogByID' -v`
4. Isolation tests (extend `components/monitoring/org_isolation_test.go`, existing pattern): org A inserts a log; org B query returns zero rows; org A sees only its own; NULL-org rows invisible to both; cross-org `handleLogByID` returns 404. → verify: `go test ./components/monitoring/... -run TestOrgIsolationLogs -v`
5. Confirm no regression in `evidence.go` (reads `monitoring_logs` for deploy diagnosis — column-specific SELECT, unaffected by added column). → verify: `go test ./components/monitoring/... -run TestEvidence -v`

## Verification Script (Step-by-Step)

1. Start BigBase locally with two orgs A and B.
2. As org A, POST a log → 201 with id.
3. As org B, GET `/api/monitoring/logs` → empty data.
4. As org A, GET `/api/monitoring/logs` → exactly the org-A row.
5. As org B, GET `/api/monitoring/logs/{orgA-id}` → 404.
6. Unauthenticated GET `/api/monitoring/logs` → 401 `org_id required`.

## Out of scope

- Pagination (e86s01), SSE (e86s02), deploy ingestion (e86s04) — separate stories.
- Retroactive org assignment for existing rows (backfill stays NULL = platform-internal).

## Risks

- SQLite ALTER TABLE on large existing `monitoring_logs` tables — low risk (table is small, log retention is LIMIT-100 oriented); verify migration test covers an existing table with rows.
- Breaking the `handleLogCreate` public contract — POST now requires an authenticated org; the manual log-aggregation use case must send org context (it already must for alerts — consistent).
