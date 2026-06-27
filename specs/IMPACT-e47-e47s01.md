## Target
`components/auth/ratelimit.go` — `RateLimiter`, `RateLimiterConfig`, `NewRateLimiter` + new `Reconfigure()` method
`main.go` — `startProxy()` — wiring + CLI flags

## Dependents (0)
No callers exist outside `ratelimit.go` itself. The `RateLimiter` is fully built and tested but was never instantiated in `main.go`. This story is first-time wiring.

## Affected Stories
None. No existing code depends on or imports `RateLimiter`.

## Test Coverage
- `components/auth/ratelimit_test.go` — covers ip_rate_limit, different_ips_independent, authenticated_user_higher_limit, user_bucket_takes_precedence, default_limits_are_sane
- Gap: No integration test for 429 on real auth endpoints (task 5–7 address this)

## Risk: Low (0/10)
- Fan-in: 0 callers (0 pts)
- Fan-out: 0 BigBase dependencies, stdlib only (0 pts)
- Recent churn: 1 commit ever touching ratelimit.go (0 pts)

## Recommended action
Proceed. The change is wiring an already-tested component into `main.go` with backwards-compatible flags (`--rate-limit-enabled=false` preserves current behavior).
