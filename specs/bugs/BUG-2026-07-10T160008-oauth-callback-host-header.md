---
bug_id: BUG-2026-07-10T160008
status: fixed
severity: high
scope: auth
title: OAuth callback paths use PublicURLOrDefault Host fallback
---

# BUG-2026-07-10T160008: OAuth callback Host header fallback

## Problem

**Security impact: HIGH** — CWE-601 open redirect / OAuth redirect URI poisoning.

`PublicURLOrDefault` fell back to `r.Host` when `publicURL` was empty. OAuth start/callback/link/popup paths used that helper for Google `redirect_uri`, so an attacker-controlled Host could poison the callback URL in test mode or any misconfigured deploy that skipped the New() panic guard.

## Root cause

Host fallback in `PublicURLOrDefault` after BUG-160001 only guarded production New() when Google OAuth was enabled; the helper itself still trusted Host.

## Fix approach

1. `PublicURLOrDefault` returns only configured `publicURL` (never Host).
2. `oauthCallbackRedirectURI()` fails closed when unset; OAuth handlers return 503.

## Verify

→ verify: `go test ./components/auth/ -run 'TestOAuthPublicURL|TestOAuthStartRequires' -count=1`
→ verify: `go test ./components/auth/ -count=1`
