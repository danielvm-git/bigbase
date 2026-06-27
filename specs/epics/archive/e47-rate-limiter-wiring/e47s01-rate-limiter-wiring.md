# e47s01: Wire rate limiter to auth public endpoints

## 1. Story ID
e47s01

## 2. Epic
e47 — Security: Rate Limiter Wiring

## 3. Status
planned

## 4. BCPs
2

## 5. WSJF Score
1.5 (BV=9 TC=9 RR=8 / JS=3)

## 6. Summary
Wire the existing token-bucket `RateLimiter` from `components/auth/ratelimit.go` into `main.go` so that all public auth endpoints (`/api/auth/login`, `/api/auth/register`, `/api/auth/otp/*`, etc.) are protected against brute-force and credential-stuffing attacks. The rate limiter already exists and is tested — this story is pure wiring + configuration flags.

## 7. Why
Without rate limiting, any public BigBase instance is vulnerable to unlimited login attempts, registration spam, and OTP flooding. The rate limiter was built in e19s01 but never instantiated in main.go. This closes that gap with one morning's work.

## 8. Zoom Out (modules touched)
| Module | Purpose | Change |
|--------|---------|--------|
| `main.go` | CLI flag parsing, component wiring | Add 3 flags, instantiate RL, wrap auth routes |
| `components/auth/ratelimit.go` | Token-bucket rate limiter | Add `Reconfigure()` method for runtime config |
| `components/auth/ratelimit_test.go` | Rate limiter tests | Add integration test for 429 response |

## 9. Constraints
- Default behavior must be backward-compatible: if `--rate-limit-enabled=false`, no rate limiting occurs (same as current)
- Rate-limited requests must return `429 Too Many Requests` with `Retry-After` header
- Must NOT rate-limit authenticated users on the IP bucket (use user bucket instead)
- Monitoring middleware must continue to collect metrics for rate-limited requests (429 counts toward endpoint stats)

## 10. Data Model
```go
// RateLimiterConfig — configures the token-bucket rate limiter
type RateLimiterConfig struct {
    IPLimit       int           // max requests per IPWindow per IP
    IPWindow      time.Duration // IP token-bucket window
    UserLimit     int           // max requests per UserWindow per authenticated user
    UserWindow    time.Duration // user token-bucket window
    CleanupEvery  time.Duration // stale bucket cleanup interval
}
```

## 11. API / Interface
```
GET/POST /api/auth/* → 429 Too Many Requests
  Headers: Retry-After: <seconds>
  Body: {"error":"rate limit exceeded"}
```

**No new endpoints.** The rate limiter is transparent middleware.

## 12. Testing Strategy
| Test | Type | What |
|------|------|------|
| `TestRateLimiterReconfigure` | Unit | Reconfigure() is thread-safe and updates limits atomically |
| `TestRateLimit` | Unit | Token-bucket allows N requests, blocks N+1 (already passing) |
| `TestRateLimitIntegration` | Integration | 61st login POST from same IP returns 429 with Retry-After |
| `TestRateLimitUserBucket` | Integration | Auth'd user uses user bucket (300/min), not IP bucket (60/min) |
| `TestRateLimitDisabled` | Integration | `--rate-limit-enabled=false` means 1000 requests all succeed |
| CLI flag smoke | Build | `serve --help` includes all 3 `--rate-limit-*` flags |

## 13. Component Contract
```
RateLimiter
  → Middleware(next http.Handler) http.Handler
  → NewRateLimiter(cfg RateLimiterConfig) *RateLimiter
  → Reconfigure(cfg RateLimiterConfig)
```
Consumed by `main.go`. Not consumed by any other component (auth routes only).

## 14. Security Review
- **Positive:** Rate limiting is the first line of defense against brute-force. Without it, auth is defenseless.
- **Risk:** If default limits are too aggressive, legitimate traffic is blocked. Mitigation: defaults are generous (60/min per IP), configurable, and can be disabled entirely with `--rate-limit-enabled=false`.
- **No new attack surface:** The rate limiter adds a counter check before auth processing. Failed attempts are counted regardless of rate limiting.

