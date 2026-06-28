# e56s01: Move OTP store from in-memory to database

## Story ID: e56s01 | Epic: e56 | BCPs: 2 | Status: planned

## 1. Type
**refactor** · domain · persistence

## 2. Context
BigBase currently stores OTP verification codes and rate-limit counters in global
in-memory maps (`otpStore`, `otpRates`) in `components/auth/otp.go`. These maps
are package-level `var` declarations with `sync.Mutex` guards:

```go
var (
    otpStoreMu sync.Mutex
    otpStore   = map[string]*otpRecord{}
    otpRateMu   sync.Mutex
    otpRates    = map[string]*otpRate{}
)
```

This means OTP codes are **lost on restart** (invalidates any in-flight verification)
and **cannot work in multi-instance deployments** (each instance has its own map).
Both the email OTP handlers (`handleSendOTP`, `handleVerifyOTP`) and phone OTP
handlers (`handleSendPhoneOTP`, `handleVerifyPhoneOTP`) use these maps directly.
Magic link creation in `magiclink.go` also reads from `otpRates` for rate limiting.

## 3. Problem / Opportunity

| Problem | Impact |
|---------|--------|
| OTP codes lost on server restart | All pending verifications fail — users must re-request codes |
| Rate limit state lost on restart | Attackers can bypass per-hour limits after restart |
| Cannot horizontally scale | Each instance has independent maps — OTP sent by instance A is unknown to instance B |
| Package-level mutable globals | Hard to test, impossible to mock, prevents proper DI |

## 4. Proposed solution

Replace both global maps with database-backed stores using the existing `DBer` interface.

**`OTPStore` interface:**
```go
type OTPStore interface {
    Store(ctx context.Context, email string, codeHash string, expiresAt time.Time) error
    Verify(ctx context.Context, email string, codeHash string) (bool, error)
    RecordAttempt(ctx context.Context, email string) error
    Delete(ctx context.Context, email string) error
}
```

**`RateLimitStore` interface:**
```go
type RateLimitStore interface {
    Increment(ctx context.Context, key string, window time.Time, max int) (int, error)
    Reset(ctx context.Context, key string) error
}
```

**Tables:**
- `otp_codes (email TEXT PK, code_hash TEXT, expires_at TEXT, attempts INTEGER)`
- `otp_rate_limits (key TEXT PK, count INTEGER, window_start TEXT)`

The `Auth` struct gets `otpStore OTPStore` and `rateLimitStore RateLimitStore` fields,
injected at construction. The concrete `dbOTPStore` and `dbRateLimitStore` implementations
use the existing `a.db` connection. An in-memory fallback (`mapOTPStore`) is kept for tests.

## 5. Alternatives considered

- **Redis** — rejected per ADR 0004 (no external services). Adds operational complexity.
- **SQLite in-memory with WAL** — would survive restart but adds config surface. The `otp_codes`
  table in the main DB is simpler and the `DBer` interface is already available.
- **Keep globals but add periodic flush** — doesn't solve multi-instance or testability.

## 6. Who are the users?
- **End users** who verify via email OTP or phone OTP
- **Developers** running BigBase instances who expect OTP to survive restarts
- **Platform operators** deploying multiple BigBase instances behind a load balancer

## 7. Dependencies
- e52 (Project Scoping — `kernel/scope.go`) — if multi-tenant OTP scoping is needed, but
  OTP is currently user-level (email-scoped), not org-scoped, so no hard block.
- `components/auth/` — all OTP and magic link handlers use the global maps
- `kernel/dber.go` — `DBer` interface for querying the DB

## 8. Assumptions
- OTP codes are short-lived (5 min TTL) — a background cleanup goroutine is worthwhile
  but can be deferred to a follow-up story. Expired rows are cleaned on read.
- Phone OTP uses the same `otpStore` and `otpRates` maps with `"phone:"+phone` key prefix —
  the refactor must preserve this prefix convention or migrate to a unified key scheme.
