# e50s02: Configurable token lifetimes
## Story ID: e50s02 | Epic: e50 | BCPs: 2 | Status: planned

## Summary
Make access token and refresh token lifetimes configurable via CLI flags (--jwt-access-expiry, --jwt-refresh-expiry). Defaults match current values (24h access, 30d refresh). Add `expires_at` and `expires_in` fields to auth response JSON for client awareness.

## Acceptance Criteria (Gherkin)
```gherkin
Scenario: Custom token lifetimes
  Given --jwt-access-expiry=1h
  When a user logs in
  Then the JWT expires 1 hour from now
  And the response includes expires_at and expires_in

Scenario: Default lifetimes when not configured
  Given no --jwt-access-expiry flag
  When a user logs in
  Then the JWT expires 24 hours from now (current behavior)
```
