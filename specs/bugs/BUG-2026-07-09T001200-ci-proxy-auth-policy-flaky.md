---
bug_id: BUG-2026-07-09T001200
status: fixed
severity: high
scope: proxy,ci
title: CI fails on flaky TestProxyAuthPolicy — connection refused
---

# BUG-2026-07-09T001200: CI fails on flaky TestProxyAuthPolicy

## Problem

- **Actual**: GitHub Actions CI (`ci.yml`) fails on `main` in the `Test (sqlite)` job. `TestProxyAuthPolicy` errors with `dial tcp 127.0.0.1:<port>: connect: connection refused`.
- **Expected**: All proxy tests pass reliably in CI and locally.
- **Reproduce**: `go test -run TestProxyAuthPolicy ./components/proxy/... -count=5` — fails ~60% of runs locally.

**Security impact**: NONE — test-only race condition, no production exploit path.

## Root Cause Analysis

`Proxy.Start()` launches the HTTP server in a background goroutine and returns immediately. Every other test in `hosts_test.go` calls `waitForServer(t, port, "/health")` after `Start()` to poll until the listener is ready. `TestProxyAuthPolicy` (added in e2360314) omits this wait, so the first HTTP request races against server startup and intermittently gets "connection refused".

Additionally, the test omits `Kernel` in `proxy.Options` while `waitForServer` probes `/health`, which calls `p.kernel.ListComponents()` without a nil guard — causing panics when the server is ready before the test request.

- **Risk level**: Low — one-line test fix, no production code change.

## TDD Fix Plan

1. **RED**: `go test -run TestProxyAuthPolicy ./components/proxy/... -count=10` — reproduce intermittent failure.
   **GREEN**: Add `Kernel: k` and `waitForServer(t, port, "/health")` after `p.Start()` in `TestProxyAuthPolicy`, matching sibling tests.
   **verify**: `go test -run TestProxyAuthPolicy ./components/proxy/... -count=10`

**REFACTOR**: None needed.

## Acceptance Criteria

- [x] `TestProxyAuthPolicy` passes 10 consecutive runs locally
- [x] `go test ./components/proxy/...` passes
- [x] `go test ./...` passes (full suite)
- [ ] CI `Test (sqlite)` job passes

## Resolution

Fixed in `components/proxy/hosts_test.go` — `TestProxyAuthPolicy` now passes `Kernel: k` and calls `waitForServer(t, port, "/health")` after `p.Start()`, matching all sibling tests in the file.

**Verified**:
- `go test -run TestProxyAuthPolicy ./components/proxy/... -count=20` — 20/20 pass
- `go test ./components/proxy/...` — 75/75 pass
- CI failure reproduced locally before fix (~60% flake rate); 0% after fix
