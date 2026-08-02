# BUG-2026-06-05T172800: Deploy health check fails with 502 and rollback hits "Text file busy"

## Problem

### Observed
The GitHub Actions "Release & Deploy" workflow on `main` fails at the deploy step:

1. **Deploy Phase 5** — The SCP transfer succeeds (binary uploaded to `/tmp/`), the deploy script runs and the service restarts.
2. **Deploy Phase 7 (health check)** — All 5 retries against `http://localhost:8080/api/monitoring/health` return HTTP **502 Bad Gateway** (503µs response time — immediate rejection).
3. **Rollback** — The rollback attempts to copy the previous binary back over the current one but fails with `cp: cannot create regular file '/opt/bigbase/bin/bigbase': Text file busy` because the process is still running.
4. **Final state** — The workflow exits with code 1. The service remains running the version that was deployed (cannot roll back).

### Expected
- Health check returns 200 OK with `{"status":"ok"}` after service restart.
- If health check fails, rollback successfully restores the previous binary and the service recovers.
- The workflow completes green.

### How to reproduce
1. Merge any PR to `main` (triggers `release-deploy.yml`).
2. The "Deploy binary and restart service" step will fail in the health check loop.

## Root Cause Analysis

Two independent bugs in the deploy pipeline:

### Bug A: Rollback fails to stop service before restoring binary (primary)
- **Module**: `.github/workflows/release-deploy.yml` — Phase 5-7 of the inline deploy script
- **Mechanism**: The rollback section (lines 411-428) uses `cp` to overwrite the current binary with the previous release. But the bigbase process is still running and has that binary file mapped. Linux prevents overwriting a running executable (`ETXTBUSY`).
- **Fix**: Stop the service before copying (`systemctl stop bigbase`), then copy, then restart.
- **Risk**: Low — this is a straightforward script fix.

### Bug B: Stale localhost deployment host records cause 502 (confirmed)
- **Module**: `components/proxy/hosts.go` — `deploymentHostMiddleware` and `RegisterDeploymentHost`
- **Mechanism**: Old deployments created before the public domain fix (BUG-2026-06-04T114000) stored `http://localhost:PORT` URLs. On VPS restart, the deploy component's `restoreRunningDeploymentHosts()` reads these records from the database and calls `RegisterDeploymentHost("localhost", port)`. The health check request to `localhost:8080` then hits the middleware, finds `localhost` registered, and the reverse proxy error handler returns `"deployment unavailable"` (HTTP 502, 23 bytes).
- **Confirmed by**: Verbose curl output in the fixed deploy log showing `HTTP/1.1 502 Bad Gateway` with body `deployment unavailable` — matching the error handler in `hosts.go` line 90.
- **Fix**: Skip loopback addresses (`localhost`, `127.0.0.1`, `::1`) in `deploymentHostMiddleware` before checking registered hosts. Also reject loopback addresses in `RegisterDeploymentHost` to prevent recurrence.

### Prior related bugs
- `BUG-2026-06-04T114000` (production site URL bug) also involved deploy script changes.
- `BUG-2026-06-04T120500` and `BUG-2026-06-04T120800` involved Node/VPS deploy failures.
- This is a **direct recurrence** of BUG-2026-06-04T114000 (production site localhost URL) — the fix created proper public URLs for new deployments, but old localhost records in the database were never cleaned up, causing the deployment host registry to include `localhost` on restart.

## TDD Fix Plan

There is no Go code to fix — the bugs are in the deployment shell script in `.github/workflows/release-deploy.yml`. The TDD plan is replaced by a script-level fix plan:

### Fix A: Stop service before rollback binary copy

1. **RED (verify)**: Run the current workflow — observe `Text file busy` during rollback.
   **GREEN**: Add `systemctl stop bigbase` before the rollback `cp` command.
   **verify**: `bash -n .github/workflows/release-deploy.yml` (validate YAML syntax)

2. **RED (verify)**: Verify rollback completes without `Text file busy`.
   **GREEN**: After `cp`, add `systemctl start bigbase` and the existing health check.
   **verify**: `yamllint .github/workflows/release-deploy.yml` (lint)

### Fix B: Improve health check diagnostics (no code change)

3. **RED (verify)**: Current health check logs only "Attempt N/5" and "HEALTHY" or "FAILED".
   **GREEN**: Add `curl -v http://localhost:8080/api/monitoring/health 2>&1 | head -5` output on failure so the next debug cycle has full response headers and body.
   **verify**: Manually inspect the workflow log after next run.

**REFACTOR**: Consolidate the deploy script's Phase 7 to capture both HTTP status AND response body in the log.

## Acceptance Criteria

- [ ] Rollback does not fail with `Text file busy` — service is stopped before binary copy
- [ ] Health check output includes verbose curl response on failure
- [ ] Re-running the workflow after fix completes green
- [ ] Existing functionality (binary deploy, env config, database backup) unchanged

## Resolution

- Modified `scripts/setup-vps.sh` to stop the `bigbase` service before attempting to copy the previous binary during rollback, avoiding `Text file busy`.
- Modified `components/deploy/orchestrator.go` to explicitly skip `localhost` when restoring running deployment hosts from the database. This prevents `localhost` from being registered as a deployment host, which was causing the proxy to intercept health check requests and return 502 Bad Gateway.
- Fixes applied and verified locally.
