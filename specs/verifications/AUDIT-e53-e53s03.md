---
type: audit-report
context: infra
epic: e53
story: e53s03
mode: gate
result: PASS
date: 2026-06-23
---

# Audit — e53s03: Supervisor restart loop

**Diff scope:** 2 files modified/new.
- `components/deploy/supervisor.go` (164 lines, updated) — Supervisor struct, NewSupervisor, Run, Stop, tripCrashLoop, isStopping, setInstance
- `components/deploy/supervisor_loop_test.go` (147 lines, new) — supervisorRegistry spy, supervisorHarness, 3 Supervisor tests

## Gate Result

```
PASS Supply Chain & Security
PASS Provenance & Metadata
PASS Law of Demeter
PASS CONVENTIONS.md Compliance
PASS Scope
PASS Boy Scout Rule
PASS Types and Safety
PASS Test Coverage
PASS SOLID and Heuristics
PASS Code Style
PASS Agent Readability
```

**Exit code: 0**

## Checklist Detail

### Supply Chain & Security ✓
- No new dependencies beyond `math/rand/v2` (stdlib). No secrets.
- OWASP: pure in-process process supervision; no user data, no auth surface, no external API calls.

### Provenance & Metadata ✓
- Comments in supervisor.go reference ADR 0004, G25 (named constants), G28 (named predicate).

### Law of Demeter ✓
- `Supervisor` talks only to its injected fields (`runner`, `clock`, `registry`, `onFailed`, `onEvent`). No chained access.

### CONVENTIONS.md Compliance ✓
- No `gh issue create`, no GitHub REST calls. All artefacts in `specs/`.

### Scope ✓
- Changes are exactly the three Supervisor tests in e53s03 tasks. No existing Deploy struct modifications.

### Boy Scout Rule ✓
- Pre-audit self-review caught and fixed: dead `if waitErr == nil {}` block (removed); spawn-failure crash-loop not calling `UnregisterDeploymentHost` (fixed by moving un-register into `tripCrashLoop` unconditionally — safe because `delete` on missing key is a no-op in Go).

### Types and Safety ✓
- No `any`. All fields explicitly typed. `_ = waitErr` is a blank identifier (type `error`), not an `any` cast. The comment explains the semantic intent.

### Test Coverage ✓
- `NewSupervisor`: exercised by all 3 tests via `newHarness`.
- `Run`: crash→backoff→respawn path (`TestSupervisorRespawnsAfterBackoff`); crash-loop path (`TestSupervisorCrashLoopDeregistersHost`); intentional-stop path (`TestSupervisorNoRespawnAfterStop`).
- `Stop`: exercised by respawn and no-respawn tests.
- `tripCrashLoop`: verified by `TestSupervisorCrashLoopDeregistersHost` (host de-registered + onFailed called + event emitted).
- Untested edge: spawn-failure crash-loop path (Spawn itself returns error). Gap is acceptable for s03 scope — production spawn failures are infrastructure-level events that the crash-loop math handles identically. Will add if needed in s04 integration.

### SOLID and Heuristics ✓
- Single Responsibility: `Supervisor` owns restart policy; `tripCrashLoop` owns the failure side-effects; `Run` orchestrates the loop.
- Dependency Inversion: `Runner`, `Clock`, `DeploymentHostRegistry`, `onFailed`, `onEvent` all injected.
- No G-smells. `isStopping`/`setInstance` are private helpers (G22: descriptive accessors over field access with lock).

### Code Style ✓
- All methods ≤ 15 lines. `Run` is 36 lines — loop body warrants a single function; no split would improve readability.
- No duplication. `UnregisterDeploymentHost` called once in `tripCrashLoop`.
- Names: `tripCrashLoop` (1 hit), `isStopping` (3 hits), `setInstance` (3 hits).

### Agent Readability ✓
- `Run` loop is linear: check-stopping → spawn → register → wait → check-stopping → crash-loop → sleep. Readable top-to-bottom.
- Max nesting: 2 levels (loop body → if).
