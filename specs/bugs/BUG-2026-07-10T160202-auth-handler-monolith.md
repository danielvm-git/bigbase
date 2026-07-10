---
bug_id: BUG-2026-07-10T160202
status: fixed
severity: medium
scope: auth
title: auth.go handler monolith split
---

# BUG-2026-07-10T160202: auth.go handler monolith

## Problem

`auth.go` was ~1815 lines mixing middleware, credentials, OAuth, and org HTTP handlers.

## Fix

Split by concern (pure moves, same package):
- `middleware.go` — ProtectedHandler, Middleware, context helpers
- `credentials.go` — register/login/users/me
- `oauth_handlers.go` — Google OAuth + state helpers
- `org_http.go` — orgs, invites, API key HTTP handlers

`auth.go` remains composition root (~500 lines).

## Verify

→ verify: `go test ./components/auth/ -count=1`
→ verify: `test $(wc -l < components/auth/auth.go) -lt 800`
