# Story e40s01: New Relic Go Agent Initialization

**type:** feat
**context:** infra
**bcps:** 2
**status:** in_progress

## Context

Add the New Relic Go Agent (`github.com/newrelic/go-agent/v3/newrelic`) as a
dependency and initialize the `Application` object in the `main.go` `serve`
command. This is the foundation for all subsequent instrumentation stories
(e40s02 HTTP tracing, e40s03 DB tracing).

The NR agent is configured via CLI flags and environment variables:
- `--newrelic-license-key` / `NEW_RELIC_LICENSE_KEY` — required for production
- `--newrelic-app-name` / `NEW_RELIC_APP_NAME` — defaults to "BigBase"
- `--newrelic-enabled` / `NEW_RELIC_ENABLED` — defaults to `true`; set to
  `false` in dev/test to skip agent startup

**Slopcheck:** `[OK]` — maintained by New Relic, 830+ stars, stable v3 API.

## Acceptance Criteria

1. `go.mod` contains `github.com/newrelic/go-agent/v3 v3.x.x`
2. `go run . serve --help` shows `--newrelic-license-key`, `--newrelic-app-name`, `--newrelic-enabled`
3. Running with `--newrelic-license-key fake-key --log-level debug` logs "new relic agent initialized"
4. Running without the flag shows "new relic disabled" at debug level
5. The `newrelic.Application` object is stored in a variable accessible for HTTP/DB instrumentation

## Implementation Steps

1. Add `github.com/newrelic/go-agent/v3` dependency
   → verify: `grep -q 'github.com/newrelic/go-agent/v3' go.mod && go mod tidy`

2. Add CLI flags: `--newrelic-license-key`, `--newrelic-app-name`, `--newrelic-enabled`
   with environment variable fallbacks
   → verify: `go run . serve --help 2>&1 | grep -q 'newrelic-license-key'`

3. Initialize the New Relic Application in `startProxy()` after config resolution,
   storing as `nrApp` and exposing via a package-level variable or passed through
   component options. Use `newrelic.ConfigFromEnvironment()` as fallback.
   → verify: `go build -o /dev/null . 2>&1`

4. Add debug log on successful init; log "new relic disabled" when `--newrelic-enabled=false`
   → verify: `go run . serve --newrelic-license-key test --newrelic-app-name Test --log-level debug 2>&1 | grep -qi 'new relic'`

## Verification Script

1. `go build .` — must succeed
2. `go run . serve --help` — flag section shows newrelic flags
3. `go run . serve --newrelic-license-key test --log-level debug 2>&1 | grep "new relic"` — shows init log
4. `go run . serve --newrelic-enabled=false --log-level debug 2>&1 | grep "disabled"` — shows disabled log

## Out of Scope

- HTTP request tracing (e40s02)
- Database query tracing (e40s03)
- Custom metrics or segments
- Distributed tracing configuration

## Risks

- The NR agent starts background goroutines on init. With `--newrelic-enabled=false`,
  no goroutines are spawned. Testing without a real license key requires `ConfigEnabled(false)`.
- The agent requires `NEW_RELIC_LICENSE_KEY` or license key to be set, or `ConfigEnabled(false)`.
  Without either, `NewApplication` returns `err`.
