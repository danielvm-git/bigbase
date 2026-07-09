# e72s03: Correlated Deployment Event Timeline
## Story ID: e72s03 | Epic: e72 | BCPs: 2 | Status: planned

## 2. Context
The monitoring component already subscribes to 6 event bus hooks (`mutation`, `request`, `deploy`, `scaffold_db`, `scaffold_repo`, `scaffold_function`) and broadcasts them via SSE at `/api/monitoring/events`. When a deployment fails, users must manually correlate what happened — recent commits, DB migrations, dependency changes. This story adds a `/api/deploy/:id/related-events` endpoint that queries monitoring events correlated to a deployment by time window and site.

## 3. Problem / Opportunity
- Deployment failures are rarely caused by the deploy itself — usually a recent change
- No mechanism to get "what changed recently" context for a failed deploy
- The event bus already captures relevant events — just not queryable per-deployment
- Reduces time-to-resolution for deployment incidents

## 4. Proposed solution (ADR 0007)

Add `components/internal/eventrecorder/` — deep module with `Record` + `Query`. Monitoring SSE `eventStream` is a fan-out adapter on top of Record.

1. Bus handlers persist every subscribed hook to `monitoring_events` (FIFO 5,000)
2. On `deploy.failed`, snapshot correlated events into `deployments.related_events_snapshot` (30-min pre-deploy window, site_id filter)
3. `GET /api/deploy/:id/related-events` served by deploy Gateway via injected `DeployRelatedEventsReader`

**Prerequisite:** proxy must emit `request` events; api must include `site_id` on `mutation` events — today neither exists for correlation.

## 5. Alternatives considered
- **Query events table directly on demand**: Simpler but requires persistent event storage. Current events are SSE-only (no persistence). Adds DB table.
- **GitHub API for recent commits**: Adds external dependency. Event bus is internal and already captures relevant signals.
- **Do nothing**: Users manually check git log + DB schema. Works for experts, not for the platform experience.

## 6. Who are the users?
- **Developers** debugging failed deployments
- **Platform operators** triaging user reports

## 7. Dependencies
- `monitoring` component (event bus subscription, SSE events)
- `deploy` component (deployment lifecycle)
- `monitoring` DB table `monitoring_events` (NEW — for persistent event storage)

## 8. Assumptions
- Events within 30 minutes of deploy start are the most relevant
- A lightweight SQLite table for recent events is acceptable (5000-row cap, FIFO)
- Site ID is available in the deploy spec for correlation

## 9. Risks
| Risk | Probability | Impact | Mitigation |
|------|-----------|--------|------------|
| Event table grows unbounded | Low | Medium | Cap at 5000 rows, FIFO cleanup |
| Too many events → overwhelming response | Medium | Low | Limit to 20 per category, prioritize by recency |
| Event bus doesn't carry enough context | Low | Medium | Add site_id and deploy_id to relevant event emissions |

## 10. Non-goals
- Full audit log (use monitoring_logs for that)
- Cross-deployment trend analysis
- Event retention beyond 5000 rows

## 11. Migration plan
New `monitoring_events` table — no migration needed for existing deployments (returns empty events array).

## 12. API / Data Model
```
GET /api/deploy/:id/related-events
→ 200 {
  "deploy_id": "abc123",
  "window_start": "2026-07-08T11:30:00Z",
  "window_end": "2026-07-08T12:00:00Z",
  "events": {
    "db_schema_changes": [
      { "hook": "scaffold_db", "data": {...}, "timestamp": "..." }
    ],
    "repo_changes": [...],
    "data_mutations": [...],
    "traffic": [...]
  }
}
→ 404 if deployment not found
```

## 13. Affected code
| File | Change |
|------|--------|
| `components/monitoring/events.go` | Add persistent event storage (SQL insert on broadcast) |
| `components/monitoring/monitoring.go` | Add `ensureMonitoringEventsTable` migration |
| `components/monitoring/monitoring.go` | Add `handleRelatedEvents` endpoint |
| `components/monitoring/monitoring_test.go` | Test event persistence + correlation query |

## 14. Testing strategy
- **Unit**: Test event persistence (insert + query by time window)
- **Unit**: Test correlation logic (filter events by site_id, time window)
- **Integration**: Deploy → fail → GET related-events → verify recent events present

## 15. Rollback plan
Drop `monitoring_events` table. Endpoint returns empty events. No functional impact.

## 16. Acceptance Criteria
```gherkin
Scenario: Related events on deploy failure
  Given a site has recent scaffold_db events
  And a deployment fails for that site
  When GET /api/deploy/:id/related-events is called
  Then the response includes the recent scaffold_db events
  And events outside the time window are excluded

Scenario: Empty related events for new site
  Given a site has no prior events
  When GET /api/deploy/:id/related-events is called
  Then the response returns empty arrays for all categories
```

## 17. Verification Script (for manual UAT)
1. `go run . serve --port 9999 --db :memory:`
2. Make a DB schema change via API
3. Deploy a repo to a site
4. `curl http://localhost:9999/api/deploy/<id>/related-events`
5. Verify the schema change event appears

## 18. Out of scope
- Cross-deployment diff (what changed between deploy N and N-1)
- Event retention policies beyond FIFO cap
- UI visualization of the correlated timeline
