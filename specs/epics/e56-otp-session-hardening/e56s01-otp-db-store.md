# e56s01: Move OTP store from in-memory to database
## Story ID: e56s01 | Epic: e56 | BCPs: 2 | Status: planned

## Summary
Replace the global in-memory `otpStore` map with a DB-backed `otp_codes` table. Add `OTPStore` interface for testability. Move OTP rate limiting from in-memory `otpRates` map to `otp_rate_limits` table. Enables OTP survival across restarts and multi-instance deployment.

## Acceptance Criteria (Gherkin)
```gherkin
Scenario: OTP codes persist across restart
  Given an OTP code is sent to user@test.com
  When BigBase restarts
  Then the OTP code is still valid and can be verified

Scenario: OTP rate limit uses DB counters
  Given max 3 OTPs per hour per email
  When 4 OTPs are requested for user@test.com
  Then the 4th request is rejected
```
