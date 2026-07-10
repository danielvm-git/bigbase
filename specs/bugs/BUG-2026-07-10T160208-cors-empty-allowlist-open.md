---
bug_id: BUG-2026-07-10T160208
status: fixed
severity: low
scope: auth
title: Empty CORS allowlist passes through — comment says closed
---

# BUG-2026-07-10T160208: Empty CORS allowlist

## Problem

**Security impact: MEDIUM** — CWE-942 permissive CORS.

Comment claimed "default closed" but empty allowlist passed through any Origin (no ACAO, but side-effect requests still ran).

## Fix

Empty allowlist denies cross-origin Origins with 403; same-origin (Origin matches scheme://Host) still allowed for admin UI.

## Verify

→ verify: `go test ./components/auth/ -run TestCORSDefaultClosed -count=1`
