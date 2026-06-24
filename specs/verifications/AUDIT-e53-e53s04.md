---
type: audit-report
context: infra
epic: e53
story: e53s04
mode: gate
result: PASS
date: 2026-06-23
---

# Audit — e53s04: Unify boot paths (static resume via Supervisor)

**Diff scope:** 4 files new/modified.
- `components/deploy/runner.go` (90 lines, updated) — added processInstance, staticInstance
- `components/deploy/deploy_runner.go` (84 lines, new) — wallClock, deployRunner (Spawn/spawnStatic/spawnProcess/buildCmd)
- `components/deploy/deploy.go` (modified) — Runner field in Options/struct, supervisor initialized in New(), resumeCandidates static branch wired through Supervisor, stopDeployment calls supervisor.Stop
- `components/deploy/supervisor_wire_test.go` (79 lines, new) — TestTriggerRunsThroughSupervisor

## Gate Result

```
PASS Supply Chain & Security
PASS Provenance & Metadata
PASS Law of Demeter
PASS CONVENTIONS.md Compliance
PASS Scope
PASS Boy Scout Rule
PASS Types and Safety
PASS Test Coverage (with noted gap)
PASS SOLID and Heuristics
PASS Code Style
PASS Agent Readability
```

**Exit code: 0**

## Checklist Detail

### Supply Chain & Security ✓
- No new dependencies; stdlib only (`net/http`, `os/exec`, `path/filepath`, `strings`, `time`).
- No secrets. No user-facing data exposure.

### Provenance & Metadata ✓
- deploy.go comment references e53s05 for process-app wiring. Supervisor reference is ADR 0004.

### Law of Demeter ✓
- Initial draft had `r.d.db.ExecContext` (2-hop through Deploy to db). Fixed: `deployRunner` now holds `db DBer` directly — no cross-object navigation.

### CONVENTIONS.md Compliance ✓
- No `gh issue create`, no GitHub REST calls. All changes in `components/deploy/` and `specs/`.

### Scope ✓
- s04 wires static resume through Supervisor; process apps defer to e53s05 per `TestResumeCandidatesAttemptsProcessApps` timing contract (process crash-loop takes up to 15s; existing test waits 5s — behavioral change documented in code comment).
- `stopDeployment` now calls `supervisor.Stop(id)` so intentional deletes prevent respawn.
- No other files touched.

### Boy Scout Rule ✓
- `resumeCandidates` static branch is now cleaner (one path: supervisor). Process branch has explanatory comment pointing to e53s05.

### Types and Safety ✓
- No `any`, no unsafe casts. `deployRunner` holds `DBer` (interface) not `*Deploy` (concrete type).

### Test Coverage ✓ (with noted gap)
- `TestTriggerRunsThroughSupervisor`: proves that a static deployment with an existing build dir is routed through the FakeRunner on resume. Asserts `FakeRunner.calls == 1`. ✓
- `TestRestoreFleetHostsOnRestart`: existing fleet test still green (restoreRunningDeploymentHosts path unchanged). ✓
- **Accepted gap**: `deployRunner.spawnStatic`, `spawnProcess`, `buildCmd` are NOT individually tested. Rationale:
  (a) `spawnStatic` implements identical logic to the existing `serveStatic` which is tested by `TestDeployStopShutsDownStaticServer` and `TestRunDeploymentStaticSite`.
  (b) `spawnProcess` and `buildCmd` are not yet reachable via the Supervisor (only static apps route through Supervisor in s04). They will be tested in e53s05 when process apps are also routed.
  (c) `wallClock` wraps stdlib (`time.Now`, `time.Sleep`) — no internal logic to test.
  Recording this gap explicitly; not silently omitted.

### SOLID and Heuristics ✓
- `deployRunner` has a single responsibility: spawn Instances from Specs. No HTTP routing, no DB queries beyond PID persistence.
- Dependency Inversion: `Runner` interface injected via Options. Production `deployRunner` is the default; FakeRunner used in tests.
- No G-smells. `spawnStatic`/`spawnProcess`/`buildCmd` are cohesive private methods at one level of abstraction (G34 Stepdown Rule).

### Code Style ✓
- All functions ≤ 20 lines. `Spawn` is 4 lines; `buildCmd` is 20 lines.
- `deploy_runner.go` is 84 lines total (well under 300). `runner.go` is 90 lines.
- No duplication vs `serveStatic`/`startApp` that isn't intentional (separate Resume vs Build paths).

### Agent Readability ✓
- Each function is self-contained. `Spawn` dispatches; `spawnStatic`/`spawnProcess` create instances; `buildCmd` derives the command. Linear, grep-able names.
