# Story e43s01: Configurable health check endpoint probing

**type:** feat
**context:** domain
**BCPS:** 2

## Context

Today a process deployment is marked `running` **and its proxy host is registered (traffic switched) before the app process even starts** — `deploy.go:794-804` jumps `building → running` then `go startApp(...)`. If the new build boots broken, traffic is already pointed at a dead port: the classic blind-deploy 502.

This story makes the deploy lifecycle *verify the app actually serves* before going live. After the build, a process deployment enters the (already-defined-but-unused) `deploying` state, the app is started, and a configurable HTTP probe runs against `http://localhost:<PORT><path>` until it passes or exhausts retries. Only on success does the deployment transition to `running` and register its proxy host. On failure it transitions to `failed` and the **previous deployment keeps serving** (no host is registered for the broken build → no downtime).

Decision (plan-work, 2026-06-26): the gate lives in the **deploy orchestrator flow**, not the Supervisor — new process deploys go through `startApp`, not `Supervisor.Run`, and the Supervisor never calls `Instance.Health()`. The e53 `Instance.Health()` stub is left untouched (no caller ⇒ no dead code). Health gating applies to **process apps only**; static-file deploys keep `building → running`.

> **State vocabulary:** the epic text says "live"; the codebase terminal state is `StateRunning` ("running"). "Live" ≡ `running` throughout this plan. `StateDeploying` ("deploying") already exists in `state_machine.go` with valid transitions `building→deploying`, `deploying→running`, `deploying→failed` — this story is its first real user.

## Module Zoom-Out

### components/deploy/manifest.go (Manifest Parser)
- **Purpose:** Load, validate, and convert `bigbase.yaml` into deploy configuration.
- **Callers:** `Deploy.runDeployment` via `LoadManifestPath`.
- **Contracts:** `Manifest{Version, Framework, Build, Start, Env}`; `validate()`. We extend it with an optional `health_check` section (purely additive — absent section ⇒ defaults).

### components/deploy/deploy.go (Deploy Orchestrator)
- **Purpose:** clone → build → run/serve lifecycle; owns the `deployments` table, state machine, and proxy host registration.
- **Callers:** deploy goroutine (`triggerDeploy`/`runDeployment`), Supervisor (resume), Admin UI HTTP endpoints, proxy via `DeploymentHostRegistry`.
- **Contracts:** state-machine valid transitions; `finalizeDeploymentURL` registers the proxy host; `TransitionState(ctx,id,status)`. We re-order: host registration moves to *after* the probe passes.

### components/deploy/health.go (NEW — Health Probe)
- **Purpose:** a pure, injected-dependency HTTP probe with retry/interval/timeout semantics.
- **Reason for Depth:** a clock+client-injected probe is the only way to test retry/timeout/max-retries behavior without real sockets or real `time.Sleep` — mirrors the e53 `FakeClock`/`FakeRunner` testing discipline. Inline code in the orchestrator would be untestable.
- **Contracts:** `probeHealth(ctx, doer httpDoer, baseURL string, cfg HealthCheck, clock Clock) HealthResult`.

## Steps

### 1. Add `health_check` config to the manifest with defaults
**→ verify:** `go test ./components/deploy/ -run TestHealthCheckConfig -v`

- Add to `manifest.go`:
  ```go
  type ManifestHealthCheck struct {
      Path                 string `yaml:"path"`
      ExpectedStatus       int    `yaml:"expected_status"`
      ExpectedBodyContains string `yaml:"expected_body_contains"`
      TimeoutSeconds       int    `yaml:"timeout_seconds"`
      IntervalSeconds      int    `yaml:"interval_seconds"`
      MaxRetries           int    `yaml:"max_retries"`
  }
  ```
- Add `HealthCheck ManifestHealthCheck `yaml:"health_check"`` field to `Manifest`.
- Add `(ManifestHealthCheck).withDefaults() ManifestHealthCheck` applying: `path=/`, `expected_status=200`, `timeout_seconds=30`, `interval_seconds=2`, `max_retries=5` (zero-valued fields only; `expected_body_contains` defaults to empty = no body assertion).
- `TestHealthCheckConfig`: a manifest with no `health_check:` yields all defaults; a partial `health_check:` overrides only the set fields.

### 2. Implement the pure `probeHealth` retry loop in `components/deploy/health.go`
**→ verify:** `go test ./components/deploy/ -run TestProbeHealth -v`

