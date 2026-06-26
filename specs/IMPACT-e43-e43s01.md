# Impact Report — e43s01 (Health Check Probing)

**Type:** Lightweight
**Risk Score:** 5/10
**Decision:** Proceed without grill-me

## Scope
- Add `ManifestHealthCheck` type + `withDefaults()` to `components/deploy/manifest.go`
- New file `components/deploy/health.go` — pure `probeHealth` retry loop
- Re-order registration in `components/deploy/deploy.go:786-804` (building → deploying → probe → running|failed)

## Dependents
- **Manifest struct:** 2 callers — `LoadManifest` (no impact), `ValidateManifest` (no impact since health_check is optional). Extended fields are additive/optional — existing YAML files are unaffected.
- **probeHealth:** No callers yet — created by this story.
- **state_machine.go:** `StateDeploying` already defined with valid transitions `building→deploying`, `deploying→running`, `deploying→failed` — no changes needed.
- **Runner/Instance:** Not touched — health gate happens in deploy orchestrator, not Supervisor.

## Regression Risks
1. **Host registration timing:** Tests that assert `finalizeDeploymentURL` is called for process apps may fail if they assume it's before `startApp`. Mitigation: the probe runs after `startApp` goroutine, not before — `finalizeDeploymentURL` moves to after probe passes.
2. **Static deploy:** Unchanged — keeps `building→running` + immediate host registration.
3. **Existing `running` deployments:** No impact — `health_summary` column will be empty for existing rows; UI code checks presence.

## Test Coverage
- TestHealthCheckConfig — manifest parsing
- TestProbeHealth — retry/backoff/body-match with FakeClock
- TestHealthCheckIntegration — full orchestrator gate via test DB + stub HTTP target
- Existing 56 tests — must remain green

## Stories Affected
- e43s01 (this story) — creates the infrastructure
- e43s02 (next) — consumes HealthResult for logging + UI
