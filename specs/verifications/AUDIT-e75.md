# AUDIT-e75 — Deploy Hardening

**Date**: 2026-07-11
**Epic**: e75
**Auditor**: build-epic Step 6 gate
**Verdict**: PASS

## Checklist

### 1. CONVENTIONS.md Compliance — PASS
- [x] Go standard layout maintained
- [x] Component interface (Init/Start/Stop) untouched
- [x] No hardcoded secrets, API keys, or tokens
- [x] No direct imports between components (event bus pattern intact)
- [x] `sync.Mutex` used for thread safety (no raw state mutation)

### 2. Boy Scout Rule — PASS
- [x] `initDeployLogs` improved from O(n) random map iteration to O(1) deterministic FIFO
- [x] `pickPort` improved from race-prone `crypto/rand` to mutex-protected counter
- [x] Test infrastructure improved: `--global --add safe.directory` → `-C <path> config --local`
- [x] `maxDeployLogLines`/`maxDeployLogDeployments` changed from `const` to `var` for testability
- [x] `logDeployments` struct encapsulates log buffer with its own `sync.Mutex`

### 3. Test Coverage — PASS
- [x] New `TestDeployLogFIFOEviction` tests deterministic eviction order
- [x] 3 flaky tests pass 10 consecutive runs (30/30)
- [x] Full deploy suite: 256/256 passed
- [x] Lint (`golangci-lint`): no issues
- [x] Go vet: no issues
- [x] Build: success

### 4. Type Safety — PASS
- [x] `logDeployments` uses concrete types (`map[string][]string`, `[]string`)
- [x] `pickPortCounter` uses `int64` (overflow infeasible with 2^63 calls)
- [x] No `any` usage in new code

### 5. SOLID Principles — PASS
- [x] **S**: `logDeployments` struct has single responsibility (log buffer management)
- [x] **O**: Capacity changed to `var` — open for test configuration, closed for production modification
- [x] **L**: No subtype changes — `Deploy` still satisfies `kernel.Component`
- [x] **I**: No interface changes
- [x] **D**: No new dependencies introduced

### 6. Security — PASS
- [x] Threat model (Step 0): Low risk, no blocking findings
- [x] `safe.directory` change reduces attack surface (no global git config mutation)
- [x] `pickPort` mutex eliminates concurrency race condition
- [x] No auth, data, or network boundary changes

## F.I.R.S.T Assessment (new test: TestDeployLogFIFOEviction)

| Criterion | Result | Notes |
|-----------|--------|-------|
| **F**ast | PASS | <2s execution |
| **I**ndependent | PASS | No shared state with other tests; resets constants via defer |
| **R**epeatable | PASS | Deterministic FIFO ordering is repeatable |
| **S**elf-validating | PASS | Uses `t.Fatal`/`t.Fatalf` for assertions |
| **T**imely | PASS | Written with the implementation (TDD pattern) |

## Files Changed

| File | Change Type | Risk |
|------|------------|------|
| `components/deploy/logs.go` | Refactor + new struct | Low — same locking, same capacity |
| `components/deploy/deploy.go` | Field replacement | Low — initialization in New() |
| `components/deploy/utils.go` | pickPort mutex | Low — test-only concurrency fix |
| `components/deploy/logs_test.go` | New test | None |
| `components/deploy/deploy_test.go` | Test fixes | None — test-only |
| `components/deploy/drain_test.go` | Test fixes | None — test-only |
| `components/deploy/env_vars_test.go` | safe.directory fix | None — test-only |
| `components/deploy/samples_test.go` | safe.directory fix | None — test-only |
| `components/deploy/health_integration_test.go` | safe.directory fix | None — test-only |
| `components/deploy/db_env_test.go` | safe.directory fix | None — test-only |

## Known Pre-existing Issues (not in e75 scope)

- `TestRuntimeInjectsNativeDBEnv_sqlite` — DB_PATH mismatch (Node.js runtime)
- `TestRuntimeLogs` — intermittent deployment failure (Node.js runtime)
- `TestDrainFailedDeployment` — health check race (Go runtime)
- Leftover Node.js processes from runtime tests not properly cleaned up

## Verdict

**PASS** — All checklist sections pass. No blocking findings. e75 changes are low-risk, well-tested, and improve code quality. Proceed to Step 7 (commit message).