- Define a minimal client seam: `type httpDoer interface { Do(*http.Request) (*http.Response, error) }` (stdlib `*http.Client` satisfies it).
- `type HealthResult struct { OK bool; Attempts int; AvgResponseMS int; FirstFailureReason string; Probes []ProbeAttempt }` where `ProbeAttempt{ Status int; DurationMS int; Err string }`.
- `probeHealth(ctx, doer httpDoer, baseURL string, cfg ManifestHealthCheck, clk Clock) HealthResult`:
  - Loop up to `cfg.MaxRetries` attempts. Each attempt: `GET baseURL+cfg.Path` with a per-request timeout of `cfg.TimeoutSeconds`; record status + duration.
  - Pass = `status == cfg.ExpectedStatus` AND (`cfg.ExpectedBodyContains == ""` OR body contains it). First pass ⇒ `OK=true`, return.
  - On a failing/erroring attempt: record `FirstFailureReason` (first one only), and `clk.Sleep(interval)` before the next attempt (no sleep after the last attempt).
  - Reuse the existing `Clock` interface (`runner.go`) so tests inject `FakeClock` — **no real `time.Sleep`**.
- `TestProbeHealth` (table-driven, `FakeClock` + a fake `httpDoer`): passes on first try; passes on 3rd try after 2×503; exhausts retries on persistent 503 (asserts `OK=false`, `Attempts==max`, `FirstFailureReason` set, and `len(FakeClock.Sleeps)==max-1`); body-mismatch fails even on 200.

### 3. Gate the orchestrator: `building → deploying → (probe) → running|failed` for process apps
**→ verify:** `go test ./components/deploy/ -run TestHealthCheckIntegration -v`

- In `runDeployment` (the `else` process-app branch around `deploy.go:794-804`), replace the unconditional `updateStatus("running")` + `finalizeDeploymentURL` + `go startApp` with:
  1. `d.TransitionState(ctx, deploy.ID, "deploying")`
  2. `go d.startApp(...)` (unchanged — still streams logs and `cmd.Wait()`s)
  3. `result := d.runHealthCheck(ctx, deploy, manifest)` — builds `baseURL=http://localhost:<deploy.Port>`, resolves `manifest.HealthCheck.withDefaults()` (or all-defaults when `manifest==nil`), calls `probeHealth` with the production `*http.Client` and `&wallClock{}`.
  4. On `result.OK`: `d.TransitionState(ctx, deploy.ID, "running")` then `d.finalizeDeploymentURL(deploy, repoName)` (host registered **only now**).
  5. On failure: `d.failDeployment(deploy.ID, ...)` (→ `failed`); do **not** register the host; signal `d.supervisor.Stop(deploy.ID)` / kill the just-started app so the broken build does not linger. The previously-running deployment (its own host already registered) is untouched ⇒ no downtime.
- Static-app branch (`deploy.go:751-757` and the post-build static branch) is unchanged.
- `TestHealthCheckIntegration` (uses the existing test DB + a stub HTTP target): a healthy app reaches `running` and its host is registered; an app that never passes ends `failed`, host **not** registered, and `status_history` shows `building→deploying→failed`.

## Verification Script (Manual)

1. In a scratch repo add a `bigbase.yaml` with `framework: node` and a `health_check: { path: /healthz, expected_status: 200 }`, plus a server that serves `/healthz` 200.
2. `go run .` then deploy the repo through the UI/CLI.
3. Watch the deploy logs: status should pass `building → deploying → running`, and the site should only become reachable after `running`.
4. Replace the server with one that returns 503 on `/healthz`, redeploy: status should end `failed`, the proxy should still serve the previous (healthy) build, and the logs should name the failure.
5. `go test ./components/deploy/ -v` is green.

## Out of scope

- Static-file deploy health gating (process apps only this story).
- Wiring the probe through the Supervisor / making `Instance.Health()` real (no caller today).
- Per-probe log lines, stored health summary, and Admin UI rendering → **e43s02**.
- TCP/exec health checks (HTTP GET only).

## Risks

- **App binds slower than the first probe** — Mitigation: defaults give 5 retries × 2s = 10s grace; `max_retries`/`interval_seconds` are tunable per app.
- **Probe blocks the deploy goroutine** — acceptable: the deploy goroutine is already per-deployment; the probe is bounded by `max_retries × (timeout+interval)`.
- **Re-ordering host registration regresses an existing test** that asserted registration timing — Mitigation: run the full `components/deploy` suite (step 3 verify) and the proxy host-registry tests before merge.
