# Audit — Epic e90 (Dev-Environment Hardening)

**type:** audit-report
**context:** execution
**epic:** e90
**target:** `git diff main...HEAD` on `feat/e90-dev-environment-hardening`
**mode:** `audit-code --gate` (self-review before request-review)
**verdict:** PASS (2 findings raised and fixed during the audit)

## Scope reviewed

| Area | Files |
|---|---|
| Deploy component | `components/deploy/engine.go`, `orchestrator.go`, `deploy.go`, `export_test.go`, `drain_test.go` |
| Gate scripts | `scripts/check-conflict-markers.sh`, `scripts/check-required-env-documented.sh`, `scripts/check-version-drift.sh`, `scripts/gen-mcp-configs.sh` (doc line) |
| Gate script tests | `scripts/*.test.cjs` (4 new files, 24 tests) |
| CI | `.github/workflows/test-build-release.yml`, `.githooks/pre-commit` |
| Tooling/docs | `Makefile` (new), `AGENTS.md`, `CONVENTIONS.md`, `.gitignore`, trace markers |
| Specs | epic capsule (9 stories), 9 `-verify.yaml`, threat model, test plan |

Churn ranking helper (`scripts/bp-churn-rank.sh`) is not present in this repo; priority
was set from blast radius instead — the deploy state machine and the CI gates, which
gate every future change.

## Checklist

### Supply Chain & Security — PASS

- ✓ No dependency added or removed (`go.mod`, `go.sum`, `package.json` untouched).
- ✓ No secrets in the diff; `gitleaks detect` green, pre-commit secret scan green.
- ✓ `gosec v2.28.0` green with the single `.gosec.json` ruleset; CodeQL green (0 regressions).
- ✓ Deploy change alters only status/audit-trail writes — no new auth, input parsing or SQL interpolation (all statements parameterised).

### Provenance & Metadata — PASS

- ✓ New spec artifacts carry `type:`/`context:` metadata (`THREAT_MODEL.md`, `e90-TEST_PLAN_LATEST.md`, epic capsule, verify evidence).
- ✓ Decisions and the two red-gate fixes are recorded in `specs/state.yaml` `decisions[]` with commit references.

### Law of Demeter — PASS

- ✓ `forceStatus` is a `Deploy` method on its own state; the drain path talks to the state machine and the DB handle it already owns.

### CONVENTIONS.md Compliance — PASS

- ✓ No `gh issue create`, no direct GitHub REST calls.
- ✓ Docs written under `specs/`; test files co-located with source, matching the existing `scripts/validate-specs-yaml.test.cjs` precedent.
- ✓ Conventional Commits only; no AI attribution; subjects within the 72-char hook limit.

### Scope — PASS (with two justified expansions)

- ✓ Stories s01, s02, s03, s06, s08 reconciled work already landed; only genuine drift was touched (`AGENTS.md` "REQUIRED First Call", `.gitignore` stray `<stdin>`).
- ✓ `Makefile` tracked because `scripts/gen-mcp-configs.sh` documents `make mcp` and the file was an untracked scratch copy — a dangling contract, not a new feature.
- ✓ Discovered defect rule honoured: the red `-race` gate was fixed at its cause (product code), not retried or skipped.

### Boy Scout Rule — PASS

- ✓ `drainDeployment` duplication removed (single draining transition before the branch).
- ✓ `TestDrainStatusHistory` lost its hand-rolled 5s status loop in favour of the existing `waitForDeployStatus` helper.
- ✓ No commented-out code, no dead code (`DrainForTest` is exercised).

### Types and Safety — PASS

- ✓ Go: no `interface{}` added; `StatusTransition` reused rather than a new shape.
- ✓ TypeScript: no `any`, no `@ts-ignore`, no `eslint-disable`.

### Test Coverage — PASS

- ✓ `forceStatus` → `TestDrainWithoutRegisteredAppRecordsHistory` (regression for the exact defect).
- ✓ Each gate script has a fixture test that drives its failing path (24 tests total), and those tests now run in CI (`verify` job) so a gate cannot silently stop detecting.
- ✓ Bug fix has a regression test that fails before the fix (verified: history empty → RED, then green).
- ✓ Async assertions poll (`waitForStatusHistory`, `waitForDeployStatus`); no new bare `time.Sleep`-then-assert.

### SOLID and Heuristics — PASS

- ✓ SRP: `forceStatus` does one thing (override + record); `drainDeployment` keeps its shutdown responsibility.
- ✓ Open/Closed: the draining transition now happens for every path instead of special-casing the orphan path.
- ✓ No `F1` violations added (`waitForStatusHistory` takes 5 params including the variadic `want...`; it is test-only and the variadic keeps call sites readable).

### Refactoring Smells — PASS

- ✓ Duplicated Code (G5) removed from the conflict-marker gate (one script, two callers).
- ✓ Mysterious Name avoided: `forceStatus`, `waitForStatusHistory`, `draining`.
- ✓ No Primitive Obsession or Feature Envy introduced.

### Code Style — PASS

- ✓ Functions within size limits; `forceStatus` reads top-down (read → guard → write → append → log).
- ✓ Comments explain WHY (e.g. why status and history are eventually consistent).
- ✓ Early returns used; indentation ≤ 2 levels.

## Findings raised by this audit and fixed

| # | Finding | Fix |
|---|---|---|
| A-1 | `forceStatus` read the current status outside the mutex used for the `status_history` read-modify-write, so a concurrent transition could leave the recorded `From` stale. | Read, UPDATE and append now share one critical section (commit `3a73859ba`). |
| A-2 | `TestDrainStatusHistory` carried its own 5s polling loop for the status even though `waitForDeployStatus` already implements it. | Replaced with the shared helper. |

## Red flags (rationalisations caught)

- "The fixture tests are local-only, they don't need CI wiring." Rejected — an unwired gate test
  is decoration. Wired into `verify` and exposed as `make test-scripts`.
- "The `-race` failure is a pre-existing flake, not my story." Rejected — diagnostics showed a
  real product defect (status written without history), and the red-gate rule forbids retry-only.

## Residual risks (accepted, logged in `state.yaml`)

1. `orchestrator.go` resume path still writes `error_message` in a second statement after the
   status write; status now reaches history, `error_message` remains a separate write.
2. Harness scripts mandated by bigpowers skills (`bp-timing`, `bp-read-agents`, `assert-epic-gate`,
   `verify-tdd-red-commit`, `verify-adversarial`) are absent from this repo; the equivalent manual
   evidence was produced instead.
3. `stopDeployment`'s force path deliberately bypasses state-machine validation; it is now the only
   place that does, and it records history.
