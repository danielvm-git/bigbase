# e56s02: Audit log for security-sensitive operations
## Story ID: e56s02 | Epic: e56 | BCPs: 2 | Status: planned

## Summary
Create `audit_events` table and wire audit recording into 15+ auth handlers (register, login, logout, refresh, OAuth, OTP, API key management, password reset, anonymous tokens, user deletion). Each event records type, user_id, email, IP address, metadata JSON, and timestamp. INSERT-only, fire-and-forget (don't block responses).

## Acceptance Criteria (Gherkin)
```gherkin
Scenario: Login attempt is audited
  Given audit recording is enabled
  When a user logs in
  Then an audit_events row is inserted with event_type=auth.login

Scenario: Failed login is audited
  When a login attempt fails
  Then an audit_events row is inserted with event_type=auth.login_failed

Scenario: Audit failure does not block auth
  Given the audit_events table is unavailable
  When a user logs in
  Then the login still succeeds (audit error is logged, not returned)
```
