---
bug_id: BUG-2026-07-10T160003
status: fixed
severity: high
scope: auth
title: Password login has no per-account lockout (CWE-307)
---

# BUG-2026-07-10T160003: Password login brute-force protection

## Problem

`POST /api/auth/login` accepts unlimited failed password attempts per email. IP-level rate limiting exists (`RateLimiter` middleware) but does not stop distributed credential stuffing against a single account.

**Security impact: HIGH** — CWE-307 brute-force against known emails.

## Root cause (verified)

- `handleLogin` returns 401 on failure with no per-email attempt tracking.
- `otp_rate_limits` / `RateLimitStore` is OTP-only; login path never increments lockout state.

## Fix approach

Add `LoginLockoutStore` with DB table `login_lockouts`: after 5 failures within 15 minutes, lock account for 15 minutes (429 + Retry-After). Clear on successful login.

## TDD cycles

1. **RED** — `TestLoginLockoutAfterMaxFailures`: 5 wrong passwords → 6th returns 429
2. **GREEN** — Implement `loginlockout.go` + wire in `handleLogin`
3. **RED** — `TestLoginClearsLockoutOnSuccess`: after failures, correct password clears and succeeds
4. **GREEN** — `ClearFailures` on success path

## Verify

→ verify: `go test ./components/auth/ -run TestLoginLockout -count=1`
→ verify: `go test ./components/auth/ -count=1`
