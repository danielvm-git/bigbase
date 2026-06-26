# Story e43s02: Health check failure handling and reporting

**type:** feat
**context:** domain
**BCPS:** 1

## Context

e43s01 gates the deploy on a health probe but reports almost nothing: a failed deploy just lands in `failed` with a generic message. This story makes the probe **observable** — every attempt is logged to the deploy log, a structured summary is persisted on the deployment row, and the Admin UI surfaces a health step on the deployment detail view so a user can see *what* the app returned and *how fast*.

Depends on e43s01 (the `HealthResult` / `ProbeAttempt` types and the orchestrator gate already exist). This story consumes them.

## Module Zoom-Out

### components/deploy/deploy.go (Deploy Orchestrator)
- **Purpose / Callers / Contracts:** as in e43s01. New here: a `health_summary` column on `deployments` (additive migration, mirrors the existing `status_history`/`error_message` migration idiom at `deploy.go:1178-1250`), surfaced through the deployment status JSON the Admin UI already consumes.

### ui/src/pages/DeployPage.tsx (Admin UI — Deployment View)
- **Purpose:** render deployment status + logs.
- **Callers:** Admin UI router; reads the deployment status API.
- **Contracts:** the deployment JSON shape it renders — we add an optional `health_summary` field; absent ⇒ nothing rendered (back-compatible with pre-e43 deployments).

## Steps

### 1. Log each probe attempt and persist a health summary
**→ verify:** `go test ./components/deploy/ -run TestHealthCheckReporting -v`

- In `runHealthCheck` (e43s01), append a deploy-log line per attempt, e.g. `→ Health: GET /healthz → 200 (12ms)` / `→ Health: GET /healthz → 503 (timeout)`; on the final failure also append the response body (truncated) so the user sees what went wrong.
- Add migration `ensureHealthSummaryColumn()`: `ALTER TABLE deployments ADD COLUMN health_summary TEXT DEFAULT ''` (follow the `ensureManifestPathColumn` pattern; register it alongside the other `ensure*` calls in `Start`).
- After the probe, marshal `HealthResult` → JSON `{probe_count, avg_response_time_ms, first_failure_reason}` and write it to `health_summary` for the deployment (on both pass and fail).
- `TestHealthCheckReporting`: after a failing deploy, the row's `health_summary` JSON has `probe_count == max_retries`, non-empty `first_failure_reason`, and a numeric `avg_response_time_ms`; the deploy log contains a per-attempt line.

### 2. Expose `health_summary` in the deployment status JSON
**→ verify:** `go test ./components/deploy/ -run TestHealthSummaryInStatusAPI -v`

- Add `HealthSummary json.RawMessage `json:"health_summary,omitempty"`` (or a typed struct) to the `Deployment` struct, and include the column in the `SELECT` used by the status/detail handler.
- `TestHealthSummaryInStatusAPI`: the deployment detail JSON returned by the handler includes `health_summary` for a probed deployment and omits it for a legacy row (empty string).

### 3. Admin UI — health step on the deployment detail page
**→ verify:** `cd ui && npm run build && npm test -- DeployPage`

- In `DeployPage.tsx`, when `deployment.health_summary` is present, render a "Health check" step in the deployment timeline: ✓ for a passed probe set, ✗ for failure (show `first_failure_reason`), and show `probe_count` and `avg_response_time_ms`. Grey/absent when there is no summary.
- Extend `DeployPage.test.tsx` with a case asserting the health step renders the reason on failure and the avg latency on success.

## Verification Script (Manual)

1. Deploy a process app whose `/healthz` returns 503 (from the e43s01 manual script).
2. Open the deployment detail page in the Admin UI: a red "Health check" step shows `first_failure_reason` and the probe count.
3. Fix the app so `/healthz` returns 200, redeploy: the step turns green and shows the average response time.
4. `go test ./components/deploy/ -v` and `cd ui && npm run build` are green.

## Out of scope

- Response-time **chart** across probes (a single avg + per-attempt log lines suffice; a chart is gold-plating for ≤5 probes).
- Alerting/notifications on health failure (belongs with e44 rollback / messaging).
- Historical health trends across deployments.

## Risks

- **`health_summary` JSON shape drift between Go and the UI** — Mitigation: keep the three documented keys stable; `omitempty` + optional UI render keeps old rows safe.
- **UI test harness coupling** — Mitigation: gate the UI step on presence of `health_summary` so existing `DeployPage.test.tsx` cases are unaffected.
