# BUG-2026-06-21T153000: Process-based apps not resumed after BigBase restart

## Problem

- **Actual**: After any BigBase redeploy (including e39s01, e39s02, the revert), Python/Go/Node SSR apps that were running as child processes are killed but never restarted. Their proxy routes are still registered (so requests arrive), but nothing is listening on the app port → proxy returns "deployment unavailable" (HTTP 502).
- **Expected**: Process-based apps should restart from the existing build directory when BigBase starts, mirroring what static sites already do.
- **Trigger**: Every BigBase upgrade cycle kills child processes. The e39 deploy/revert cycle (three restarts in quick succession) exposed this in production with bolao.bigbase.click.

## Root Cause Analysis

The deploy component's `Stop()` kills all child processes tracked in `d.apps`. On `Start()`, `restoreRunningDeploymentHosts()` re-registers proxy routes for deployments with `status = 'running'`, and `resumeCandidates()` re-starts the apps.

However, `resumeCandidates()` only handled static apps (including Node builds that output to `dist/` or `build/`). Non-static apps — Python, Go, Node SSR — hit an unconditional `continue` and were silently dropped. The proxy route was registered but no process was started, causing every request to the site to fail with 502.

**Risk level**: High — affects all process-based sites on every BigBase upgrade.

## TDD Fix Plan

1. **RED**: Test that after a simulated BigBase restart (stop k1, start k2 with same DB), a "running" process-based deployment is restarted (port accepts connections or startApp is called).
   **GREEN**: In `resumeCandidates`, replace `if appType != AppStatic { continue }` with a call to `startApp` for process-based apps, passing the correct port and build directory.
   **verify**: `go test ./components/deploy/ -timeout 120s`

**REFACTOR**: None needed — change is minimal and self-contained.

## Acceptance Criteria

- [x] `resumeCandidates` calls `startApp` for Python, Go, and Node SSR deployments
- [x] All 96 existing deploy tests pass
- [x] Code change is one logical block (no collateral changes)
- [x] Deployed to production via GitHub Actions (commit dc5b5c6)

## Resolution

**Fixed:** 2026-06-21  
**Root cause confirmed:** `resumeCandidates()` had an unconditional `continue` for non-static apps — they were killed by `Stop()` and never restarted.  
**Fix applied:** Replaced `if appType != AppStatic { continue }` with a `startApp()` call using the existing build directory and original port. Static sites retain their `serveStatic` path.  
**Hardening added:** `TestResumeCandidatesAttemptsProcessApps` — seeds a "running" Go deployment, simulates restart with a fresh Deploy, asserts status transitions from "running" to "failed" (proving `startApp` was called, not skipped).  
**Evidence:** 662 tests pass (`go test ./... -timeout 120s`); `https://bolao.bigbase.click/` returns HTTP 200.  
**Commits:** `dc5b5c6` (fix), `<hardening commit>` (regression test)
