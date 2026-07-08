# e56s02: Audit log for security-sensitive operations

## Story ID: e56s02 | Epic: e56 | BCPs: 2 | Status: planned

## 1. Type
**feat** · domain · observability

## 2. Context
BigBase's auth component handles 15+ sensitive operations (login, logout, register,
OAuth callback, OTP send/verify, password reset, API key management, anonymous token
issuance, user deletion). None of these operations record **who did what and when**.

Without an audit log:
- A compromised API key can create more API keys silently
- Failed login attempts leave no forensic trace
- There is no way to detect or investigate insider abuse
- Compliance requirements (SOC 2, GDPR Art. 33 breach detection) cannot be met

The existing monitoring component (`components/monitoring/`) has host metrics, alerts,
and process tracking, but no per-request security event log.

## 3. Problem / Opportunity

| Problem | Impact |
|---------|--------|
| No audit trail for auth events | Cannot investigate security incidents or abuse |
| No way to detect brute-force patterns | Failed logins leave no trace beyond request logs |
| No compliance evidence | SOC 2, GDPR, and SOC require audit logging |
| OTP events untracked | No record of OTP send/verify/success/failure |

## 4. Proposed solution

Create an `audit_events` table and a `recordAudit()` helper on the `Auth` struct.
Wire audit recording into 15+ auth handlers. The recording is **INSERT-only,
fire-and-forget** — it must never block or fail the primary operation.

**`audit_events` table:**
```sql
CREATE TABLE IF NOT EXISTS audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    user_id INTEGER,
    email TEXT,
    ip_address TEXT,
    metadata TEXT,        -- JSON blob, nullable
    created_at TEXT NOT NULL
);
```

**`recordAudit` helper signature:**
```go
func (a *Auth) recordAudit(ctx context.Context, eventType string, userID int64, email, ip string, metadata map[string]any)
```

**Events to record (18 total):**

| Event Type | Handler | Data |
|-----------|---------|------|
| `auth.register` | handleRegister | user_id, email |
| `auth.login` | handleLogin (success) | user_id, email |
| `auth.login_failed` | handleLogin (bad password) | email |
| `auth.logout` | handleLogout | user_id, email |
| `auth.token_refresh` | handleRefresh | user_id, email |
| `auth.oauth_callback` | handleGoogleCallback | user_id, email |
| `auth.otp_sent` | handleSendOTP | email |
| `auth.otp_verified` | handleVerifyOTP | email |
| `auth.otp_failed` | handleVerifyOTP (bad code) | email |
| `auth.phone_otp_sent` | handleSendPhoneOTP | phone |
| `auth.phone_otp_verified` | handleVerifyPhoneOTP | phone |
| `auth.phone_otp_failed` | handleVerifyPhoneOTP (bad code) | phone |
| `auth.api_key_created` | handleCreateAPIKey | user_id, email |
| `auth.api_key_deleted` | handleDeleteAPIKey | user_id, email |
| `auth.password_reset_requested` | handleForgotPassword | email |
| `auth.password_reset_completed` | handleResetPassword | user_id, email |
| `auth.anonymous_token` | handleAnonymousToken | user_id (auto-created) |
| `auth.user_deleted` | handleUserByID (DELETE) | user_id |

## 5. Alternatives considered

- **Log-based audit** (parsing server logs) — fragile, logs may rotate, no structured querying.
- **Monitoring component** (`components/monitoring/`) — the monitoring component already has
  metrics and event streams. Adding audit events there would require cross-component
  communication via the event bus. Keeping audit in `auth` keeps it self-contained and
  avoids coupling to an independent component that may be disabled in lite profiles.
- **External SIEM** — overkill for a single-binary BaaS. The table enables future export.

## 6. Who are the users?
- **Platform operators** who need to investigate security events
- **Compliance officers** who need audit trails for SOC 2 / GDPR
- **Developers** debugging auth issues

## 7. Dependencies
- `components/auth/auth.go` — all 15+ handler functions to instrument
- `kernel/dber.go` — `DBer.ExecContext` for INSERT
- No external dependencies — pure SQLite table

## 8. Assumptions
- Audit table growth: at ~100 bytes per row × 100 events/hour × 24h = ~240KB/day.
  Not a concern for SQLite. A future story can add TTL-based cleanup.
