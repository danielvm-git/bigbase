# User Agent Fuzzer

**Source:** seal DAST HTTP scan (ZAP) — `https://bigbase.click`
**Seal ID:** f72fbf7d-e4ca-4894-b99a-9d22e81bb53d
**Severity:** MINOR
**CWE:** CWE-200 (Exposure of Sensitive Information to an Unauthorized Actor)

## Description
The server responds differently based on User-Agent header values, potentially revealing behavioral differences that an attacker can exploit.

## Exploit Scenario
An attacker fuzzes User-Agent headers to discover device-type-specific vulnerabilities or bypass access controls that rely on User-Agent checks.

## Recommendation
Ensure consistent responses across all User-Agent values unless there is a deliberate feature-flag mechanism. Avoid User-Agent-based access control.

## Status
triage

## Source
seal.dast_http

## Discovered
2026-07-02
