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

### Bug B: Health check returns 502 immediately (environment/transient)
- **Module**: Components involved: proxy (routes `/api/monitoring/health`), monitoring (handler returns 200)
- **Mechanism**: The health handler in the monitoring component explicitly returns `HTTP 200` with `{"status":"ok"}`. The proxy's `loggingMiddleware` captures 502, meaning something in the middleware or mux chain returns 502 before reaching the handler. Response time of ~500µs rules out timeouts.
- **Hypotheses** (ranked by probability):
  1. **Route shadowing**: A prior deploy might have registered deployment hosts that shadow `/api/monitoring/health` on certain host headers (unlikely for `localhost`).
  2. **Mux registration order**: The proxy's `Start()` registers `/` (catch-all) after `main.go` registers `/api/monitoring/health`. On Go 1.26 `http.ServeMux`, this should still work (longest-prefix wins), but the order could matter if the mux behavior changed.
  3. **Transient environment issue**: The VPS may have been under load or the Caddy frontend interfered (though the health check goes directly to port 8080).
- **Risk**: High — without SSH access to the VPS, the root cause of the 502 cannot be definitively confirmed. The fix for Bug A makes the pipeline more robust regardless.

### Prior related bugs
- `BUG-2026-06-04T114000` (production site URL bug) also involved deploy script changes.
- `BUG-2026-06-04T120500` and `BUG-2026-06-04T120800` involved Node/VPS deploy failures.
- This is **novel** — previous deploy bugs were about missing tools or wrong URLs, not a healthy service failing health checks.

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

<!-- filled in by validate-fix -->
