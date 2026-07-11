# Cookie 'Secure' attribute is not set to true (3 instances)

**Source:** GHS Code Scanning (CodeQL)
**Severity:** NORMAL
**CWE:** CWE-614 (Sensitive Cookie in HTTPS Session Without 'Secure' Attribute)
**GitHub Alerts:** #1, #2, #3

## Description
CodeQL detected 3 cookies set without the `Secure` flag. These cookies are transmitted over unencrypted HTTP connections, making them susceptible to network interception.

## Recommendation
Set `Secure: true` on all cookies when the connection is HTTPS. The existing `cookieSecure()` helper in the auth component should be used consistently.

## Status
triage

## Source
seal.github_code_scanning

## Discovered
2026-07-11
