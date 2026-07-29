# Vitest UI (npm) — Arbitrary file read/execute

**Source:** GHS Dependabot
**Seal IDs:** 7cb92e12, f4a9e94a
**Severity:** CRITICAL (9.8)
**Ecosystem:** npm (ui/)
**CVE:** CVE-2026-47429
**GHSA:** GHSA-5xrq-8626-4rwp

## Description
When Vitest UI server is listening, arbitrary file can be read and executed. This is a development-time vulnerability.

## Recommendation
Update vitest to the latest version. Not exploitable in production since Vitest UI is not exposed.

## Status
fixed

## Source
seal.github_dependabot

## Discovered
2026-07-11
