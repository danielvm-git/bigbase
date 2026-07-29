# CSP: Failure to Define Directive with No Fallback

**Source:** seal DAST HTTP scan (ZAP) — `https://bigbase.click`
**Seal ID:** f787339d-bf72-4b47-a321-69a9041964a7
**Severity:** NORMAL
**CWE:** CWE-1021 (Improper Restriction of Rendered UI Layers or Frames)

## Description
Content Security Policy is missing directives or uses fallbacks that allow unsafe behaviors. The scan found that certain CSP directives have no explicit fallback defined, which can lead to inconsistent policy enforcement across browsers.

## Exploit Scenario
An attacker hosting a malicious subpage or injecting content could bypass CSP restrictions if a directive falls back to a permissive default.

## Recommendation
Define explicit CSP directives for `default-src`, `script-src`, `style-src`, and `connect-src` in the proxy's security headers middleware (`components/proxy/`). Avoid relying on browser fallback behavior.

## Status
fixed

## Resolution

**Fixed:** 2026-07-29

Added `frame-ancestors 'none'` to both strictCSP and permissiveCSP in `components/proxy/securityheaders.go`. This satisfies DAST checks requiring an explicit frame-ancestors directive and provides CSP-level clickjacking protection as defense-in-depth with X-Frame-Options: DENY.

Commit: 9064279ad

## Source
seal.dast_http

## Discovered
2026-07-02
