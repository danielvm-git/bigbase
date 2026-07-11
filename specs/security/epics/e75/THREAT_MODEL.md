# e75 Threat Model — Deploy Hardening

**Date**: 2026-07-11
**Epic**: e75 — Log Eviction + Flaky Tests
**Risk Level**: Low

## Surface Area

| Component | Change | Risk |
|-----------|--------|------|
| `components/deploy/logs.go` | FIFO log eviction (in-memory) | None — deterministic ordering only |
| `components/deploy/deploy.go` | Replace `deployLogs map` with struct | None — same locking, same capacity |
| `components/deploy/utils.go` | Mutex-protected `pickPort()` | None — test-only concurrency fix |
| `components/deploy/deploy_test.go` | New test + port retry + safe.directory fix | None — test-only |
| `components/deploy/drain_test.go` | safe.directory fix | None — test-only |
| `components/deploy/env_vars_test.go` | safe.directory fix | None — test-only |
| `components/deploy/samples_test.go` | safe.directory fix | None — test-only |
| `components/deploy/health_integration_test.go` | safe.directory fix | None — test-only |
| `components/deploy/db_env_test.go` | safe.directory fix | None — test-only |

## Vulnerability Analysis

### 1. Injection — NONE
No user input processing. Log eviction is an in-memory data structure operation. No SQL, command, or template injection surface.

### 2. Authentication/Authorization Bypass — NONE
No auth-related code changes. No route changes.

### 3. Secrets Exposure — NONE
No secret handling changes. `pickPort()` already uses `crypto/rand` for non-security port selection; mutex change does not alter random number generation.

### 4. Data Leakage — NONE
Log data remains in-memory with same capacity limits (`maxDeployLogDeployments=100`, `maxDeployLogLines=500`). FIFO eviction is a behavioral change only — same data, deterministic ordering.

### 5. Resource Exhaustion (DoS) — NONE
Capacity limits unchanged. FIFO eviction is O(1) vs. previous O(n) random map iteration — slight performance improvement.

### 6. Unsafe Deserialization — NONE
No serialization/deserialization changes.

### 7. Race Conditions — MITIGATED
- `pickPort()` concurrency fix adds explicit `sync.Mutex` serialization — this is a hardening improvement, eliminating a race condition.
- `logDeployments` struct with its own `sync.Mutex` maintains thread-safety that was already present via `deployLogsMu`.

### 8. Environment Variable Injection — NONE
`safe.directory` fix changes from `git config --global` to `git -c safe.directory=<path>` — this reduces the attack surface by not modifying global git config. Positive change.

## Risk Assessment

| Category | Rating | Rationale |
|----------|--------|-----------|
| Exploitability | None | All changes are in-memory or test-only |
| Impact if exploited | None | No production code paths changed |
| Data sensitivity | None | No data boundary changes |
| Overall Risk | **Low** | Purely behavioral and test-only changes |

## Verdict

**NO BLOCKING FINDINGS.** All changes are low-risk: one behavioral refinement (FIFO eviction), three test-only fixes (mutex, port retry, safe.directory). No auth, data, or network boundary changes. No secrets, injection, or deserialization surface.

### Mitigations Not Required

No additional mitigations needed beyond existing:
- Code review (Step 6 audit-code)
- Test suite passing (Step 5 verify-work hard gates)
