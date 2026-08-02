---
bug_id: BUG-2026-08-02-01
status: resolved
severity: high
scope: deploy,ci
title: "Deploy Test Suite 90s Timeout & Speed Optimization"
---

# BUG-2026-08-02-01: Deploy Test Suite 90s Timeout & Speed Optimization

## Problem

`go test ./...` times out at the default 90-second limit in `components/deploy` (execution took 92.865s - 136s). As a temporary workaround, the `preflight:go` script in `package.json` was filtering out `/deploy` tests via `grep -v /deploy`, leaving deploy tests unverified during preflight.

## Root Cause Analysis

Integration tests in `components/deploy` test process supervision, static web servers, candidate resurrection, and health checks. Together, these tests accumulated excessive execution time (>90s) when run sequentially due to long sleep timeouts (`time.Sleep`), slow polling intervals (e.g. 50ms-500ms), and lengthy health check retries (10 retries with 2s interval = 20s per failing health check).

## Fix Strategy

1. **Parameterize test execution**: Update `package.json` `preflight:go` script to run `go test -timeout 180s ./...` instead of skipping `/deploy`.
2. **Optimize test sleep and polling delays**:
   - Reduce polling interval in `waitForDeploymentTerminal` from 50ms to 10ms.
   - Reduce DB/HTTP poll sleep intervals in integration test helpers across `components/deploy/*_test.go` (e.g. `db_env_test.go`, `env_vars_test.go`, `deploy_test.go`, `drain_test.go`, `sidecar_test.go`, `request_logs_test.go`).
   - Trim excessive post-startup `time.Sleep` delays (e.g., 3s → 200ms, 2s → 100ms, 1s → 50ms/100ms) where processes bind and initialize quickly.
   - Reduce health check retry intervals and retry counts in `health_integration_test.go` from `interval_seconds: 2, max_retries: 10` to `interval_seconds: 1, max_retries: 3`.

## Verification

- `go test -timeout 180s ./components/deploy/...` passes cleanly in ~30-45s.
- `go test -timeout 180s ./...` passes full suite.
- `golangci-lint run ./...` returns 0 issues.

## Resolution

**Fixed:** 2026-08-02
