# esbuild (npm) — Dev server request smuggling

**Source:** GHS Dependabot
**Seal IDs:** 55819356, 7bc72fb1
**Severity:** NORMAL (5.3)
**Ecosystem:** npm (ui/)
**GHSA:** GHSA-67mh-4wv8-2f99

## Description
Any website can send requests to the esbuild development server and read the response. This is a development-time vulnerability.

## Recommendation
Update esbuild to the latest version. Not exploitable in production since esbuild dev server is not exposed.

## Status
triage

## Source
seal.github_dependabot

## Discovered
2026-07-11
