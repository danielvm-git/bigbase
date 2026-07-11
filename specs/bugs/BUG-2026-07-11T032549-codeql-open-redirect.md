# Open URL redirect

**Source:** GHS Code Scanning (CodeQL)
**Severity:** NORMAL
**CWE:** CWE-601 (URL Redirection to Untrusted Site)
**GitHub Alert:** #4

## Description
CodeQL detected an open redirect where user-controlled input determines the redirect target. An attacker could use this to phish users by redirecting them to malicious sites.

## Recommendation
Validate redirect URLs against an allowlist of trusted domains. Return a 400 error for untrusted redirect targets instead of following them.

## Status
triage

## Source
seal.github_code_scanning

## Discovered
2026-07-11
