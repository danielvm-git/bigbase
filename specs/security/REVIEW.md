# Security Review — Epic e56: OTP Persistence & Session Audit

**Date:** 2026-07-08
**Branch/Diff:** `main` (direct commit)
**Threat Model:** `specs/security/epics/e56/THREAT_MODEL.md`

## Scope Resolution
This review covers all changes introduced in Epic e56, including:
- SQLite-backed `dbOTPStore` and `dbRateLimitStore` implementations in `components/auth/store.go`.
- Audit event logging engine (`recordAudit`) in `components/auth/auth.go`.
- OTP verification handlers (`handleVerifyOTP` and `handleVerifyPhoneOTP`) in `otp.go` and `phone.go`.
- Auth handler integration across all 18 security-sensitive event types.

## Vulnerability Assessment

### 1. Timing Attacks (CWE-208 / CWE-385)
- All OTP verification operations compare the SHA-256 hashes of the user-supplied code against the stored hash.
- Verification uses Go's cryptographic constant-time comparison library function `subtle.ConstantTimeCompare([]byte(rec.codeHash), []byte(inputHash))`.
- This ensures that execution time does not leak information about code prefixes, preventing brute-force timing extraction.

**Verdict:** PASS. Confidence: 10/10.

### 2. Race Conditions / TOCTOU Rate Limiting (CWE-307 / CWE-362)
- Incrementing rate limits could theoretically be raced if the check (SELECT count) and update (UPDATE count = count + 1) are split.
- `dbRateLimitStore.Increment` has been hardened to use an atomic SQL query:
  `UPDATE otp_rate_limits SET count = count + 1 WHERE key = ? AND count < ?`
- RowsAffected is then checked. If RowsAffected is 0, it indicates that the limit has already been reached or the row does not exist, completely resolving the Time-of-Check to Time-of-Use (TOCTOU) race.

**Verdict:** PASS. Confidence: 10/10.

### 3. SQL Injection (CWE-89)
- All queries, updates, and inserts in `components/auth/store.go` and `components/auth/auth.go` use parameterized query placeholders (`?`).
- There is no custom string formatting or string interpolation of user-controlled variables.

**Verdict:** PASS. Confidence: 10/10.

### 4. Audit Log Failure Resiliency & Request Lifecycle (CWE-755)
- `recordAudit` runs in a separate background goroutine with its own context `context.Background()` and a 5-second execution timeout.
- This ensures that if a database table lock occurs, if the table is dropped, or if the client terminates the connection early (canceling the request context), the audit log write fails gracefully in the background and does not block or fail the primary authentication flow.

**Verdict:** PASS. Confidence: 10/10.

## Conclusion
All security threats identified in `specs/security/epics/e56/THREAT_MODEL.md` have been resolved. The code is secure and conforms to security best practices.

**Verdict:** PASS
