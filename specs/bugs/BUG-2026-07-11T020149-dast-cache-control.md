# Re-examine Cache-control Directives

**Source:** seal DAST HTTP scan (ZAP) — `https://bigbase.click`
**Seal ID:** 3bddac93-060e-47e1-addd-05d5e3222c79
**Severity:** MINOR
**CWE:** CWE-525 (Use of Web Browser Cache Containing Sensitive Information)

## Description
HTTP responses lack or have suboptimal `Cache-Control` headers, potentially allowing sensitive data to be cached in the browser.

## Exploit Scenario
A user logs into BigBase on a shared computer; their session data or API responses remain in browser cache and are accessible to the next user.

## Recommendation
Set `Cache-Control: no-store, no-cache, must-revalidate` on all API responses. Only allow caching on static assets with versioned URLs.

## Status
triage

## Source
seal.dast_http

## Discovered
2026-07-02
