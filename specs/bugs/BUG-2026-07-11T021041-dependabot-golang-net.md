# Go net/html — Denial of Service

**Source:** GHS Dependabot
**Seal ID:** 502a0962-29bc-4739-b140-5d3e4aa01d5c
**Severity:** NORMAL (6.5)
**Ecosystem:** Go (gomod)
**CVE:** CVE-2026-25680
**GHSA:** GHSA-5cv4-jp36-h3mw

## Description
Go standard library net/html HTML parser is vulnerable to denial of service via crafted input. This affects any component that parses HTML, including the Admin UI rendering pipeline.

## Exploit Scenario
An attacker uploads or serves a crafted HTML page that causes the parser to hang or consume excessive CPU/memory, leading to service degradation.

## Recommendation
Update Go to a version that patches CVE-2026-25680 (Go 1.26.x or later).

## Status
fixed

## Source
seal.github_dependabot

## Discovered
2026-07-11

## Resolution

**Fixed:** Dependabot PR merged. CI green.