- Fire-and-forget means: if the INSERT fails (DB error), we log the error via
  `a.logger.Error()` and continue. The auth operation is never rolled back.
- IP address is extracted from `r.RemoteAddr` (stripping port). For proxy deployments,
  this will be the proxy IP unless BigBase is configured to trust `X-Forwarded-For`.
  No `X-Forwarded-For` parsing is added in this story (documented limitation).

## 9. Risks
- **Performance**: 18 `INSERT` statements per auth operation. Mitigation: each handler
  does at most 1 INSERT, and the table has no indexes beyond the PK auto-increment.
- **Table bloat**: No cleanup policy in this story. Mitigation: at ~240KB/day, this is
  acceptable for months. A cleanup story can be added when table size becomes a concern.
- **Forgotten handlers**: A new auth handler added later might be missed. Mitigation:
  any new handler should include `recordAudit`. This will be documented in the convention
  comment at the top of `auth.go`.

## 10. Non-goals
- No audit event query API or admin UI (future story)
- No TTL/cleanup for old audit events (future story)
- No X-Forwarded-For trust configuration (documented limitation)
- No integration with the monitoring component's event stream
- No indexed columns beyond the PK

## 11. Migration plan
1. Add `audit_events` table migration in `auth.Start()`.
2. Add `recordAudit()` helper method on `Auth`.
3. Wire into 6 core handlers: handleRegister, handleLogin (success + fail), handleLogout, handleRefresh.
4. Wire into 4 OTP/phone handlers: handleSendOTP, handleVerifyOTP, handleSendPhoneOTP, handleVerifyPhoneOTP.
5. Wire into 2 API key handlers: handleCreateAPIKey, handleDeleteAPIKey.
6. Wire into 2 password reset handlers: handleForgotPassword, handleResetPassword.
7. Wire into handleGoogleCallback, handleAnonymousToken, handleUserByID (DELETE).
8. Add tests for each event type and the fire-and-forget behavior.
9. Add integration test proving auth succeeds even when audit_events table is unavailable.

## 12. Wireframes / Diagrams
No UI changes. Data flow:

```
auth handler ──► primary logic (login, register, etc.)
              │
              └──► recordAudit() ──► INSERT INTO audit_events
                   (fire-and-forget, error logged only)
```

## 13. API / Data Model

