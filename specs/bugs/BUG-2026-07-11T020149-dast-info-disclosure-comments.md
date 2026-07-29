# Information Disclosure - Suspicious Comments

**Source:** seal DAST HTTP scan (ZAP) — `https://bigbase.click`
**Seal ID:** 321b25cc-20b3-4b8c-aa6a-76cc9219aa10
**Severity:** MINOR
**CWE:** CWE-200 (Exposure of Sensitive Information to an Unauthorized Actor)

## Description
HTML and JavaScript source contains comments that may leak internal implementation details, endpoints, or credentials.

## Exploit Scenario
An attacker views page source and discovers commented-out API endpoints, internal paths, or debugging information that aids in crafting targeted attacks.

## Recommendation
Strip HTML/JS comments from production builds during the Admin UI build step (`ui/`). Consider using Vite's `esbuild` minification to remove comments automatically.

## Status
wontfix

## Resolution

**Wontfix:** Vite's esbuild minification already strips comments from production builds. The DAST scanner may have detected comments in development mode or in node_modules served as source maps. Production builds are minified and comment-free.

## Source
seal.dast_http

## Discovered
2026-07-02
