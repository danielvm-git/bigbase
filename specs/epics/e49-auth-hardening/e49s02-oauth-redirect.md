# e49s02: Fix OAuth redirect URI — use configured base URL

## 1. Story ID
e49s02

## 2. Epic
e49 — Security: Auth Hardening (Anonymous, OAuth, Path Traversal)

## 3. Status
planned

## 4. BCPs
2

## 5. Type
fix

## 6. Context
domain

## 7. Summary
OAuth redirect URIs in `handleGoogleOAuth`, `handleGoogleCallback`, and `handlePopupCallback` are built from `r.Host`, which is attacker-controlled via the HTTP Host header. An attacker sending a forged `Host: evil.com` header tricks BigBase into requesting Google to redirect back to `evil.com`, enabling token theft. Fix by adding a `PublicURL` config field that takes precedence over `r.Host`, with a warning-logged fallback to `r.Host` for backward compatibility.

## 8. Problem Statement
Three code locations construct the OAuth `redirect_uri` using `r.Host`:

**`handleGoogleOAuth`** (auth.go:757):
```go
scheme := "http"
if r.TLS != nil { scheme = "https" }
redirectURI := fmt.Sprintf("%s://%s/api/auth/oauth/google/callback", scheme, r.Host)
```

**`handleGoogleCallback`** (auth.go:805):
```go
redirectURI: fmt.Sprintf("%s://%s/api/auth/oauth/google/callback", scheme, r.Host),
```

**`handlePopupCallback`** (anonymous.go:85):
```go
redirectURI: fmt.Sprintf("%s://%s/api/auth/oauth/google/callback", scheme, r.Host),
```

The HTTP `Host` header is set by the client and can be trivially spoofed:
```bash
curl -H "Host: evil.com" https://bigbase.click/api/auth/oauth/google
# → redirects to Google with redirect_uri=https://evil.com/api/auth/oauth/google/callback
```

Google's OAuth server validates the `redirect_uri` against registered URIs — but since `bigbase.click` and `*.bigbase.click` are likely both registered, an attacker controlling a subdomain or spoofing the Host header on a shared host could capture the authorization code.

**Note**: `r.Host` in Go's `net/http` includes the port (e.g., `localhost:9999`), so `fmt.Sprintf("%s://%s/...")` naturally produces `http://localhost:9999/...` — correct for dev. But in production behind a reverse proxy, `r.Host` reflects the incoming Host header.

## 9. Proposed Solution
1. Add `PublicURL string` field to `Options` and `Auth` struct
2. Add helper method `func (a *Auth) publicURLOrDefault(r *http.Request) string`:
   - If `a.publicURL != ""`: return `a.publicURL` (takes precedence)
   - Else: construct URL from `r.Host` + scheme (current behavior) with a warning log
3. Replace all three `fmt.Sprintf("%s://%s/api/auth/oauth/google/callback", scheme, r.Host)` calls with `a.publicURLOrDefault(r) + "/api/auth/oauth/google/callback"`
4. Wire `--public-url` CLI flag to `BIGBASE_PUBLIC_URL` env var in `main.go`

## 10. Affected Modules

| Module | Purpose | Callers | Contracts |
|--------|---------|---------|-----------|
| `components/auth/` | Authentication, JWT minting/verification, OAuth flow | Proxy (event hooks), all OAuth clients via `/api/auth/oauth/*` | `handleGoogleOAuth` returns 302 to Google; `handleGoogleCallback` exchanges code for token; `handlePopupCallback` returns HTML with postMessage |
| `main.go` | CLI entry point, flag parsing | User via CLI | Parses `--public-url` flag, passes to auth.Options |

## 11. Dependencies
- **Zero new external packages** — uses existing `net/http`, `fmt`, `net/url`

## 12. Implementation Steps

### Story e49s02: Fix OAuth redirect URI — Implementation Steps

**type:** fix
**context:** domain
**Context**: OAuth redirect URIs are built from the client-controlled `Host` header, enabling Host header poisoning attacks. Replace with a configurable `PublicURL` that takes precedence, falling back to `r.Host` with a deprecation warning for backward compatibility.

## Steps

1. Add `PublicURL` field to `Options` and `Auth` struct; add `publicURLOrDefault(r)` helper method on Auth → verify: `go build ./...`

2. Replace `r.Host`-based redirect URI construction in `handleGoogleOAuth` with `a.publicURLOrDefault(r)` → verify: `go test ./components/auth/ -run TestGoogleCallbackRedirectConfigurable -v`

3. Replace `r.Host`-based redirect URI in `handleGoogleCallback` (`realGoogleVerifier.redirectURI`) with `a.publicURLOrDefault(r)` → verify: `go test ./components/auth/ -run TestGoogleCallbackSPATokenDelivery -v`

4. Replace `r.Host`-based redirect URI in `handlePopupCallback` with `a.publicURLOrDefault(r)` → verify: `go test ./components/auth/ -run TestGoogleCallback -v`

5. Add `TestOAuthPublicURL` — verifies PublicURL takes precedence over Host header → verify: `go test ./components/auth/ -run TestOAuthPublicURL -v`

6. Add `--public-url` flag to CLI in `main.go`, wired to auth.Options.PublicURL → verify: `go build . && go run . serve --help 2>&1 | grep public-url`

7. Run full auth test suite to confirm no regressions → verify: `go test ./components/auth/ -v`

## Verification Script (Step-by-Step)

