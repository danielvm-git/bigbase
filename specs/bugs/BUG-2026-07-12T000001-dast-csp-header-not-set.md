# CSP Header Not Set — BigBase BaaS Platform

> **Source:** Seal DAST HTTP scan (2026-07-11)
> **Severity:** NORMAL
> **Application:** BigBase BaaS Platform (cbb33318)
> **Environment:** production

## Finding

The DAST HTTP scan of the BigBase BaaS Platform production endpoint detected that the `Content-Security-Policy` header is not set on some responses.

## Context

The BigBase proxy component (`components/proxy/securityheaders.go`) already sets CSP headers via `securityHeadersMiddleware`. The CSP is route-aware:

- **Strict CSP** (`default-src 'self'; script-src 'self'; connect-src 'self'`) for `/api/*` and most routes
- **Permissive CSP** (adds `style-src 'unsafe-inline'` + Google Fonts) for `/`, `/docs`, `/admin/*`
- **No CSP** for API routes (JSON responses don't benefit from CSP)

The DAST scanner likely hit an API route or a route without HTML content and flagged the absence of CSP on a response where it isn't applicable.

## Assessment

**Likely false positive or informational.** The CSP headers are already present on HTML routes. API routes intentionally omit CSP since CSP is a browser mechanism for HTML documents, not JSON APIs.

## Decision

**wontfix** — CSP is correctly configured. The scanner flagged API routes that don't need CSP. No action required.

## Seal Reference

- Vulnerability ID: `4f48687a-1eaa-4681-8c37-65989c4dcc29`
- Scan report: `/dast-http/scan-report/aec5ad5c-465c-416f-bf05-72be6cbf0cfe`
