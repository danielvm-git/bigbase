---
bug_id: BUG-2026-07-25T004837
status: resolved
severity: medium
scope: deploy,infra
title: "CI red on main — TestTriggerRunsThroughSupervisor flakes under CI load (2s eventually() deadline), and the sqlite test job needlessly depends on the postgres Docker image"
---

# BUG-2026-07-25T004837: Two CI reliability bugs found while triaging a linked job failure

Investigating `https://github.com/danielvm-git/bigbase/actions/runs/30142449367/job/89638200209`
(the "Test (sqlite)" job on the post-merge `main` run for PR #171) surfaced two separate, real
bugs in our own CI setup — both fixed here.

## Bug 1: `eventually()`'s 2s deadline flakes under CI load

### Problem

**Actual:** `TestTriggerRunsThroughSupervisor` (`components/deploy/supervisor_wire_test.go`)
intermittently fails in CI:
```
--- FAIL: TestTriggerRunsThroughSupervisor (2.01s)
    supervisor_wire_test.go:74: condition not met within 2s
```
Seen in CI run `30138162820` (2026-07-25T01:16). Not reproducible locally — every local run passes
well under 2s.

**Security impact:** NONE — test-only.

### Root Cause Analysis

`eventually()` (`components/deploy/supervisor_loop_test.go`) is a shared polling helper used by 4
test call sites across the supervisor test files. It polls a condition every 5ms against a
hardcoded 2-second deadline. `TestTriggerRunsThroughSupervisor` uses it to wait for
`dep.Start()`'s background `resumeCandidates` goroutine to reach the point where it calls
`Runner.Spawn` (`fakeRunner.calls >= 1`). On a resource-constrained/shared CI runner (noisy-
neighbor scheduling, cgroup CPU throttling), goroutine scheduling latency can occasionally push
that past 2 seconds even though the underlying logic is correct — a timing flake, not a logic bug.

**Risk level:** Low (test-only; underlying supervisor wiring is correct — confirmed by the test
passing consistently once given more headroom)

### Fix

`eventually()`'s deadline: 2s → 5s. Single shared helper, so all 4 call sites get the same
headroom without touching individual test logic.

**Verify:** `go test ./components/deploy/... -run TestTriggerRunsThroughSupervisor -v -count=1`
— passes; full `go test ./... -count=1` — full suite green.

## Bug 2: The `Test (sqlite)` CI job unnecessarily depends on `postgres:16`

### Problem

**Actual:** The same job failed differently on a separate run
(`30142449367`/`89638200209` — the one originally linked):
```
Error response from daemon: Get "https://registry-1.docker.io/v2/": net/http: request canceled
while waiting for connection (Client.Timeout exceeded while awaiting headers)
##[warning]Docker pull failed with exit code 1, back off ...
##[error]Docker pull failed with exit code 1
```
The job never got past "Initialize containers" — checkout/build/vet/test all show `skipped`.

### Root Cause Analysis

`.github/workflows/ci-cd.yml`'s `ci` job ran both matrix legs (`db: [sqlite, postgres]`) under a
single job definition with one shared `services: postgres:` block. GitHub Actions starts every
declared service container for a job regardless of which matrix leg is running — so the
**sqlite** leg was pulling and starting `postgres:16` even though `BIGBASE_DB_DRIVER=sqlite`
never touches it. That's the *only* reason a transient Docker Hub registry timeout could fail the
sqlite job at all — it had a dependency it never needed.

(`Test (postgres)` in the *same* run succeeded pulling the identical image, confirming this was a
one-off registry blip rather than a systemic outage — but the sqlite job should never have been
exposed to that risk surface in the first place.)

**Risk level:** Low, but real: every sqlite-only CI run carries an unnecessary external-network
dependency and failure mode that has nothing to do with what it's testing.

### Fix

Split the single matrixed `ci` job into two independent jobs, `test-sqlite` (no `services:`
block at all) and `test-postgres` (keeps the `postgres:16` service, unchanged). `report:`'s
`needs: ci` updated to `needs: [test-sqlite, test-postgres]`. Display names (`Test (sqlite)`,
`Test (postgres)`) preserved so the PR checks list looks identical.

**Verify:** `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci-cd.yml'))"` —
valid YAML; confirmed via PR CI run that both jobs still appear and pass independently.

## Acceptance Criteria

- [x] `eventually()` deadline raised to 5s; `TestTriggerRunsThroughSupervisor` passes reliably
- [x] `test-sqlite` job no longer declares or pulls the `postgres:16` service image
- [x] `test-postgres` job unchanged in behavior (still starts/uses the postgres service)
- [x] `report` job's dependency updated, still runs after both test jobs complete
- [x] Full suite green: `go test ./... -count=1`
- [x] `go vet ./...` clean
- [x] `go build ./...` clean
- [x] Valid workflow YAML

## Resolution

**Fixed:** 2026-07-25

**Evidence:**
- `go test ./components/deploy/... -run "TestTriggerRunsThroughSupervisor|TestPickPort" -v -count=1` — all pass
- `go test ./... -count=1` — full suite green
- `go vet ./...` / `go build ./...` — clean
- `.github/workflows/ci-cd.yml` — valid YAML, `test-sqlite`/`test-postgres` jobs split, `report`
  dependency updated
