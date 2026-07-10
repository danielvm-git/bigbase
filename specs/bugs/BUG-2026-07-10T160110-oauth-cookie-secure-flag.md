---
bug_id: BUG-2026-07-10T160110
status: fixed
severity: medium
scope: auth
title: OAuth/token cookies use Secure: r.TLS != nil behind TLS proxy
---

# BUG-2026-07-10T160110: Cookie Secure behind TLS proxy

## Problem

**Security impact: MEDIUM** — CWE-319 cleartext transmission of sensitive cookie.

Auth cookies used `Secure: r.TLS != nil`. Behind Caddy/nginx TLS termination, `r.TLS` is nil so session cookies were sent without Secure.

## Fix

`cookieSecure(r)` returns true when `r.TLS != nil`, PublicURL is `https://…`, or `X-Forwarded-Proto: https`.

## Verify

→ verify: `go test ./components/auth/ -run TestCookieSecure -count=1`
