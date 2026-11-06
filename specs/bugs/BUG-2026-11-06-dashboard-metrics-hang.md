# BUG-2026-11-06 — Admin dashboard hangs on "Loading dashboard..." (metrics state-age query unbounded)

**type:** bug
**context:** monitoring, ui
**severity:** high
**priority:** critical
**status:** fixed

## Summary
v2.93.0 (commit `06f86d39d`, "feat(monitoring): expose non-terminal state age gauge and alerts")
made `/api/monitoring/metrics` run `stateAgeReport` against the `deployments` and `cici_runs`
tables using the **unbounded request context** (no timeout). The admin `DashboardPage` awaits
`fetchMonitoringMetricsWarmed` (which hits that endpoint) before its `setLoading(false)` runs,
so a slow or write-locked SQLite connection left the page stuck on "Loading dashboard..."
indefinitely — the reported "the page wasn't loading" regression that forced the v2.93.0 rollback.

## Root cause
`components/monitoring/monitoring.go` `handleMetrics`/`handlePrometheusMetrics` called
`m.stateAgeReport(r.Context())` — the request context has no deadline. The alert checker
already bounded the same query with `stateAgeQueryTimeout = 2s` (`stateAgeReportForAlerts`);
the metrics endpoints did not.

## Fix
`fix(monitoring): bound state-age query in metrics endpoints` (commit `8f320e32b`):
- Renamed `stateAgeReportForAlerts` → `stateAgeReportTimed` and used it in both metrics
  endpoints, so the state-age query is bounded by `stateAgeQueryTimeout` (2s) everywhere.

## Evidence
- `go test -run TestStateAgeReportTimedBoundsBlockedQuery ./components/monitoring` — PASS
  (a fake DB blocks `QueryContext` until context cancellation; the report returns bounded).
- `go test ./components/monitoring` — PASS
- `go test ./...` — PASS
