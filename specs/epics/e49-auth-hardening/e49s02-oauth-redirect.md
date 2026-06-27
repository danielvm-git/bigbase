# e49s02: Fix OAuth redirect URI — use configured base URL
## Story ID: e49s02 | Epic: e49 | BCPs: 2 | Status: planned

## Summary
OAuth redirect URIs are built from `r.Host` which is attacker-controlled via the Host header. Replace with a configured `--public-url` flag (env: BIGBASE_PUBLIC_URL) that takes precedence, falling back to `r.Host` with a warning log for backward compatibility. Applies to handleGoogleOAuth, handleGoogleCallback, and handlePopupCallback.

## Acceptance Criteria (Gherkin)
```gherkin
Scenario: PublicURL takes precedence over Host header
  Given BIGBASE_PUBLIC_URL is set to https://bigbase.click
  When OAuth flow initiates
  Then the redirect_uri sent to Google is https://bigbase.click/api/auth/oauth/google/callback

Scenario: Fallback to r.Host when PublicURL not configured
  Given BIGBASE_PUBLIC_URL is not set
  When OAuth flow initiates
  Then the redirect_uri is built from r.Host (current behavior)
  And a warning is logged
```
