# e86s04 — Deploy-component automatic log ingestion

**type:** feat
**risk:** P1
**context:** domain
**BCPs:** 3

## Summary

Logs currently exist in `monitoring_logs` only via manual `POST /api/monitoring/logs`. Make deploy lifecycle events auto-ingest as log rows so deploy activity appears in the Logs tab without manual posting. The subscription surface already exists: `knownEventHooks` (observability.go:45) includes `deploy.state_changed`, `deploy.failed`, `deploy.diagnosed`, and `wireObservabilityHooks` (observability.go:100+) already subscribes to `deploy.failed` at priority 500. This story adds the log-insertion seam to those handlers and enriches deploy events with org context.

## Context

Deploy component emits: `deploy.state_changed` (state_transitions.go:55 — `deployment_id`, `from_state`, `to_state`, `site_id`) and `deploy.failed` (observability.go:87 — `deployment_id`, `site_id`, status). Neither event currently carries `org_id` — the event data must be enriched (state_transitions.go + observability.go emit paths) so monitoring can scope the inserted row (requires e86s03's org_id column). Site→org resolution happens at emit time in deploy (it has the site/org context) — prefer enrich-at-emit over a monitoring-side sites-table lookup (keeps monitoring decoupled).

## Requirements

#### ADDED: Deploy lifecycle events auto-ingest as log rows
`deploy.state_changed` (started/succeeded transitions) inserts a `level=info` row; `deploy.failed` inserts `level=error`. Message is a structured summary: `deploy <state>: site=<slug> deployment=<id>`. Rows carry `org_id` from the enriched event and broadcast to the live log stream (e86s02). Manual `POST /api/monitoring/logs` keeps working.

#### MODIFIED: Deploy lifecycle events carry org context
**Before:** `deploy.state_changed` / `deploy.failed` event payloads contain `deployment_id`, `site_id`, `from_state`/`to_state` — no `org_id`.
**After:** Both event payloads include `org_id` resolved from the emitting handler's context (site → org), so downstream consumers (monitoring ingestion) can scope rows.

## Implementation Steps

1. Recon/verify — confirm the deploy emit paths (state_transitions.go:54-66, observability.go:87-94) have site context; find the site→org lookup available at emit time. If events lack org at emit, resolve via the deploy component's sites seam. → verify: `go test ./components/deploy/... -run 'TestStateTransition|TestFailedEvent' -v`
2. Deploy — enrich `deploy.state_changed` (state_transitions.go) and `deploy.failed` (observability.go) event Data with `org_id`; add/adjust tests asserting the field is present. → verify: `go test ./components/deploy/... -run 'TestStateTransition|TestFailedEvent' -v`
3. Monitoring — extend `wireObservabilityHooks` (observability.go:100): add a `deploy.state_changed` subscription (priority ~600) whose handler inserts an info row; extend the existing `deploy.failed` handler (priority 500, runs `onDeployFailed`) with a sibling `insertDeployLog` that inserts an error row. Both read `org_id`/`site_id`/`deployment_id` from `ev.Data`; broadcast via `broadcastLog` (e86s02). → verify: `go test ./components/monitoring/... -run TestDeployIngest -v`
4. Ingestion tests (new `deploy_ingest_test.go` in monitoring): emit `deploy.state_changed` success → info row with org_id; emit `deploy.failed` → error row; live SSE subscriber receives the row; manual POST endpoint unaffected. → verify: `go test ./components/monitoring/... -run 'TestDeployIngest|TestLogStream|TestLogCreate' -v`
5. E2E smoke — run a real deploy against local BigBase, assert lifecycle rows appear in the Logs tab (exercises s01 pagination + s02 stream + s03 org scope together). → verify: `cd ui && npm run build && go build -o /tmp/bigbase-check . && go test ./components/deploy/... ./components/monitoring/...`

## Verification Script (Step-by-Step)

1. Deploy any repo locally (or use a test that drives the state machine).
2. Open Monitoring → Logs tab → info rows appear for started/succeeded transitions.
3. Force a failure (bad build) → an error-level row appears.
4. Confirm the rows carry the deploying user's org (org B never sees org A's deploy logs).
5. Live stream delivers the rows to an open Logs tab.

## Out of scope

- Ingesting cici/forge/function events — deploy lifecycle only (extensible via the same seam).
- Log rotation/retention policy changes — existing LIMIT-100 reader semantics unchanged.

## Risks

- Event volume: every deploy transition becomes a log row — fine at current scale (per-deployment, not per-log-line; build output stays in deploy's own WebSocket stream, not monitoring_logs).
- org enrichment correctness: a missing org_id at emit → row inserted with NULL (platform-internal, invisible to tenants) — the ingest handler must log a warning when org_id is absent so gaps surface.
- Double-write ordering: ingestion inserts must not block the event-bus dispatch — run inserts in a goroutine like the existing `onDeployFailed` (observability.go:111).

## Acceptance Criteria

- [ ] deploy.state_changed success → info row; deploy.failed → error row
- [ ] Rows org-scoped; visible only to the owning org
- [ ] Live subscribers receive ingested rows
- [ ] Manual POST endpoint unchanged
