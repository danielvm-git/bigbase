---
bug_id: BUG-144
status: open
severity: high
scope: auth
title: "Insecure postMessage targetOrigin '*' leaks JWT to any origin in OAuth popup flow"
---

# BUG-144: Insecure postMessage targetOrigin leaks JWT to any origin

## Problem

**File:** `components/auth/anonymous.go:119-133`

When the `redirect` query param on the popup-based Google OAuth flow is absent or not allowlisted, the completed JWT is delivered via `window.opener.postMessage(token, "*")` — deliverable to any origin that opened the popup.

**Actual behavior:** `postMessage(token, "*")` sends the JWT to ANY origin, including attacker-controlled pages.

**Expected behavior:** If no valid `redirect` is present and allowlisted, the popup should NOT send the token via postMessage. It should fail closed.

**Exploit path:**
1. Attacker opens login popup from their page (no `redirect` param or with non-allowlisted redirect)
2. Victim completes Google consent
3. Callback renders HTML with `postMessage(token, "*")`
4. Attacker's `message` listener captures a valid JWT

**Security impact: HIGH** — Full account takeover via JWT theft.

## Root Cause Analysis

### Phase 1: Reproduce

The vulnerability is in `handlePopupCallback` (anonymous.go:119-133):

```go
// Determine target origin for postMessage.
targetOrigin := "*"
if spaRedirect != "" && a.isSPAOriginAllowed(spaRedirect) {
    targetOrigin = spaRedirect[:strings.Index(spaRedirect, "/")]
}

// Render HTML page that does window.opener.postMessage.
html := fmt.Sprintf(`<!DOCTYPE html><html><head><script>
window.opener.postMessage({type:"bigbase-auth:oauth-complete",token:"%s"}, "%s");
window.close();
</script></head><body>Sign in complete. You may close this window.</body></html>`, jwtToken, targetOrigin)
```

**The problem:** `targetOrigin` defaults to `"*"` and only changes if both conditions are met:
1. `spaRedirect` is non-empty
2. `spaRedirect` passes the allowlist check

If either fails, the wildcard remains, and `postMessage(token, "*")` broadcasts the JWT to any origin.

### Phase 2: Isolate

**Affected code path:**
1. `handleGoogleOAuth` sets up the flow with optional `?redirect=` query param
2. User completes Google consent
3. `handlePopupCallback` receives the callback with code + state
4. State is validated, JWT is created
5. **VULNERABILITY:** `targetOrigin` defaults to `"*"` if redirect is missing/invalid

**The secure pattern already exists in the codebase:**
- `handleGoogleCallback` (oauth_handlers.go:140-147) uses HTTP redirect with fragment (`#token=`)
- If `spaRedirect` is missing/invalid, it falls back to setting HttpOnly cookie + redirect to `/admin/`
- This is secure because fragments aren't sent to servers and HttpOnly cookies can't be stolen via XSS

**Why popup flow differs:**
- Popup flow was designed for SPAs that can't use cookies (cross-origin)
- Used `postMessage` as a way to communicate back to the opener
- But the wildcard fallback defeats the purpose of the allowlist

### Phase 3: Hypothesize

**Hypothesis 1:** The code author intended to always have a valid redirect when using popup flow, but didn't enforce it.

**Evidence:** The `handleGoogleOAuth` function (oauth_handlers.go:30-36) only stores `spaRedirect` in signed state if it's non-empty AND allowlisted. Otherwise it uses standard state. But `handlePopupCallback` tries to extract `spaRedirect` from state regardless.

**Hypothesis 2:** The wildcard was intended as a "development mode" fallback but was never removed.

**Evidence:** No environment check — the wildcard is used unconditionally.

### Phase 4: Verified Root Cause

**Root Cause:** `handlePopupCallback` fails open — it sends the JWT via `postMessage` to a wildcard origin when no valid redirect is configured, instead of failing closed.

**Contributing factors:**
1. No test coverage for `handlePopupCallback` (0 tests found)
2. No test for the "no redirect" case in popup flow
3. The secure pattern in `handleGoogleCallback` was not reused

**Risk Level: HIGH** — Active security vulnerability with clear exploit path.

