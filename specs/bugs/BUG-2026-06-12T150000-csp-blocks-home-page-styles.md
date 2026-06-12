---
bug_id: BUG-2026-06-12T150000-csp-blocks-home-page-styles
status: fixed
severity: low
scope: proxy
title: CSP blocks home page styles and Google Fonts
---

# BUG-2026-06-12T150000: CSP `default-src 'self'` blocks home page inline styles and Google Fonts

## Problem

After the security headers middleware was introduced (commit `e430cb4`), the home page at `/` lost all styling and custom fonts. The page renders as unstyled HTML.

**Actual behavior**: Home page renders with no CSS and falls back to system fonts — the layout looks like a plain HTML document.

**Expected behavior**: Home page renders with the full design system (Inter/Fira Code fonts, all CSS variables and rules defined in the inline `<style>` block).

**How to reproduce**: `GET /` and observe the rendered page in a browser, or inspect the `Content-Security-Policy` response header.

## Root Cause Analysis

The `securityHeadersMiddleware` sets:

```
Content-Security-Policy: default-src 'self'
```

`default-src 'self'` is a catch-all that restricts every resource type. For the home page template this breaks two things:

1. **Inline `<style>` blocks** — `style-src` inherits from `default-src`, so inline styles require `'unsafe-inline'` or a nonce. Without it, the browser silently ignores the entire `<style>` block.
2. **Google Fonts** — `font-src` and `style-src` also inherit from `default-src 'self'`, blocking the external `fonts.googleapis.com` / `fonts.gstatic.com` origins.

The middleware applies a single global CSP to **every route** — it has no awareness that the home page is server-rendered with inline styles, while other routes (e.g. `/api/*`, `/health`) serve JSON and have no style dependencies.

**Risk level**: Low — fix is additive (widen CSP directives); the security headers themselves remain in place.

## TDD Fix Plan

1. **RED**: Write a test asserting that `GET /` responds with a `Content-Security-Policy` header that allows `fonts.googleapis.com` and `fonts.gstatic.com` as `style-src` and `font-src` sources, and permits `'unsafe-inline'` for `style-src`.
   **GREEN**: Add explicit `style-src` and `font-src` directives to the CSP in `securityHeadersMiddleware` (or build a route-aware CSP helper).
   **verify**: `go test ./components/proxy/... -run TestCSPAllowsHomeFonts`

2. **RED**: Write a test that GETs `/`, parses the response body, and asserts it contains a non-empty `<style>` tag (proving styles are not stripped by the browser's CSP enforcement — or at minimum that the page delivers the expected stylesheet content).
   **GREEN**: Confirmed by step 1; no additional code change needed if the CSP is fixed.
   **verify**: `go test ./components/proxy/... -run TestHomePageContainsStyleBlock`

3. **RED**: Write a test asserting that API routes (`/health`, `/api/version`) still receive the strict `default-src 'self'` CSP (regression guard so the fix doesn't over-permissive non-HTML routes).
   **GREEN**: Implement a per-route CSP strategy — either check `r.URL.Path` in the middleware or apply different middleware chains per route group.
   **verify**: `go test ./components/proxy/... -run TestSecurityHeaders`

**REFACTOR**: Extract CSP policy constants so `TestSecurityHeaders` and the middleware share a single source of truth, eliminating string duplication.

## Acceptance Criteria

- [x] `GET /` delivers a CSP that allows `fonts.googleapis.com` and `fonts.gstatic.com` under `style-src` and `font-src`
- [x] `GET /` CSP allows `'unsafe-inline'` for `style-src` (or uses nonces — but inline `<style>` must be rendered)
- [x] API and health routes still receive the strict `default-src 'self'` policy
- [x] Google Fonts load on the home page (Inter and Fira Code visible)
- [x] All existing security header tests pass
- [x] All new tests pass

## Resolution

Fixed by making `securityHeadersMiddleware` route-aware.
- Extracted `strictCSP` and `permissiveCSP` constants.
- Added logic to apply `permissiveCSP` to `/` and `/docs` routes.
- Added comprehensive tests in `components/proxy/securityheaders_test.go`.

