# Sub Resource Integrity Attribute Missing

**Source:** seal DAST HTTP scan (ZAP) — `https://bigbase.click`
**Seal ID:** 9cbaa2bf-26ee-4f23-8715-c14a2d13bf9d
**Severity:** NORMAL
**CWE:** CWE-345 (Insufficient Verification of Data Authenticity)

## Description
External resources loaded by the Admin UI (scripts, stylesheets) are fetched without Subresource Integrity (SRI) hashes. A compromised CDN could serve malicious content.

## Exploit Scenario
An attacker compromises the CDN hosting a JavaScript library used by the Admin UI, injecting malware served to all BigBase admin users.

## Recommendation
Add `integrity` attributes to all `<script>` and `<link>` tags loading external resources. For bundled assets, ensure the build process generates integrity hashes.

## Status
triage

## Source
seal.dast_http

## Discovered
2026-07-02
