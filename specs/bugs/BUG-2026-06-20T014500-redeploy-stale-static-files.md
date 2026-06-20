# BUG-2026-06-20T014500: Redeployment does not replace static files — old content persists

## Problem

Triggering a new deployment for an existing site (via `POST /api/sites/{id}/deploy` or `POST /api/deploy`) creates a new build and starts a new static server on a new port, but traffic at the domain URL continues to serve old content from the previous deployment.

### Symptoms
- Dashboard shows deployment status `running` with the new commit SHA
- The domain returns HTML/JS/CSS from an older commit (verified by asset hash fingerprint)
- Multiple redeployments accumulate: each creates a new deployment record and a new listening port, but none replace the previous one
- Old static servers remain running, leaking ports

### Actual vs Expected
| | Actual | Expected |
|---|---|---|
| Proxy routes to | First deployment's port (locked at first registration) | Latest deployment's port |
| Old servers after redeploy | Keep running forever | Stopped and cleaned up |
| Deployment records | Multiple records, all `running`, same URL, different ports | Previous record marked `replaced`, only one serving |

### Reproduction
1. Deploy site A → works, port P1 registered in proxy
2. Push new commit → trigger new deploy → port P2 assigned, app starts
3. `RegisterDeploymentHost` for P2 fails ("host already registered")
4. Site URL still serves from P1 (content from commit 1)
5. Repeat steps 2-4 → accumulating ghost servers on P3, P4, P5...

## Root Cause Analysis

The redeployment `Trigger` function creates a fresh deployment with no awareness of existing running deployments for the same site. The failure chain is:

1. **Trigger** allocates a new port and creates a new deployment record
2. **runDeployment** → **finalizeDeploymentURL** calls `RegisterDeploymentHost(host, newPort, siteID)`
3. **RegisterDeploymentHost** rejects the call: `"host %q already registered"` because the previous deployment's port is still registered
4. The error is logged as `Warn` only — execution continues
5. The proxy continues to route traffic to the **first-ever registered port** for that host
6. The old static server on that port still serves old build artifacts
7. The new deployment's server runs on its new port, unreachable from the domain

Two missing behaviors:
- **No stop-previous**: `Trigger` doesn't look up and shut down existing running deployments for the same site before starting a new one
- **No replace-semantics in host registration**: `RegisterDeploymentHost` refuses to update an existing mapping; there's no equivalent "ReplaceDeploymentHost" or "update-if-exists" path
- **No port cleanup**: Old static servers accumulate indefinitely, leaking file descriptors

### Risk level: High
Every redeployment from a site's dashboard silently fails to take effect. Users see "success" in the UI but get old content. Ports leak, consuming server resources.

## TDD Fix Plan

### Cycle 1 — Stop previous deployment before starting new one

**RED**: Write a test `TestRedeployReplacesPrevious` that:
- Creates deployment A for a site → registers host
- Creates deployment B for same site (via `Trigger` or `HandleCreate`)
- Asserts that after B completes, A's deployment status is changed (e.g., `replaced` or `stopped`)
- Asserts that the proxy host now points to B's port, not A's

**GREEN**: In `Trigger`, before inserting the new deployment record:
1. Query for existing running deployments with the same `site_id` (or same `repo_id` if no `site_id`)
2. For each match:
   a. Stop the running app process / static server (from `d.apps` map)
   b. Unregister the host from the proxy router
   c. Update DB status to `"replaced"`
3. Then proceed with creating the new deployment as before

Make `finalizeDeploymentURL` safe to succeed after the old host was unregistered.

**verify**: `go test ./components/deploy/... -run TestRedeployReplacesPrevious`

### Cycle 2 — Stop-previous also works for process-based apps (Node/Go/Python)

**RED**: Write `TestRedeployReplacesPreviousProcessApp` — same as Cycle 1 but with a Node/Go app type where `startApp` manages the process. Verify old process is killed.

**GREEN**: The stop-previous logic already handles this via the `d.apps` map (which stores both `server` and `cmd`). Ensure the generic "stop deployment" helper handles both static and process types.

**verify**: `go test ./components/deploy/... -run TestRedeployReplacesPreviousProcessApp`

### Cycle 3 — RegisterDeploymentHost supports replacing existing mapping

**RED**: Write `TestRegisterDeploymentHostReplace` on the proxy host router interface that:
- Registers host `test.example.com → port 9001`
- Registers same host with `test.example.com → port 9002`
- Asserts no error, and that the second registration replaces the first

**GREEN**: Modify `RegisterDeploymentHost` to update the port if the host is already registered (instead of returning an error). The siteID should also be updated.

Alternatively, add a new method `ReplaceDeploymentHost` that does the same but is explicit.

**verify**: `go test ./components/proxy/... -run TestRegisterDeploymentHostReplace`

### REFACTOR

- Extract a `d.stopDeployment(id string)` helper that encapsulates the stop/unregister/cleanup logic currently duplicated in `handleDeleteDeployment` and `Stop()`.
- Use it in `Trigger` during the stop-previous phase.

## Acceptance Criteria

- [x] Redeploying a site serves new content (verified by asset hash or content check)
- [x] Previous deployment's static server is shut down after redeploy
- [x] Previous deployment's proxy host mapping is released
- [x] Previous deployment's DB status updated to `"replaced"`
- [x] Old ports are freed / not leaked
- [x] All existing tests pass
- [x] Regression: fresh deployments (no prior deployment) still work

## Resolution

**Fixed:** 2026-06-20
**Root cause confirmed:** `RegisterDeploymentHost` rejected re-registration of an existing host with a different port (returned "host already registered" error). The error was silently logged as Warn, so new deployments appeared to succeed in the UI but the proxy continued routing to the old port. Ghost deployments accumulated on unreachable ports.
**Fix applied:**
1. `RegisterDeploymentHost` in proxy now replaces existing host→port mapping instead of erroring
2. New `stopDeployment` helper kills process, shuts down server, unregisters host, marks `replaced`
3. `stopPreviousDeployments` called from `Trigger` stops old deployments for same site/repo before creating new one
4. `deploymentHostMiddleware` forwards `/api/*` paths to BigBase (allows SPA same-origin API calls through dynamic proxy)
5. Hardcoded Caddy entry for `bolao.bigbase.click` removed — uses `*.bigbase.click` wildcard + dynamic proxy
**Hardening added:**
- `TestRedeployReplacesPrevious` — integration test deploying twice to same site, verifying proxy mapping replacement and status change
- `TestRegisterDeploymentHostReplacesExisting` — direct test that re-registration succeeds and updates port
- `TestDetectAppTypePythonRequirementsOnlyFallsToStatic` — prevents false Python detection
- `deploymentHostMiddleware` API passthrough — prevents SPA API routing failures
**Evidence:** all tests pass (`go test ./... -timeout 180s`)
**Commit:** `fix(deploy,proxy): redeployment now replaces previous — no more stale content (#35)` + `fix(proxy): forward /api/* to BigBase for deployment hosts; export hostInfo`
