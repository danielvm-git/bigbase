# BUG-2026-08-02-02: Port Allocator & Asynchronous Assertion Flakiness

## Problem

Under concurrent or repeated test runs (`go test -count=5 ./components/deploy/...`), integration tests in `components/deploy` experience potential port collisions and race conditions (EADDRINUSE). Additionally, short or inconsistent polling deadlines in asynchronous test assertions cause flakiness on busy or slow CPU scheduling.

### Symptoms
- Intermittent `EADDRINUSE` errors during fast test re-runs or parallel test execution when processes attempt to bind to ports recently closed or picked via static ranges.
- Assertion failures under CPU load due to restrictive polling timeouts in asynchronous assertions (`eventually()` or custom polling loops).

## Root Cause Analysis

1. **Port Allocation & Socket Release:**
   - Static port ranges or fixed base ports in test setups and hasty socket closures can result in ports remaining in socket release transition states (e.g. TIME_WAIT or delayed kernel cleanup).
   - Without dynamic free port allocation backed by `net.Listen("tcp", "127.0.0.1:0")` and explicit verification of socket release, concurrent test instances risk colliding on the same port numbers.

2. **Asynchronous Assertion Timeouts:**
   - Varied polling loops and short timeouts across `components/deploy` test assertions (e.g. 1s-2s limits) fail under heavy test parallelism or CPU scheduling delay when goroutines take slightly longer to complete execution.

## Fix Approach

1. **Dynamic Free Port Allocation (`components/deploy/port_allocator.go` & `port_allocator_test.go`):**
   - Implement dynamic free port allocation using `net.Listen("tcp", "127.0.0.1:0")` in `GetFreePort()`.
   - Verify listener socket release before handing off the port to caller routines.
   - Refactor `pickPort` in `components/deploy/port_allocator.go` to utilize `GetFreePort()` when `base <= 0` or fallback to OS-assigned dynamic free ports, ensuring clean isolation.

2. **Standardized Asynchronous Assertion Timeouts:**
   - Standardize `eventually()` / polling helper timeouts across `components/deploy` integration tests (5s timeout with 5ms-50ms poll intervals).
   - Ensure clean process and listener cleanup in teardowns.

## Acceptance Criteria

- [x] Bug spec `specs/bugs/BUG-2026-08-02-02-port-allocator-flake.md` created.
- [x] Dynamic free port allocation and socket release verification implemented in `components/deploy/port_allocator.go`.
- [x] Unit tests for port allocator added in `components/deploy/port_allocator_test.go`.
- [x] Polling timeouts across `components/deploy` tests standardized to handle CPU scheduling delay.
- [x] `go test -count=5 ./components/deploy/...` passes cleanly with zero flakes.
- [x] `golangci-lint run ./...` passes with zero errors.

## Resolution

**Fixed:** 2026-08-02
**Root cause confirmed:** Fixed base port ranges, unverified socket releases, and tight polling deadlines in deploy integration tests caused test flakiness under high concurrency.
**Fix applied:**
1. Created `components/deploy/port_allocator.go` with `GetFreePort()` using `net.Listen("tcp", "127.0.0.1:0")` and verified socket release loop. Refactored `pickPort` and `portIsFree` to use `port_allocator.go`.
2. Created `components/deploy/port_allocator_test.go` to test dynamic allocation, port isolation, and socket release.
3. Standardized polling timeouts in deploy integration tests to 5s.
**Evidence:** All tests in `components/deploy/...` pass 5/5 consecutive runs without flakiness (`go test -count=5 ./components/deploy/...`). `golangci-lint run ./...` returns clean.
