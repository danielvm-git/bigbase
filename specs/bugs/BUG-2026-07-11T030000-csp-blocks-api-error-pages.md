---
bug_id: BUG-2026-07-11T030000
status: fixed
severity: high
scope: proxy, infra
title: CSP blocks browser stylesheets on API error pages; deploy pipeline wipes GitHub credentials
---

# BUG-2026-07-11T030000: v2.76.9 CSP regression + GitHub 503

## Problem

After v2.76.9 deploy (Phase 4 DAST CSP hardening):

1. **CSP blocks stylesheets on API error pages:** Navigating to `/api/github/install`
   triggers browser console error:
   `Refused to apply a stylesheet because its hash, its nonce, or 'unsafe-inline' appears in neither the style-src directive nor the default-src directive of the Content Security Policy. (install, line 1)`

2. **GitHub App returns 503:** The GitHub integration shows:
   `{"error":"GitHub App not configured; set --github-app-id, --github-app-slug, --github-app-private-key-path"}`

## Root Cause Analysis

### CSP issue (code regression)

`securityheadersMiddleware` applied `strictCSP` (no `style-src 'unsafe-inline'`
or `font-src`) to ALL routes including `/api/` routes. When the GitHub
`handleInstall` returns a 503 JSON response, the browser renders its built-in
error page which references stylesheets — those get blocked by CSP.

API routes serve JSON (`Content-Type: application/json`), not HTML. CSP is a
browser security mechanism for HTML documents and should not be applied to
JSON API responses.

**Root cause:** Phase 4 DAST fix (commit `3dca1ba8`) applied CSP to all routes
without exempting `/api/` paths.

### GitHub 503 (deploy pipeline design flaw)

`.github/workflows/release-deploy.yml` Phase 4 (lines 278-288) writes
`/opt/bigbase/.env` with ONLY Google OAuth credentials, replacing any
manually-added GitHub App credentials. When `GOOGLE_CLIENT_ID` secret is
empty, the `.env` file is **deleted entirely**.

```bash
if [ -n "${GOOGLE_CLIENT_ID:-}" ] && [ -n "${GOOGLE_CLIENT_SECRET:-}" ]; then
  { echo "GOOGLE_CLIENT_ID=..."; echo "GOOGLE_CLIENT_SECRET=..." } > "${VPS_ENV_FILE}"
else
  rm -f "${VPS_ENV_FILE}"  # DELETES .env, wiping GitHub App creds
fi
```

The GitHub App flags (`--github-app-id`, `--github-app-slug`,
`--github-app-private-key-path`) are NOT passed in the systemd unit file, and
the `.env` file is the only mechanism for GitHub credentials.

## Fix

### CSP fix

`components/proxy/securityheaders.go`: Skip CSP header for `/api/` routes.
API routes serve JSON, which doesn't need CSP. The `Cache-Control` header is
still applied for `/api/` routes.

### Deploy fix

GitHub App credentials must survive deploys. Options:
- Add GitHub App env vars to the release-deploy.yml Phase 4 write block
- Configure systemd unit with GitHub App flags directly
- Use a separate env file for manually-added credentials

## Verification

- [x] `go build ./...` passes
- [x] `go test ./components/proxy/...` — 10/10 tests pass (includes new test
  verifying API routes have no CSP header)
- [ ] Manual verification: deploy and confirm GitHub integration works
- [ ] Manual verification: deploy and confirm CSP errors are gone on API routes

## Resolution

**Fixed:** 2026-07-10
**Root cause confirmed:** CSP applied to JSON API routes; deploy pipeline
overwrites `.env` without preserving GitHub credentials.
**Fix applied:** CSP skipped for `/api/` routes in `securityheaders.go`.
