# TestRedeployReplacesPrevious flaky — host not registered after first deploy

**Source:** CI gate failure (GitHub Actions run 31549338708, Build & Package job 93969441133)
**Severity:** MEDIUM
**Scope:** deploy

## Description

CI `go test` in the Build & Package job failed intermittently:
`deploy_test.go:1866: host not registered after first deploy` in
`TestRedeployReplacesPrevious`. Reproduced locally: **4/5 passes, 1 flake**
(1-in-5). All other jobs in the run (Lint, Test sqlite/postgres, verify,
Release) passed.

## Root cause (4-phase RCA)

1. **Reproduce**: `go test ./components/deploy/ -run TestRedeployReplacesPrevious -count=5` → run 3 failed with the same assertion.
2. **Isolate**: The assertion is the first-deploy host check — `hostReg.getPort(host)` immediately after `waitForDeploymentTerminal` returns "running". The second-deploy check (line ~1906) already polls with retry and never flakes — the defect is specific to the one-shot check.
3. **Hypothesize**: `runDeployment` executes in a goroutine (`engine.go:79` — `go d.runDeployment(...)`). Inside it, `updateStatus("running")` writes the status to the DB, then `finalizeDeploymentURL` → `RegisterDeploymentHost` writes to the **in-memory** host map (test mock). A test reading the status from the DB can observe "running" microseconds before the host registration lands.
4. **Verify**: Confirmed the call order in `engine.go` (267-268: `updateStatus("running")` then `finalizeDeploymentURL`), and that the window is real under scheduler load (1-in-5).

## Fix applied

Shared `waitForHostRegistration` test helper polls until the host maps to the expected port (or any port), mirroring the existing status-polling pattern. Applied to the one-shot check in `TestRedeployReplacesPrevious` and the **same defect class** in `drain_test.go` (lines 160, 205, 272).

## Hardening added

- Polling helper for host-registration assertions (replaces one-shot checks across 2 test files)
- Generalize sweep: all 9 `getPort` usages audited; the 4 one-shot sites fixed, remaining uses are the helper itself or synchronous domain-routing assertions (no goroutine window)

## Evidence

- `go test ./components/deploy/ -run TestRedeployReplacesPrevious -count=15` → **all pass** (was 1-in-5 flake)
- `go test ./components/deploy/ -run 'TestDrain|TestRedeploy' -count=10` → all pass
- `go test ./components/deploy/ -count=2` → pass
- `go test ./...` (full suite, the CI gate) → pass, no failures

## Status
fixed

## Source
fix-bug-e51s04-2026-08-12 (CI gate failure)

## Discovered
2026-08-12
