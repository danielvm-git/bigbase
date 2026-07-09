# e72s04: Alert → Automated Investigation Trigger
## Story ID: e72s04 | Epic: e72 | BCPs: 3 | Status: planned

## 1. Type
**feat** · domain

## 2. Context
The monitoring component's `alert_checker.go` already evaluates alert rules every 30 seconds and emits `alert.triggered` events when thresholds are breached. Currently, these events are logged and emitted on the event bus — but nothing consumes them. No investigation is triggered.

Tracer-Cloud's opensre demonstrates the value of automated investigation: when an alert fires, the system gathers correlated evidence (logs, metrics, recent changes) and presents a unified view. This story adds a lightweight investigation flow triggered by BigBase alerts.

## 3. Problem / Opportunity
- Alert events are emitted but ignored — no automated response
- Platform operators must manually check metrics when an alert fires
- New Relic provides APM but doesn't correlate BigBase's internal state (recent deploys, DB changes, traffic)
- Adding automated investigation turns alerts from "something is wrong" into "here's what changed"

## 4. Proposed solution (ADR 0007)

**AlertRule** (existing `monitoring_alerts`) ≠ **AlertIncident** (new `monitoring_alert_incidents`). One open incident per rule; deduplicated `alert.triggered` emissions.

1. When `alert.triggered` fires with `incident_id`, spawn investigation goroutine
2. `EvidenceGatherer.Gather` collects:
   - **Current metrics snapshot** (CPU, memory, disk, goroutines — already available)
   - **Recent deployments** (query `deployments` table for last 30 minutes)
   - **Recent monitoring events** (from e72s03's `monitoring_events` table)
   - **Recent log entries** (from `monitoring_logs` table, last 100)
3. Packages into `InvestigationReport`, stores in `monitoring_investigations` keyed by **`incident_id`**
4. Emits `alert.investigation_complete` with `incident_id`
5. Exposes via `GET /api/monitoring/incidents/:id/investigation`

If `BIGBASE_LLM_API_KEY` is configured (from e72s01), also generates an AI summary of the investigation findings.

## 5. Alternatives considered
- **Full opensre integration**: Cross-language, heavy-weight. Rejected — we're building a focused Go-native equivalent.
- **Only store raw data, no summary**: Less useful for operators. The value is in correlation.
- **Real-time streaming investigation**: Ambitious for v1. Snapshot-on-trigger is simpler and sufficient.

## 6. Who are the users?
- **Platform operators** receiving alert notifications
- **On-call engineers** who need context when paged

## 7. Dependencies
- `monitoring` component (alert_checker, event bus, host metrics, system metrics)
- `deploy` component (deployments table for recent deploy queries)
- `monitoring_events` table (from e72s03)
- `monitoring_logs` table (existing)
- `components/internal/llm/` (from e72s01, optional — for AI summary)

## 8. Assumptions
- 30 minutes is a reasonable investigation window for most alerts
- Deploy component's DB is accessible for querying recent deployments
- Investigation data is small enough for SQLite storage (< 10KB per report)

## 9. Risks
| Risk | Probability | Impact | Mitigation |
|------|-----------|--------|------------|
| Investigation table grows unbounded | Medium | Low | Cap at 1000 rows, FIFO cleanup |
| Investigation takes too long | Low | Medium | 10s timeout on all queries; goroutine, non-blocking |
| Cross-component DB access (monitoring → deploy tables) | Low | Medium | Use shared DB reference (already pattern in system) |

## 10. Non-goals
- Automated remediation (rolling back deploys, scaling resources)
- Multi-step investigation with branching logic
- Integration with external incident management (PagerDuty, Opsgenie)

## 11. Migration plan
New `monitoring_investigations` table — no migration needed for existing data.

## 12. Wireframes / Diagrams
```
alert.triggered event
       │
       ▼
┌──────────────────────────────────────────────┐
│  Investigation Goroutine (non-blocking)       │
│                                               │
│  1. Snapshot: SystemMetrics + HostMetrics     │
│  2. Query: Recent deployments (30 min)        │
│  3. Query: Recent monitoring_events (30 min)  │
│  4. Query: Recent monitoring_logs (100 rows)  │
│  5. Optional: LLM summary (if API key set)    │
│                                               │
│  → Store investigation report                 │
│  → Emit alert.investigation_complete          │
└──────────────────────────────────────────────┘
```

## 13. API / Data Model
```
GET /api/monitoring/incidents/:id/investigation
→ 200 {
  "alert_id": "alert123",
  "triggered_at": "2026-07-08T12:00:00Z",
  "snapshot": {
    "system": { "cpu_percent": 87.5, ... },
    "host": { "mem_used_bytes": 8589934592, ... }
  },
  "recent_deployments": [ ... ],
  "recent_events": [ ... ],
  "recent_logs": [ ... ],
  "ai_summary": "CPU spike correlates with deployment of api-backend at 11:58. Build process consumed high CPU. No memory pressure detected." // optional
}
→ 404 if no investigation
```

### New DB table
```sql
CREATE TABLE IF NOT EXISTS monitoring_investigations (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL,
  report_json TEXT NOT NULL,
  ai_summary TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (incident_id) REFERENCES monitoring_alert_incidents(id)
);
```

## 14. Affected code
| File | Change |
|------|--------|
| `components/monitoring/alert_checker.go` | After emitAlertTriggered, spawn investigation goroutine |
| `components/monitoring/investigation.go` | **NEW** — InvestigationReport struct, gather evidence, store, emit |
| `components/monitoring/investigation_test.go` | **NEW** — unit tests for evidence gathering |
| `components/monitoring/monitoring.go` | Add `ensureInvestigationTable` migration + `handleAlertInvestigation` endpoint |
| `components/deploy/deploy.go` | Expose `RecentDeployments(siteID string, since time.Time)` query (or query via DB directly) |

## 15. Testing strategy
- **Unit**: Mock deployments table, verify investigation gathers correct evidence
- **Unit**: Test investigation storage and retrieval
- **Integration**: Trigger alert → verify investigation stored → verify API returns it
- **Integration**: Test with and without LLM API key configured

## 16. Rollback plan
Drop `monitoring_investigations` table. Alert checker continues emitting events without investigation.

## 17. Acceptance Criteria
```gherkin
Scenario: Investigation triggered on alert
  Given an alert rule for cpu_percent > 80 is active
  And CPU usage exceeds 80% for the alert's duration_seconds
  When the alert checker evaluates the rule
  Then an investigation is stored in monitoring_investigations
  And GET /api/monitoring/alerts/:id/investigation returns correlated evidence

Scenario: AI summary when LLM configured
  Given BIGBASE_LLM_API_KEY is configured
  And an alert triggers an investigation
  Then the investigation report includes an ai_summary field

Scenario: No AI summary without LLM
  Given BIGBASE_LLM_API_KEY is not configured
  And an alert triggers an investigation
  Then the investigation report has ai_summary: null
```

## 18. Verification Script (for manual UAT)
1. Set `BIGBASE_LLM_API_KEY` if desired
2. `go run . serve --port 9999 --db :memory:`
3. Create an alert: `curl -X POST /api/monitoring/alerts -d '{"name":"High CPU","metric":"cpu_percent","threshold":1,"operator":"gt","enabled":true,"duration_seconds":30}'`
4. Run CPU-intensive workload to trigger alert
5. Wait 30s for alert checker + investigation
6. `curl http://localhost:9999/api/monitoring/alerts/<id>/investigation`
7. Verify snapshot, recent deployments, recent events present

## 19. Out of scope
- Automated remediation actions
- PagerDuty/Opsgenie integration
- Investigation retention policies beyond 1000-row FIFO
- Custom investigation playbooks (v1 is fixed evidence set)
