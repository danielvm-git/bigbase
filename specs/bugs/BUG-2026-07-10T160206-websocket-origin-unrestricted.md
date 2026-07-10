---
bug_id: BUG-2026-07-10T160206
status: fixed
severity: low
scope: realtime
title: WebSocket accepts all origins
---

# BUG-2026-07-10T160206: WebSocket CheckOrigin unrestricted

## Problem

**Security impact: MEDIUM** — any Origin could upgrade to WebSocket.

## Fix

Per-instance CheckOrigin: allow empty Origin, configured AllowedOrigins (wired from CORS flags), and same-origin; deny others.

## Verify

→ verify: `go test ./components/realtime/ -run TestRealtimeCheckOrigin -count=1`