## 15. Performance
- Token-bucket check is O(1): read counter, decrement if available, return bool
- Bucket map uses per-key mutex (not global lock) for concurrent access
- Cleanup goroutine runs every 5 minutes, pruning stale entries
- Memory: ~1KB per active IP/user key (bucket struct + map entry overhead)

## 16. Backward Compatibility
- `--rate-limit-enabled` defaults to `true` (new behavior), but can be `false` (old behavior)
- Existing tests that hit auth endpoints may fail with 429 if they exceed limits — run with `--rate-limit-enabled=false` or set `--rate-limit-ip-max=9999` in CI
- No database migration required (rate limiter is in-memory)

## 17. Acceptance Criteria (Gherkin)
```gherkin
Feature: Rate limiting on auth endpoints

  Scenario: Login under rate limit succeeds
    Given the rate limiter allows 60 requests per minute per IP
    When I send 59 POST requests to /api/auth/login
    Then all 59 requests return 401 (invalid credentials, not rate-limited)

  Scenario: Login over rate limit returns 429
    Given the rate limiter allows 60 requests per minute per IP
    When I send 61 POST requests to /api/auth/login from the same IP
    Then request 61 returns 429 Too Many Requests
    And the response includes a Retry-After header

  Scenario: Authenticated user uses user bucket
    Given the rate limiter allows 300 requests per minute per user
    And the rate limiter allows 60 requests per minute per IP
    And I am authenticated as user 42
    When I send 61 GET requests to /api/auth/me
    Then all 61 requests succeed (user bucket, limit 300)

  Scenario: Rate limiter can be disabled
    Given the rate limiter is disabled with --rate-limit-enabled=false
    When I send 1000 POST requests to /api/auth/login
    Then no request returns 429

  Scenario: Rate limit configuration is logged at startup
    Given BigBase starts with --rate-limit-ip-max=30 --rate-limit-user-max=150
    Then the startup log includes "rate limiter configured ip=30/min user=150/min"
```

## 18. Rollout Plan
1. Merge PR with rate limiter wired, `--rate-limit-enabled=true` (default)
2. Deploy to production via GitHub Actions CI/CD
3. Monitor 429 response rate on `/api/auth/*` endpoints via `/api/monitoring/metrics/prometheus`
4. If 429 rate is too high, adjust limits: `--rate-limit-ip-max=120` and redeploy
5. If production can't tolerate any rate limiting yet, deploy with `--rate-limit-enabled=false` and re-enable after tuning

## 19. Definition of Done
- [ ] `go test ./components/auth/ -run TestRateLimiterReconfigure -v` passes (unit)
- [ ] `go test ./components/auth/ -run TestRateLimit -v` passes (unit, already green)
- [ ] `go test ./components/auth/ -run TestRateLimitIntegration -v` passes (integration)
- [ ] `go test ./components/auth/ -run TestRateLimitUserBucket -v` passes (integration)
- [ ] `go test ./components/auth/ -run TestRateLimitDisabled -v` passes (integration)
- [ ] `go build ./...` succeeds with no new warnings
- [ ] `go run . serve --help` shows `--rate-limit-enabled`, `--rate-limit-ip-max`, `--rate-limit-user-max`
- [ ] Startup log includes `rate limiter configured ip=60/min user=300/min`
- [ ] CI passes (`npm run preflight`)
- [ ] PR merged to main, semantic-release deploys

## 20. Notes
- The `Reconfigure()` method is a new addition to `RateLimiter` — currently it only supports config at construction time
- Reconfigure must be thread-safe (acquire write locks on ipMap/userMap before swapping config)
- The rate limiter should NOT apply to `authComp.ProtectedHandler()` routes (those are already behind JWT auth — the JWT validation cost is a natural rate limiter)
