# Story e40s02: Instrument HTTP Proxy Request Tracing

**type:** feat
**context:** infra
**bcps:** 2
**status:** done

## Context

Add New Relic transaction tracing to the proxy component's HTTP handler chain.
Each incoming HTTP request is wrapped in a NR transaction, providing:
- Transaction names (e.g. "GET /api/auth/login")
- Response time and status code breakdowns
- Error traces with stack traces
- Distributed tracing headers (for downstream service calls)

## Implementation

- Added `NRApp *newrelic.Application` to `proxy.Options` and `Proxy` struct
- Created `newRelicMiddleware` that starts a transaction, sets WebRequest/WebResponse, and stores the transaction in the request context
- Injected middleware in the Handler() chain (outermost, after CORS/security but wrapping the rest)
- Wired `nrApp` from main.go into `proxy.Options`

## Acceptance Criteria

1. Build succeeds with proxy importing newrelic
2. All tests pass
3. Running with NR license key logs transactions for HTTP requests
