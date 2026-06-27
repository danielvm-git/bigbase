# e50s01: Persist JWT secret — env var with fallback
## Story ID: e50s01 | Epic: e50 | BCPs: 2 | Status: planned

## Summary
Add BIGBASE_JWT_SECRET env var support. When set, use it as the JWT signing key (persists across restarts). When not set, auto-generate a random secret on startup (current behavior, with a warning log). Validate provided secrets: min 32 bytes, not a known default.

## Acceptance Criteria (Gherkin)
```gherkin
Scenario: JWT secret loaded from env var
  Given BIGBASE_JWT_SECRET is set to a 64-char hex string
  When BigBase starts
  Then tokens signed with that secret validate successfully
  And "JWT secret loaded from configuration" is logged

Scenario: Auto-generated secret when not configured
  Given BIGBASE_JWT_SECRET is not set
  When BigBase starts
  Then a random 64-char hex secret is generated
  And a warning is logged
```
