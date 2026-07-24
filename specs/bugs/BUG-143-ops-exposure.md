---
bug_id: BUG-143
status: fixed
severity: high
scope: monitoring
title: "Cross-Tenant Operational Data Exposure (monitoring/alerts/incidents/events)"
created: 2026-07-23
github_issue: 143
---

# BUG-143: Cross-Tenant Operational Data Exposure

## Summary

`/api/monitoring/alerts`, `/api/monitoring/incidents`, and SSE `/api/monitoring/events` endpoints require only authentication (JWT/API key) but perform **no org_id scoping**. Any authenticated user can see every tenant's:
- Alert rules
- Alert incidents (including investigation reports)
- Live SSE event stream (deployments, mutations, logs)

## Affected Files

| File | Lines | Vulnerability |
|------|-------|--------------|
| `components/monitoring/monitoring.go` | `handleAlertList` | Queries all `monitoring_alerts` without `org_id` filter |
| `components/monitoring/monitoring.go` | `handleAlertCreate` | Inserts alert without `org_id` |
| `components/monitoring/observability.go` | `handleIncidents` | Queries all `monitoring_alert_incidents` without `org_id` filter |
| `components/monitoring/observability.go` | `handleIncidentInvestigation` | Queries all `monitoring_investigations` without `org_id` filter |
| `components/monitoring/events.go` | `handleSSEEvents` | Streams ALL events without org_id filtering |

## Root Cause

### Database Schema (Missing `org_id`)

The monitoring tables were created without an `org_id` column:

```sql
-- monitoring_alerts (monitoring.go:153)
CREATE TABLE IF NOT EXISTS monitoring_alerts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    metric TEXT NOT NULL,
    threshold REAL NOT NULL DEFAULT 0,
    operator TEXT NOT NULL DEFAULT 'gt',
    enabled INTEGER NOT NULL DEFAULT 1,
    duration_seconds INTEGER NOT NULL DEFAULT 300
    -- MISSING: org_id INTEGER NOT NULL
)

-- monitoring_alert_incidents (observability.go:66)
CREATE TABLE IF NOT EXISTS monitoring_alert_incidents (
    id TEXT PRIMARY KEY,
    rule_id TEXT NOT NULL,
    metric TEXT NOT NULL,
    value REAL NOT NULL DEFAULT 0,
    threshold REAL NOT NULL DEFAULT 0,
    operator TEXT NOT NULL DEFAULT 'gt',
    opened_at TEXT NOT NULL DEFAULT (datetime('now')),
    resolved_at TEXT DEFAULT NULL,
    triggered INTEGER NOT NULL DEFAULT 0
    -- MISSING: org_id INTEGER NOT NULL
)
```

### Handler Code (No Scoping)

**handleAlertList** (monitoring.go):
```go
rows, err := m.db.QueryContext(r.Context(),
    "SELECT id, name, metric, threshold, operator, enabled, duration_seconds FROM monitoring_alerts ORDER BY name")
// No WHERE org_id = ? filter
```

**handleIncidents** (observability.go:322-326):
```go
query := `SELECT id, rule_id, metric, value, threshold, operator, opened_at, resolved_at FROM monitoring_alert_incidents`
if status == "open" {
    query += ` WHERE resolved_at IS NULL`
}
// No AND org_id = ? filter
```

**handleSSEEvents** (events.go):
```go
// Streams all events from m.stream without filtering by org_id
id, ch := m.stream.subscribe()
defer m.stream.unsubscribe(id)
```

## Precedent: `/api/sql` Endpoint

The `/api/sql` endpoint properly scopes queries by `org_id` using:
```go
orgID, ok := auth.OrgIDFromContext(r.Context())
if !ok {
    kernel.WriteJSON(w, http.StatusUnauthorized, ...)
    return
}
// Query with: WHERE org_id = ?
```

## Fix Approach

### 1. Add `org_id` column via migration
```sql
ALTER TABLE monitoring_alerts ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE monitoring_alert_incidents ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE monitoring_investigations ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0;
```

### 2. Update INSERT statements
```go
// handleAlertCreate
_, err := m.db.ExecContext(r.Context(),
    "INSERT INTO monitoring_alerts (id, name, metric, threshold, operator, enabled, duration_seconds, org_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
    id, alert.Name, alert.Metric, alert.Threshold, alert.Operator, enabled, alert.DurationSeconds, orgID)
```

### 3. Add WHERE clause to all SELECT queries
```go
// handleAlertList
rows, err := m.db.QueryContext(r.Context(),
    "SELECT id, name, metric, threshold, operator, enabled, duration_seconds FROM monitoring_alerts WHERE org_id = ? ORDER BY name", orgID)

// handleIncidents
query := `SELECT ... FROM monitoring_alert_incidents WHERE org_id = ?`
args := []any{orgID}
if status == "open" {
    query += ` AND resolved_at IS NULL`
}
```

### 4. Scope SSE event stream
Add `org_id` field to events and filter in the stream handler.

## Verification Steps

1. **Unit test**: Create alerts for org A and org B, verify org A user only sees org A alerts
2. **Unit test**: Create incidents for org A and org B, verify org A user only sees org A incidents
3. **Integration test**: Verify SSE stream only delivers events for the caller's org
4. **Security test**: Verify regular user cannot access other tenant's data (cross-tenant isolation)
5. **Migration test**: Verify existing data gets `org_id = 0` (default org) after migration

## Impact

- **Severity**: HIGH — any authenticated user can enumerate all tenants' operational data
- **Scope**: monitoring/alerts, monitoring/incidents, monitoring/events SSE stream
- **Data exposed**: Alert rules, incident details, investigation reports, live deployment events

## References

- GitHub Issue: #143
- Security review 2026-07-23 — Finding #15
- Precedent fix: BUG-132 IDOR injectDB (scope db.collection() queries by org_id)
