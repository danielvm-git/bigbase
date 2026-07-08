# Threat Model: e56 — Security: OTP Persistence & Session Audit

## 1. Epic Scope & Surface Area

The epic **e56** moves OTP store and rate limit counters from in-memory maps to database tables, and introduces a persistent security audit log.

### Attack Surface
* **OTP Send/Verify Endpoints**: `/api/auth/otp/send`, `/api/auth/otp/verify`, `/api/auth/auth/phone/send`, `/api/auth/auth/phone/verify`
* **Magic Link Endpoint**: `/api/auth/magiclink` (uses rate limiting)
* **Database Queries**: Reads/writes to `otp_codes`, `otp_rate_limits`, and `audit_events` tables.
* **Audit Event Source**: Auth handlers throughout the authentication module writing to `audit_events`.

---

## 2. Vulnerability Categories & CWE Mapping

### A. SQL Injection (CWE-89)
* **Threat**: Dynamic construction of SQL statements for storing, retrieving, or verifying OTP codes/rate limits or writing audit logs could allow SQL injection, leading to database compromise.
* **Risk Level**: **CRITICAL**
* **Mitigation Guidance**:
  * All database operations must use parameterized queries (bind arguments). No string concatenation or format string templates of user input in query strings.
  * Implement automated testing to verify SQL parameterized patterns.

### B. Rate Limit Bypass / Excessive Authentication Attempts (CWE-307)
* **Threat**: Flaws in the rate limit window check or counter increment logic (e.g., race conditions, window boundary mismatches, reset vulnerabilities) could allow attackers to brute-force OTP codes or spam requests.
* **Risk Level**: **HIGH**
* **Mitigation Guidance**:
  * Rate-limiting key naming must be consistent (e.g. `email` vs `phone:phone`).
  * Ensure the rate limit window uses robust window logic (e.g. checking if current time is within `window_start` + window duration, otherwise resetting the window).
  * Enforce maximum attempts limit and fail-secure logic.

### C. Timing Attacks on OTP Verification (CWE-208)
* **Threat**: Non-constant-time comparison of OTP hashes or codes allows an attacker to deduce the correct OTP code letter-by-letter using timing differences.
* **Risk Level**: **MEDIUM**
* **Mitigation Guidance**:
  * Use cryptographic constant-time comparison (`crypto/subtle.ConstantTimeCompare`) when comparing user-supplied codes or hashes with stored values.

### D. Log Injection / Forgeability (CWE-117)
* **Threat**: Attackers could inject special character sequences, control codes, or escape sequences into the audit log payload (e.g. user-agent, email inputs) to spoof log entries or corrupt log parse tools.
* **Risk Level**: **MEDIUM**
* **Mitigation Guidance**:
  * Sanitize or strictly parameterize all inputs logged into `audit_events`.
  * Ensure the database audit table is **INSERT-only** (fire-and-forget), with no update/delete capabilities granted to standard application components.

### E. Insufficient Session / OTP Expiration (CWE-613)
* **Threat**: OTP codes not immediately invalidated or checked for expiration can be reused or brute-forced after they should have expired.
* **Risk Level**: **HIGH**
* **Mitigation Guidance**:
  * Check the `expires_at` timestamp strictly on read.
  * Delete/invalidate the OTP code immediately upon a successful verification or after the maximum attempt threshold is reached.

---

## 3. Threat Matrix & Risk Level Summary

| Threat ID | Vulnerability | CWE | Severity | Mitigation Verification |
| :--- | :--- | :--- | :--- | :--- |
| **THR-01** | SQLi in OTP / Rate Limit / Audit queries | CWE-89 | **CRITICAL** | Parameterized query assertions |
| **THR-02** | OTP brute-force via rate limit bypass | CWE-307 | **HIGH** | Test rate limit persistence & windows |
| **THR-03** | Timing attack on verification comparison | CWE-208 | **MEDIUM** | `subtle.ConstantTimeCompare` verification |
| **THR-04** | Audit Log poisoning/injection | CWE-117 | **MEDIUM** | Query serialization & sanitization |
| **THR-05** | OTP replay or reuse after expiry | CWE-613 | **HIGH** | Test expired OTP reject & immediate delete |
