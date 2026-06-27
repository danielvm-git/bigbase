# e50s03: Refresh token revocation — per-user invalidation
## Story ID: e50s03 | Epic: e50 | BCPs: 1 | Status: planned

## Summary
Add POST /api/auth/logout-all endpoint that invalidates all refresh tokens for the authenticated user. Does NOT invalidate the current access token (it remains valid for its remaining lifetime, max 24h). Follows Supabase/Auth0 pattern: short-lived access tokens + refresh token rotation + replay detection.

## Acceptance Criteria (Gherkin)
```gherkin
Scenario: Logout all invalidates refresh tokens
  Given user has 3 active refresh tokens
  When POST /api/auth/logout-all is called
  Then all 3 refresh tokens are marked as used
  And subsequent refresh attempts return 401
```