- Magic link creation reuses `otpRates` for rate limiting — this coupling must be preserved.

## 9. Risks
- **Migration order risk**: The `otp_codes` and `otp_rate_limits` tables must be created
  before any handler writes to them. Since `Start()` runs migrations before registering
  HTTP handlers, this is safe.
- **Zero-downtime risk**: A rolling restart has a window where one instance writes to DB
  and another reads from its in-memory map. Mitigation: the in-memory map is checked first
  as a fallback during transition, then removed in a follow-up after all instances deploy.
- **Performance risk**: DB round-trip for every OTP send/verify adds latency vs in-memory
  map. Mitigation: OTP volume is low (max 3 per email per hour), so this is negligible.

## 10. Non-goals
- This story does NOT introduce a background cleanup goroutine (TTL pruning). Expired
  rows are cleaned on read (by deleting stale rows before returning).
- This story does NOT change the OTP code format, TTL, or rate limit constants.
- This story does NOT add audit logging (that's e56s02).
- This story does NOT add Redis or any external cache.

## 11. Migration plan
1. Add `otp_codes` and `otp_rate_limits` table migrations in `auth.Start()`.
2. Add `OTPStore` and `RateLimitStore` interfaces to a new `components/auth/store.go`.
3. Implement `dbOTPStore` and `dbRateLimitStore` structs.
4. Add fields to `Auth` struct; inject in `New()`.
5. Update `handleSendOTP`, `handleVerifyOTP`, `handleSendPhoneOTP`, `handleVerifyPhoneOTP`,
   and magic link's rate limit check to use the injected stores.
6. Remove the global `otpStore`, `otpRates` maps and their mutexes.
7. Add in-memory store implementations for tests (`mapOTPStore`, `mapRateLimitStore`).
8. Update all existing OTP tests to use the new interfaces.

## 12. Wireframes / Diagrams
No UI changes. Data flow before:

```
HTTP handler → otpStore map (global, in-memory)
             → otpRates map (global, in-memory)
```

Data flow after:

```
HTTP handler → a.otpStore.Store/Verify() → otp_codes table
             → a.rateLimitStore.Increment() → otp_rate_limits table
```

## 13. API / Data Model

**`otp_codes` table:**
```sql
CREATE TABLE IF NOT EXISTS otp_codes (
    key TEXT PRIMARY KEY,       -- email or "phone:"+phone
    code_hash TEXT NOT NULL,
    expires_at TEXT NOT NULL,    -- ISO 8601
    attempts INTEGER NOT NULL DEFAULT 0
);
```

**`otp_rate_limits` table:**
```sql
CREATE TABLE IF NOT EXISTS otp_rate_limits (
    key TEXT PRIMARY KEY,           -- email or "phone:"+phone
    count INTEGER NOT NULL DEFAULT 0,
    window_start TEXT NOT NULL      -- ISO 8601
);
```

**New interfaces (in `components/auth/store.go`):**
```go
package auth

import "context"

type OTPStore interface {
    Store(ctx context.Context, key, codeHash string, expiresAt time.Time) error
    Get(ctx context.Context, key string) (*otpRecord, error)
    Delete(ctx context.Context, key string) error
}

type RateLimitStore interface {
    Increment(ctx context.Context, key string, windowStart time.Time, max int) (bool, error)
}
```

## 14. Affected code
| File | Change |
|------|--------|
| `components/auth/store.go` | **NEW** — OTPStore + RateLimitStore interfaces + db implementations |
| `components/auth/otp.go` | Replace globals with injected stores; update all handlers |
| `components/auth/phone.go` | Replace globals with injected stores |
| `components/auth/magiclink.go` | Replace `otpRates` read with injected store |
| `components/auth/auth.go` | Add `otpStore` and `rateLimitStore` fields; add migrations in `Start()`; update `New()` |
| `components/auth/auth_test.go` | Wire in-memory stores in `setupAuth()` |

## 15. Testing strategy

| Test | What it verifies |
|------|-----------------|
| `TestOTPStore` | Store, get, delete round-trip on `dbOTPStore` |
| `TestRateLimitStore` | Increment respects window, rejects over max |
| `TestOTPHandlerIntegration` | Full send → verify flow via HTTP, OTP survives simulated restart (new DB connection) |
| `TestPhoneOTPHandlerIntegration` | Phone send → verify flow |
| `TestRateLimitAcrossRestart` | Rate counter persists across simulated restart |
| `TestMagicLinkRateLimit` | Magic link creation respects OTP rate limits |
| `TestOTPStoreInMemory` | In-memory store implementation works for tests |

All tests use `:memory:` SQLite. The "survives restart" test creates a new `db.New()`
pointing to the same `:memory:` (which won't survive — use a temp file or `t.TempDir()`
with file-based SQLite to prove persistence).

## 16. Rollback plan
1. Revert the `auth.go` migrations (remove CREATE TABLE IF NOT EXISTS — safe no-op).
2. Revert store definitions and handler changes.
3. Restore global maps and mutexes.
4. Deploy. All existing tests cover the old code path.

## 17. Acceptance Criteria

- [ ] `otp_codes` and `otp_rate_limits` tables created in `auth.Start()` migration block
- [ ] `OTPStore` and `RateLimitStore` interfaces defined in `store.go`
- [ ] `dbOTPStore` implements `OTPStore` backed by `otp_codes` table
- [ ] `dbRateLimitStore` implements `RateLimitStore` backed by `otp_rate_limits` table
- [ ] `Auth` struct has `otpStore OTPStore` and `rateLimitStore RateLimitStore` fields
- [ ] `handleSendOTP`, `handleVerifyOTP` use injected stores instead of globals
- [ ] `handleSendPhoneOTP`, `handleVerifyPhoneOTP` use injected stores
- [ ] Magic link creation uses injected rate limit store
- [ ] All 5 original OTP-related tests pass with new store interfaces
- [ ] OTP sent before simulated restart can be verified after restart (file-based SQLite)
- [ ] No global `otpStore`, `otpRates`, `otpStoreMu`, `otpRateMu` remain in production code
- [ ] `go test ./components/auth/ -count=1` passes (all tests)
- [ ] `go vet ./components/auth/` clean

## 18. Implementation Steps (see e56s01-tasks.yaml)

## 19. Verification Script (for manual UAT)

```bash
# 1. Start BigBase with file-based DB
go run . serve --db /tmp/e56s01-test.db --port 9999 &
PID=$!
sleep 2

# 2. Request an OTP code
curl -s -X POST http://localhost:9999/api/auth/otp/send \
  -d '{"email":"test@example.com"}' | jq .

# 3. Kill and restart (simulates restart)
kill $PID
sleep 1
go run . serve --db /tmp/e56s01-test.db --port 9999 &
PID=$!
sleep 2

# 4. Verify can still be verified (will see "invalid code" rather than "not found"
#    because we can't see the code in dev mode — but the store is working)
#    For full verification, the handler log shows the code in dev mode.

# 5. Rate limit persists: request 4 OTPs — 4th should be rejected
for i in 1 2 3; do
  curl -s -X POST http://localhost:9999/api/auth/otp/send \
    -d '{"email":"ratelimit@test.com"}' | jq .
done
# After restart, request again — still rate limited
kill $PID
go run . serve --db /tmp/e56s01-test.db --port 9999 &
sleep 2
curl -s -X POST http://localhost:9999/api/auth/otp/send \
  -d '{"email":"ratelimit@test.com"}' | jq .
# Should still return "too many requests"

kill $PID 2>/dev/null
rm -f /tmp/e56s01-test.db
```

## 20. Out of scope
- OTP cleanup/expiry goroutine (deferred to future story)
- Multi-instance OTP broadcast (out of scope — DB store enables it, config is separate)
- Audit logging for OTP events (e56s02)
- OTP code format or TTL changes
- Any phone sender implementation changes