1. **Verify --public-url flag is accepted**:
   ```bash
   go run . serve --help 2>&1 | grep -i 'public-url'
   # Expected: --public-url string  Public base URL for OAuth redirects (env: BIGBASE_PUBLIC_URL)
   ```
2. **Start BigBase with PublicURL set**:
   ```bash
   BIGBASE_PUBLIC_URL=https://bigbase.click go run . serve --port 9999 --google-client-id test --google-client-secret test &
   sleep 3
   ```
3. **Trigger OAuth flow and inspect the redirect URL** (simulate without actual Google):
   ```bash
   curl -sI -H "Host: evil.com" http://localhost:9999/api/auth/oauth/google 2>&1 | grep -i location
   # Expected: Location contains redirect_uri=https://bigbase.click/api/auth/oauth/google/callback
   # NOT: redirect_uri=http://evil.com/...
   ```
4. **Verify fallback when PublicURL not set**:
   ```bash
   go run . serve --port 9999 --google-client-id test --google-client-secret test &
   sleep 3
   curl -sI http://localhost:9999/api/auth/oauth/google 2>&1 | grep -i location
   # Expected: Location contains redirect_uri=http://localhost:9999/api/auth/oauth/google/callback
   # Logs should show: "msg"="public_url not configured, using request Host header" "host"="localhost:9999"
   ```
5. **Stop servers**:
   ```bash
   kill %1 %2 2>/dev/null
   ```

## Out of scope

- Google Cloud Console redirect URI whitelist update (ops task)
- Validating PublicURL format at startup (could add regex validation in a follow-up)
- Using PublicURL for other Host-dependent features (CORS origins handled separately)
- Per-provider OAuth callback URLs (only Google OAuth currently supported)

## Risks

- **PublicURL misconfiguration**: If PublicURL is set to the wrong value, OAuth flows will fail because Google won't match the redirect URI. Mitigation: document clearly that PublicURL must match the Google Cloud Console registered redirect URI.
- **Dev environment breakage**: Developers who set `BIGBASE_PUBLIC_URL` will have OAuth redirects go to production instead of localhost. Mitigation: the warning log when PublicURL is not set is only logged once at startup; document that dev environments should NOT set this env var.
- **Backward compatibility**: Existing deployments without `BIGBASE_PUBLIC_URL` set will continue to use `r.Host` — but will now see a warning log. Mitigation: the warning is INFO level, not ERROR. Grace period before making it required.

## 13. Definition of Done
- [x] `PublicURL` field added to `Options` and `Auth` struct
- [x] `publicURLOrDefault(r)` helper method implemented
- [x] All three OAuth endpoints use `publicURLOrDefault(r)` instead of raw `r.Host`
- [x] `--public-url` CLI flag wired to `BIGBASE_PUBLIC_URL` env var
- [x] Warning logged when PublicURL not configured (fallback to Host header)
- [x] `TestOAuthPublicURL` verifies PublicURL takes precedence
- [x] All existing OAuth tests pass (`TestGoogleCallback*`)
- [x] `go test ./components/auth/ -v` passes

## 14. Acceptance Criteria (Gherkin)
```gherkin
Feature: OAuth redirect URI uses configured PublicURL

  Scenario: PublicURL takes precedence over Host header
    Given BigBase is configured with BIGBASE_PUBLIC_URL=https://bigbase.click
    When I request GET /api/auth/oauth/google with Host: evil.com
    Then the redirect Location contains redirect_uri=https://bigbase.click/api/auth/oauth/google/callback

  Scenario: Fallback to r.Host when PublicURL not configured
    Given BIGBASE_PUBLIC_URL is not set
    When I request GET /api/auth/oauth/google from localhost
    Then the redirect Location contains redirect_uri=http://localhost:PORT/api/auth/oauth/google/callback
    And a warning is logged

  Scenario: Callback uses PublicURL for token exchange
    Given BigBase is configured with BIGBASE_PUBLIC_URL=https://bigbase.click
    When the OAuth callback handler exchanges the code
    Then the redirect_uri sent to Google is https://bigbase.click/api/auth/oauth/google/callback

  Scenario: Popup callback uses PublicURL
    Given BigBase is configured with BIGBASE_PUBLIC_URL=https://bigbase.click
    When the popup callback handler exchanges the code
    Then the redirect_uri sent to Google is https://bigbase.click/api/auth/oauth/google/callback

  Scenario: Existing OAuth flow works without PublicURL
    Given BIGBASE_PUBLIC_URL is not set
    When I complete the full OAuth flow
    Then the OAuth flow succeeds (existing behavior preserved)
```

## 15. Non-Functional Requirements
- Helper method overhead: single string comparison + optional `fmt.Sprintf` (microseconds)
- Logging: one info-level log per request when PublicURL is not set (could be noisy; consider rate-limiting or logging once at startup instead)

## 16. Test Strategy
- **Unit**: `TestOAuthPublicURL` — table-driven with PublicURL set (Host header ignored), PublicURL unset (Host header used), PublicURL with trailing slash
- **Integration**: Reuse existing `TestGoogleCallback*` tests, adding variants with PublicURL set
- **Regression**: Full `go test ./components/auth/ -v`

## 17. Rollback Plan
- Remove `PublicURL` from `Options` and `Auth` struct
- Revert redirect URI construction to use `r.Host` directly
- Remove `--public-url` flag from `main.go`
- No database changes — pure code rollback
