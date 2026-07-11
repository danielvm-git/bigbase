# Open URL redirect

**Source:** GHS Code Scanning (CodeQL)
**Severity:** NORMAL
**CWE:** CWE-601 (URL Redirection to Untrusted Site)
**GitHub Alert:** #4

## Description
CodeQL detected an open redirect where user-controlled input determines the redirect target via `http.Redirect(w, r, spaURL, http.StatusFound)` in `magiclink.go:157`. The `redirectTo` value from `?redirect_to=` is concatenated into a `#token=` redirect URL, passed through `isSPAOriginAllowed`.

## Original Weakness
The guard function `isSPAOriginAllowed` used `strings.HasPrefix(redirectURL, origin)` to validate against an allowlist. This is vulnerable to subdomain bypass: given allowlist entry `https://trusted.example.com`, the URL `https://trusted.example.com.evil.com/phish` would pass the check because `strings.HasPrefix` only checks string prefix, not origin boundaries.

## Fix Applied
Changed `isSPAOriginAllowed` in `components/auth/auth.go` from `strings.HasPrefix` to proper URL parsing and origin comparison:
1. Parse the redirect URL with `url.Parse`
2. Relative URLs (no host) are always safe — allowed
3. For absolute URLs, extract `scheme + host` and compare exactly with each allowlist entry
4. Reject protocol-relative URLs (`//evil.com`) — scheme won't match allowlist entries

This prevents the `strings.HasPrefix` subdomain bypass while maintaining backward compatibility for relative URLs and legitimate SPA origins.

## Status
fixed

## Source
seal.github_code_scanning

## Discovered
2026-07-11

## Resolution
2026-07-11

## Files Changed
- `components/auth/auth.go` — `isSPAOriginAllowed` rewritten with proper URL origin comparison
- `components/auth/oauth_redirect_test.go` — added `TestGoogleCallbackSPARejectsSubdomainBypass` regression guard

## Regression Guards
- `components/auth/oauth_redirect_test.go TestGoogleCallbackSPARejectsSubdomainBypass`
- `components/auth/oauth_redirect_test.go TestGoogleCallbackSPARejectedOrigin`
- `components/auth/oauth_redirect_test.go TestGoogleCallbackSPATokenDelivery`