## TDD Fix Plan

### Cycle 1: RED — Test that popup callback without redirect returns error (not postMessage)

**RED:** Write a test that sends a popup callback with NO `redirect` query param and expects:
- Response is NOT a postMessage HTML page
- Response is an error page or JSON error
- No JWT is sent in the response body

**GREEN:** Modify `handlePopupCallback` to return an error response when `spaRedirect` is empty or not allowlisted.

**verify:** `go test ./components/auth/ -run TestPopupCallbackNoRedirect -v`

### Cycle 2: RED — Test that popup callback with non-allowlisted redirect returns error

**RED:** Write a test that sends a popup callback with `redirect=https://evil.com/token` and expects:
- Response is NOT a postMessage HTML page
- Response is an error page or JSON error
- No JWT is sent in the response body

**GREEN:** Ensure `isSPAOriginAllowed` check blocks non-allowlisted origins in popup flow.

**verify:** `go test ./components/auth/ -run TestPopupCallbackNonAllowlistedRedirect -v`

### Cycle 3: RED — Test that popup callback with valid redirect sends correct origin

**RED:** Write a test that sends a popup callback with `redirect=https://bolao.example.com/dashboard` (in allowlist) and expects:
- Response is postMessage HTML page
- `targetOrigin` is `https://bolao.example.com` (not `"*"`)
- JWT is included in the message

**GREEN:** Ensure valid redirect produces correct origin extraction.

**verify:** `go test ./components/auth/ -run TestPopupCallbackValidRedirect -v`

### Cycle 4: RED — Test for Cross-Origin-Opener-Policy header

**RED:** Write a test that checks popup callback responses include:
- `Cross-Origin-Opener-Policy: same-origin` header

**GREEN:** Add the header to popup callback responses.

**verify:** `go test ./components/auth/ -run TestPopupCallbackCOOPHeader -v`

### REFACTOR

After all tests pass:
- Extract the "fail closed" logic into a shared helper if reusable
- Ensure consistent error handling between popup and redirect flows
- Add comments explaining why wildcard origin is never acceptable

## Acceptance Criteria

- [ ] `postMessage` is NEVER called with `"*"` as targetOrigin
- [ ] Popup callback without valid redirect returns error (no JWT leak)
- [ ] Popup callback with non-allowlisted redirect returns error (no JWT leak)
- [ ] Popup callback with valid redirect uses exact origin (not wildcard)
- [ ] `Cross-Origin-Opener-Policy: same-origin` header is set on popup responses
- [ ] All new tests pass
- [ ] Existing auth tests still pass
- [ ] No regression in valid popup flow

## Resolution

**Fixed:** 2026-07-23
**Root cause confirmed:** `handlePopupCallback` failed open — sent JWT via `postMessage(token, "*")` when redirect was missing or not allowlisted, allowing any origin to steal the JWT.
**Fix applied:**
1. Return 403 error page when redirect is missing or not allowlisted (fail closed)
2. Fix origin extraction: use `url.Parse` instead of naive `strings.Index` (was returning `https:` instead of `https://example.com`)
3. Add `Cross-Origin-Opener-Policy: same-origin` header as defense in depth

**Hardening added:**
- Type guard: `url.Parse` validates redirect URL structure
- Fail closed: no postMessage when redirect is missing/invalid
- COOP header: prevents cross-origin opener interaction

**Evidence:** all tests pass (`go test ./... -timeout 300s` — 27 packages)
**Commit:** `fix(auth): GREEN — fail closed on popup postMessage to prevent JWT leak`
**Regression guards:**
- `TestPopupCallbackNoRedirect_FailsClosed`
- `TestPopupCallbackNonAllowlistedRedirect_FailsClosed`
- `TestPopupCallbackValidRedirect_UsesExactOrigin`
- `TestPopupCallbackSetsCOOPHeader`

## References

- [CWE-942: Permissive Cross-domain Policy with Untrusted Domains](https://cwe.mitre.org/data/definitions/942.html)
- [MDN: Window.postMessage()](https://developer.mozilla.org/en-US/docs/Web/API/Window/postMessage)
- [Cross-Origin-Opener-Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cross-Origin-Opener-Policy)
