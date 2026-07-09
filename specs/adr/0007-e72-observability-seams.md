# ADR 0007 — E72 Observability: Events, Incidents, and Cross-Component Seams

type: adr
context: BigBase monitoring epic e72 — AI-assisted incident response and deploy observability
status: accepted

## Decision

E72 ships four stories on three deep modules and explicit domain entities. We do **not** wait for ADR 0005 Engine decomposition; timeline instrumentation lands in `deploy` with types named for future Engine migration.

### 1. Deployment failure = state + derived event (both)

- **Invariant:** `status = failed` is only reachable via `TransitionState`.
- **Derived event:** `deploy.failed` emits **exactly once** when entering `failed`, carrying `deployment_id`, `site_id`, `repo_id`, `app_type`, `error_message`, `build_log` (tail already persisted).
- **Keep** `deploy.state_changed` for all transitions (existing consumers).
- **Fix** monitoring bus subscriptions: replace dead hook `"deploy"` with `deploy.state_changed`, `deploy.failed`, `alert.triggered`, and existing scaffold/mutation hooks.

### 2. EventRecorder (`components/internal/eventrecorder/`)

FIFO-capped persistent event store behind a small interface:

- `Record(ctx, RecordedEvent) error`
- `Query(ctx, Filter) ([]RecordedEvent, error)`

Monitoring's SSE `eventStream` becomes a fan-out adapter on top of `Record`, not the persistence owner. e72s03 builds the recorder; e72s04 queries it.

### 3. Pipeline timeline ≠ deployment status

- **`PipelineTimeline`** (clone/build/start/health timestamps) is orthogonal to **`DeploymentState`** (pending/building/deploying/running/failed).
- Stored as `pipeline_timeline` JSON column on `deployments` (rename from spec's `stage_timeline` — "stage" collides with state machine language).
- Instrumented in `runDeployment` today; migrates to Engine `Result` when ADR 0005 lands.

### 4. Deploy-owned URLs, monitoring-owned logic (composition-root wiring)

Deploy Gateway registers:

- `GET /api/deploy/:id/diagnosis`
- `GET /api/deploy/:id/related-events`

Deploy does **not** import monitoring. `main.go` injects optional interfaces:

```go
type DeployDiagnosisReader interface {
    GetDiagnosis(ctx context.Context, deployID string) (Diagnosis, bool, error)
}
type DeployRelatedEventsReader interface {
    GetRelatedEvents(ctx context.Context, deployID string) (RelatedEvents, error)
}
```

Nil adapters → 404. Tests use fakes on the deploy Gateway seam.

### 5. Alert Rule vs Alert Incident

- **`AlertRule`** (`monitoring_alerts`) — threshold configuration. Unchanged.
- **`AlertIncident`** (`monitoring_alert_incidents`) — one open incident per rule while breached; new row when breach clears and re-occurs.
- **`alert.triggered`** carries `incident_id` (not just `alert_id`).
- **Invariant:** `emitAlertTriggered` fires at most once per open incident (dedup in checker).
- Investigations FK → `incident_id`. API: `GET /api/monitoring/incidents/:id/investigation` (not rule ID).

### 6. EvidenceGatherer (monitoring-internal deep module)

```go
Gather(ctx, EvidenceScope) (EvidenceBundle, error)
```

Scope = `{SiteID, WindowStart, WindowEnd, Metric}`. Bundles metrics snapshot, recent deployments (SQL on shared DB — existing pattern), eventrecorder query, log tail. LLM summary via `internal/llm` when configured.

### 7. LLM package (`components/internal/llm/`)

OpenAI-compatible HTTP client (DeepSeek default). Prompt kinds (`deploy_failure`, `alert_investigation`) are implementation detail behind `Complete(ctx, Prompt)`.

**Default provider (operator config):**

| Env var | Default | Note |
|---------|---------|------|
| `BIGBASE_LLM_API_KEY` | — | Required to enable AI features; accepts DeepSeek API key |
| `BIGBASE_LLM_BASE_URL` | `https://api.deepseek.com` | OpenAI-compatible; POST `{base}/chat/completions` |
| `BIGBASE_LLM_MODEL` | `deepseek-chat` | Cost-effective for one-shot diagnosis; override with `deepseek-v4-pro` for harder cases |

`DEEPSEEK_API_KEY` is accepted as a fallback when `BIGBASE_LLM_API_KEY` is unset (operator convenience only).

### 8. Event enrichment prerequisite

`mutation` and `request` bus events must include `site_id` when available, or e72s03 correlation returns empty traffic/mutation buckets. Proxy emits `request` events; API enriches `mutation` with org/project context mapped to site where possible.

## Rationale

- Specs assumed `deploy.failed` and `"deploy"` hook — code had neither; fixing bus contracts is prerequisite, not optional.
- Alert investigations keyed to rule IDs would duplicate every 30s (current checker behavior).
- Persistence inside `eventStream.broadcast` fails the deletion test — SSE and storage are separate adapters on one recorder.
- ADR 0005 Engine split is accepted but unimplemented; blocking e72 on it violates minimum-scope delivery.

## Consequences

- e72s03 must land before e72s04; e72s01 needs deploy `deploy.failed` emission (deploy task, not monitoring-only).
- Admin UI routes investigations by `incident_id`; alert rule pages link to latest open incident.
- `GET /api/monitoring/alerts/:id/investigation` from original spec is **superseded** by incident-scoped endpoint.

## Related

- ADR 0005 (Engine decomposition) — `PipelineTimeline` field moves to `Engine.Result` later; column name unchanged.
- e72 epic capsule: `specs/epics/e72-monitoring-enhancements/`
