# ADR 005 — Deploy Component Decomposition: Engine + Gateway + Orchestrator

type: adr
context: BigBase deploy component — split 1819-line monolith into three modules separated by when code runs

## Status

Accepted — **Implemented** (e45 cycle, July 2026).
`deploy.go` reduced from 1875 → 342 lines. Engine, Gateway, and Orchestrator
extracted to separate files. See `components/deploy/engine.go`, `gateway.go`,
`orchestrator.go`.

## Principle

**The Deploy component must split along the same seam every comparable platform uses:**
request-time (HTTP handlers) vs background (per-deployment execution) vs startup-time
(fleet management). A single struct with 40+ methods violates CONVENTIONS (files < 300
lines, one responsibility per module, functions 4–20 lines). The split uses the Go
optional-interface pattern (`database/sql/driver`) — minimal required, optional
enhancements — and is informed by the architectures of Appwrite (Controllers vs Workers
vs Tasks), Supabase (Gateway vs Services), Neon (Proxy vs Control Plane vs Compute), and
PocketBase (core.App + hooks + tools/cron).

## The Modules

### Engine (background — per-deployment lifecycle)

- **Interface:** `Runner` — `Run(ctx, spec) → (Result, error)`. One blocking call per
  deployment. Optional: `LogStreamer` (`RunWithLogs`), `Builder` (`Build`).
- **Hides:** clone, build (npm/go/pip), node_modules cache, start app/serve static,
  health probe, host registration, supervisor.Run.
- **Depth:** 4. One entry point hides 6 concerns. Every caller (HTTP, webhook, MCP,
  sites, resume-on-boot) crosses the same seam.
- **Test:** `FakeRunner` + temp dirs + in-memory cache. No exec.Cmd, no HTTP server.

### Gateway (request-time — HTTP API)

- **Interface:** `Handler() → http.Handler`. Standard Go HTTP mux with CRUD endpoints.
- **Delegates:** all deployment work to `engine.Run(spec)`. Zero process management.
- **Depth:** 3. 5 endpoints through one `Handler()` method.
- **Test:** `httptest.NewRecorder` + `FakeEngine` returning scripted Results. Assert on
  HTTP status codes and JSON bodies (CONVENTIONS T8: observable outcomes, never internal
  state).

### Orchestrator (startup-time — fleet management)

- **Interface:** `Resume(ctx)`, `Drain(id)`, `Rollback(from, to)`, `DeleteSiteDeployments(ctx, siteID, repoID)`.
- **Hides:** DB queries for candidates, host restoration, drain timeout, status transitions.
- **Depth:** 3. 4 fleet operations through one public API.
- **Test:** `FakeEngine` + `FakeDB`. Assert Engine receives correct Specs.

### Composition Root (deploy.go)

- ~150 lines (down from 1819). Implements `kernel.Component`. Wires Engine + Gateway +
  Orchestrator. Owns ECC lifecycle (Init/Start/Stop). No process management, no log
  capture, no HTTP routing — delegates everything.

## Rationale

### Prior art

Every comparable platform uses this seam:

| Platform | Request-time | Background | Startup-time |
|----------|-------------|------------|--------------|
| Appwrite | `Http/` controllers | `Workers/` | `Module.php` + Tasks |
| Supabase | Kong Gateway | GoTrue/Realtime/Storage | Service bootstrap |
| Neon | Proxy (auth+routing) | Control plane (lifecycle) | Compute init |
| PocketBase | `se.Router.GET` handlers | `app.Cron()` | `OnServe().BindFunc` hooks |
| **BigBase** | **Gateway** | **Engine** | **Orchestrator** |

### Counterfactual

Without this decomposition, the deploy component continues as a 1819-line monolith
where testing any behavior requires a full Deploy with DB, git dir, builds dir, host
registry, and env key — even for pure logic like state transitions or URL construction.
Build caching bugs affect `resumeCandidates`, `runDeployment`, and `startApp`
independently. Adding a new deployment trigger (e.g., CLI) requires copy-pasting the
clone→build→start→supervise sequence.

## Quick Wins (pre-requisite)

1. **Hoist Logger to kernel** — 18 duplicate `type Logger interface` + 19 duplicate
   `type noopLogger struct{}` deleted. All components use `kernel.Logger` +
   `kernel.NopLogger`.
2. **Unify DBer to kernel.DBer** — `mcp`, `backup`, `webhooks` change from their own
   `DBer` to `type DBer = kernel.DBer`.
3. **Extract migrations** — 10 `ensure*Column()` calls move to `migrateDeploySchema(db)`.
4. **Surface stateMachine** — move to `internal/deploystate/` as exported `Machine`.

## Consequences

- The deploy package gains 3 new files (`engine.go`, `gateway.go`, `orchestrator.go`)
  but the total line count decreases (~2500 → ~1800) because old test boilerplate
  (building a full Deploy just to test one method) vanishes.
- `deploy_test.go` shrinks from 2342 lines to ~1200 lines. Gateway tests use
  `httptest.NewRecorder` (standard library). Engine tests use fakes. No test needs a
  real `exec.Cmd` except integration tests.
- Callers (`sites`, `mcp`, `webhooks`, `main.go`) continue to use the same `Deploy`
  struct and `Trigger()` method — the interface they already depend on. Only the
  `main.go` wiring changes (injecting Engine + FakeEngine for tests).
- The `Runner` interface expands from `Spawn` to `Run` (full lifecycle). The existing
  `Supervisor` and `FakeRunner` adapt naturally.
- This ADR does not change any external API, CLI flag, or event bus contract. It is a
  pure internal refactor.

## Related

- ADR 0004 (no containers, in-process Go Supervisor) — Engine wraps Supervisor, doesn't
  replace it.
- CONVENTIONS.md (files under 300 lines, one responsibility per module, functions 4–20
  lines, dependencies injected through constructor, tests through public interfaces).
- Prior art: Appwrite CONTRIBUTING.md, Supabase architecture.mdx, Neon docs/glossary.md,
  PocketBase core/base.go, Go database/sql/driver.go.
