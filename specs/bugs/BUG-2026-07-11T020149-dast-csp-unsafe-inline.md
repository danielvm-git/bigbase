# CSP: style-src unsafe-inline

**Source:** seal DAST HTTP scan (ZAP) — `https://bigbase.click`
**Seal ID:** 683f4b90-7c6e-4eac-8d5e-83c1c6f07e77
**Severity:** NORMAL
**CWE:** CWE-79 (Improper Neutralization of Input During Web Page Generation)

## Description
Content Security Policy allows `style-src unsafe-inline`, permitting inline style injection. This weakens XSS protections.

## Exploit Scenario
An attacker who finds an injection point can use inline styles to alter page appearance or perform data exfiltration via CSS injection.

## Recommendation
Replace inline styles with CSS classes and set `style-src 'self'` or use a nonce/hash for allowed inline styles. Update CSP in `components/proxy/` security headers.

## Status
triage

## Source
seal.dast_http

## Discovered
2026-07-02