**`audit_events` table:**
```sql
CREATE TABLE IF NOT EXISTS audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    user_id INTEGER,
    email TEXT,
    ip_address TEXT,
    metadata TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

**Helper method:**
```go
func (a *Auth) recordAudit(ctx context.Context, eventType string, userID int64, email, ip string, metadata map[string]any) {
    metaJSON := "{}"
    if metadata != nil {
        b, err := json.Marshal(metadata)
        if err == nil {
            metaJSON = string(b)
        }
    }
    _, err := a.db.ExecContext(ctx,
        `INSERT INTO audit_events (event_type, user_id, email, ip_address, metadata, created_at) VALUES (?, ?, ?, ?, ?, datetime('now'))`,
        eventType, userID, email, ip, metaJSON,
    )
    if err != nil {
        a.logger.Error("audit write failed", "event_type", eventType, "error", err)
    }
}
```

## 14. Affected code
| File | Change |
|------|--------|
| `components/auth/auth.go` | Add migration in `Start()`; add `recordAudit()` method; wire into all 15+ handlers |
| `components/auth/otp.go` | Wire `recordAudit` into `handleSendOTP`, `handleVerifyOTP` |
| `components/auth/phone.go` | Wire `recordAudit` into `handleSendPhoneOTP`, `handleVerifyPhoneOTP` |
| `components/auth/auth_test.go` | Add tests for each event type |

## 15. Testing strategy

| Test | What it verifies |
|------|-----------------|
| `TestAuditEventsTable` | Migration creates the table; INSERT and SELECT round-trip |
| `TestRecordAudit` | `recordAudit()` writes a row with correct fields |
| `TestAuditLoginEvents` | Login success → `auth.login`; login failure → `auth.login_failed` |
| `TestAuditLogoutEvent` | Logout → `auth.logout` |
| `TestAuditOTPEvents` | OTP send → `auth.otp_sent`; OTP verify → `auth.otp_verified`; bad code → `auth.otp_failed` |
| `TestAuditAPIKeyEvents` | API key create → `auth.api_key_created`; delete → `auth.api_key_deleted` |
| `TestAuditPasswordResetEvents` | Forgot password → `auth.password_reset_requested`; reset → `auth.password_reset_completed` |
| `TestAuditRegistrationEvent` | Register → `auth.register` |
| `TestAuditIntegration` | All 18 event types recorded; each with correct event_type, user_id/email, ip_address |
| `TestAuditFireAndForget` | When audit table is unavailable (drop table mid-test), auth operation still succeeds |

## 16. Rollback plan
1. Remove the `audit_events` CREATE TABLE migration — safe no-op on re-deploy.
2. Remove all `recordAudit()` calls from handlers.
3. Remove the `recordAudit()` method.
4. Deploy. All existing tests pass (audit tests are additive).

## 17. Acceptance Criteria

- [ ] `audit_events` table created in `auth.Start()` migration block
- [ ] `recordAudit()` helper method exists on `Auth` with correct INSERT
- [ ] All 18 event types are wired:
  - [ ] `auth.register` (handleRegister)
  - [ ] `auth.login` / `auth.login_failed` (handleLogin)
  - [ ] `auth.logout` (handleLogout)
  - [ ] `auth.token_refresh` (handleRefresh)
  - [ ] `auth.oauth_callback` (handleGoogleCallback)
  - [ ] `auth.otp_sent` / `auth.otp_verified` / `auth.otp_failed` (handleSendOTP, handleVerifyOTP)
  - [ ] `auth.phone_otp_sent` / `auth.phone_otp_verified` / `auth.phone_otp_failed` (handleSendPhoneOTP, handleVerifyPhoneOTP)
  - [ ] `auth.api_key_created` / `auth.api_key_deleted` (handleCreateAPIKey, handleDeleteAPIKey)
  - [ ] `auth.password_reset_requested` / `auth.password_reset_completed` (handleForgotPassword, handleResetPassword)
  - [ ] `auth.anonymous_token` (handleAnonymousToken)
  - [ ] `auth.user_deleted` (handleUserByID DELETE)
- [ ] Fire-and-forget: when audit INSERT fails, the error is logged and the auth operation succeeds
- [ ] `go test ./components/auth/ -count=1 -v` passes all tests including new audit tests
- [ ] `go vet ./components/auth/` clean
- [ ] IP address extracted from `r.RemoteAddr` (port stripped)

## 18. Implementation Steps (see e56s02-tasks.yaml)

## 19. Verification Script (for manual UAT)

```bash
# 1. Start BigBase
go run . serve --db /tmp/e56s02-test.db --port 9999 &
PID=$!
sleep 2

# 2. Register a user (generates auth.register event)
RESP=$(curl -s -X POST http://localhost:9999/api/auth/register \
  -d '{"email":"audit@test.com","password":"secret123"}')
echo "$RESP" | jq .

# 3. Directly query audit_events table to verify
sqlite3 /tmp/e56s02-test.db "SELECT event_type, email, ip_address FROM audit_events;"

# 4. Try login with wrong password (generates auth.login_failed)
curl -s -X POST http://localhost:9999/api/auth/login \
  -d '{"email":"audit@test.com","password":"wrong"}' | jq .
sqlite3 /tmp/e56s02-test.db "SELECT event_type, email FROM audit_events WHERE event_type='auth.login_failed';"

# 5. Login succeeds (generates auth.login)
curl -s -X POST http://localhost:9999/api/auth/login \
  -d '{"email":"audit@test.com","password":"secret123"}' | jq .
sqlite3 /tmp/e56s02-test.db "SELECT event_type, email FROM audit_events WHERE event_type='auth.login';"

kill $PID 2>/dev/null
rm -f /tmp/e56s02-test.db
```

## 20. Out of scope
- Audit event query API or admin UI page
- TTL/cleanup of old audit events
- X-Forwarded-For trust configuration
- Integration with monitoring component's event stream or SSE
- Indexed columns for fast querying of specific event types
