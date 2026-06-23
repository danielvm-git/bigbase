---
bug_id: BUG-2026-06-23T184500
status: fixed
severity: medium
scope: monitoring
title: BigBase app logs not forwarded to New Relic — nrslog integration missing
---

# BUG-2026-06-23T184500: BigBase app logs not forwarded to New Relic

## Problem

**Actual**: New Relic account shows zero application log lines from BigBase. The 369k logs
in the account are exclusively host OS logs (`/var/log/syslog`, `/var/log/auth.log`). No
BigBase-emitted `slog` records appear in the NR Logs UI or via NerdGraph NRQL.

**Expected**: Every `slog` record emitted by BigBase (request traces, errors, component
lifecycle events, etc.) should appear in New Relic Logs, correlated to the active APM
transaction so they're searchable alongside spans and metrics.

**Context**: The NR APM entity "BigBase" exists and reports (5 transactions in 7 days),
confirming the license key is valid and the agent connects. The gap is log forwarding
only.

## Root Cause Analysis

**Phase 1 — Reproduce**: NerdGraph NRQL `SELECT count(*) FROM Log WHERE logtype = 'app'`
returns 0. `FROM Transaction WHERE appName = 'BigBase'` returns data. APM works; logs do
not reach NR.

**Phase 2 — Isolate**: Two sub-causes, both in the main entrypoint:

1. The `logcontext-v2/nrslog` integration package is not in `go.mod`/`go.sum` at all.
   The only NR dependency is the base `go-agent/v3`. The NR Go agent does NOT
   automatically intercept `slog` output — a wrapping integration must be installed.

2. The slog logger is constructed *before* the NR application is initialised. Even if the
   integration were added today, the current ordering makes wrapping impossible at
   construction time.

**Phase 3 — Hypothesize**: Installing `nrslog`, inverting the init order (NR app first,
then logger), and wrapping the handler via `nrslog.JSONHandler(nrApp, ...)` is the
standard "Logs in Context" pattern for the NR Go agent.

**Phase 4 — Verify**: `go.sum` contains no `logcontext` entries (confirmed by grep). Grep
of all `.go` files for `nrslog`, `logWriter`, `AppLogForwarding`, `RecordLog` returns
nothing. Root cause is confirmed: missing integration package + wrong init order.

**Risk level**: Low — change is isolated to main entrypoint logger construction. No
component interface changes. Fallback (nil nrApp → plain JSON handler) preserves existing
local-dev behaviour.

## TDD Fix Plan

### Cycle 1 — extract and test `buildHandler` with NR disabled (nil app)

**RED**: Write a test `TestBuildHandler_NRDisabled` in `main_test.go` that calls
`buildHandler(nil, os.Stdout, &slog.HandlerOptions{})` and asserts the returned
`slog.Handler` is a plain `*slog.JSONHandler` (type assertion succeeds).

**GREEN**: Extract function `buildHandler(nrApp *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) slog.Handler` that returns `slog.NewJSONHandler(w, opts)` when `nrApp == nil`.

**verify**: `go test -run TestBuildHandler_NRDisabled ./...`

---

### Cycle 2 — `buildHandler` wraps via nrslog when NR app is present

**RED**: Write `TestBuildHandler_NREnabled` that creates a disabled NR app
(`newrelic.ConfigEnabled(false)` — no network, no key needed) and asserts the handler
returned by `buildHandler(app, ...)` is NOT a `*slog.JSONHandler` (i.e. the nrslog
wrapper is used).

**GREEN**:
1. `go get github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrslog`
2. In `buildHandler`, when `nrApp != nil` return
   `nrslog.JSONHandler(nrApp, w, opts)`.

**verify**: `go test -run TestBuildHandler_NREnabled ./...`

---

### Cycle 3 — wire `buildHandler` into main, fix init order

**RED**: `go build ./...` fails (or `go vet`) because `buildHandler` is not yet called
from `main`.

**GREEN**:
1. Move NR application init block above the logger construction in `main`.
2. Replace `slog.New(slog.NewJSONHandler(...))` with `slog.New(buildHandler(nrApp, os.Stdout, &slog.HandlerOptions{Level: level}))`.
3. Add `newrelic.ConfigAppLogForwardingEnabled(true)` to `newrelic.NewApplication` options.

**verify**: `go build ./... && go test ./...`

---

**REFACTOR**: If `buildHandler` grows beyond two branches, extract to `internal/logging` package. Not needed now.

## Acceptance Criteria

- [ ] `buildHandler(nil, ...)` returns `*slog.JSONHandler` (local/dev behaviour unchanged)
- [ ] `buildHandler(disabledApp, ...)` returns nrslog-wrapped handler (not plain JSON)
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes (all existing + 2 new tests green)
- [ ] After deploying with a valid license key, NerdGraph `FROM Log WHERE appName = 'BigBase'` returns > 0 results

## Resolution

**Fixed:** 2026-06-23
**Root cause confirmed:** `nrslog` integration package was missing from go.mod and the slog logger was constructed before NR app initialization, preventing log forwarding.
**Fix applied:**
  - Added `github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrslog` dependency
  - Added `newrelic.ConfigAppLogForwardingEnabled(true)` to NR app init options
  - Moved NR app initialization before logger construction in `startProxy()`
  - Extracted `buildHandler(nrApp, w, opts)` function that returns `slog.NewJSONHandler` when NR is disabled or wraps via `nrslog.WrapHandler` when NR is enabled
  - Replaced direct `slog.NewJSONHandler` call with `buildHandler(nrApp, os.Stdout, &slog.HandlerOptions{Level: level})`
**Hardening added:** Two regression tests (`TestBuildHandler_NRDisabled` and `TestBuildHandler_NREnabled`) verify both codepaths and will fail if the handler wrapping logic is removed or broken.
**Evidence:**
  - `go test -run TestBuildHandler` — 2 passed in . (main package)
  - `go test ./...` — 708 passed in 26 packages
  - `go vet ./...` — No issues found
  - `golangci-lint run main.go main_test.go` — No issues found
**Commit:** N/A — fix was already applied (check git log for details)
