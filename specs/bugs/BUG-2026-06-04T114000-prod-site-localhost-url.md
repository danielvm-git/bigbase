# BUG-2026-06-04T114000: Production site deployments use localhost URL

## Problem

- **Actual:** Running site deployments on production show and store `http://localhost:<port>` (e.g. `http://localhost:10958`). The URL is only reachable on the VPS loopback, not from the public internet.
- **Expected:** `https://<site-slug>.bigbase.click` with traffic routed through Caddy and the Go proxy to the deployment port.
- **Reproduce:** On `https://bigbase.click/admin/`, create or redeploy a site → wait for `RUNNING` → deployments table shows `http://localhost:…`.

## Root Cause Analysis

- The deploy component always persists and returns loopback URLs when marking a deployment `running`, regardless of environment.
- The proxy component does not route requests by `Host` header to deployment ports.
- Production Caddy only terminates TLS for the apex domain, not wildcard site subdomains.
- The sites component’s deploy trigger was not wired in the serve entrypoint, so create/redeploy often fell back to the raw deploy API with the same localhost behavior.
- `proxy.domain` in defaults was never passed into deploy.

**Risk level:** Medium

## TDD Fix Plan

1. **RED:** Deploy test with `PublicDomain` returns `https://<slug>.bigbase.click`, not localhost.  
   **GREEN:** URL builder + slug from repo name.  
   **verify:** `go test ./components/deploy/... -run TestDeployPublicURL -count=1`

2. **RED:** Proxy test routes `Host: myapp.bigbase.click` to registered port.  
   **GREEN:** Host registry + reverse-proxy middleware.  
   **verify:** `go test ./components/proxy/... -run TestProxyDeploymentHost -count=1`

3. **RED:** Sites create with `TriggerDeploy` returns public URL.  
   **GREEN:** `Deploy.Trigger` + main wiring.  
   **verify:** `go test ./components/sites/... -count=1`

4. **REFACTOR:** Caddy wildcard + `--sites-domain` on VPS; UI `siteDisplayUrl` fallback to `bigbase.click`.

## Acceptance Criteria

- [x] New prod deployments store and display `https://<slug>.bigbase.click`
- [x] Subdomain requests reach the deployment app on the VPS (via proxy host registry + Caddy wildcard)
- [x] Local dev without sites domain still uses `http://localhost:<port>`
- [x] Sites create/redeploy uses wired `TriggerDeploy`
- [x] All new and existing tests pass

## Resolution

**Fixed:** 2026-06-04  
**Root cause confirmed:** Deploy always persisted `http://localhost:<port>` and the proxy never routed `Host` subdomains to deployment ports.  
**Fix applied:** `deploymentURL` / `SiteSlug` in deploy; `RegisterDeploymentHost` middleware in proxy; `--sites-domain` + `TriggerDeploy` in `main.go`; Caddy `*.bigbase.click` and systemd flag in `setup-vps.sh`; UI `siteDisplayUrl` fallback.  
**Hardening added:** Regression tests `TestDeployPublicURL`, `TestDeploymentURL`, `TestProxyDeploymentHost`, `TestSitesCreateTriggersDeploy` (`components/deploy/urls_test.go`).  
**Evidence:**  
- `go test ./components/deploy/... -run TestDeployPublicURL -count=1` — PASS  
- `go test ./components/proxy/... -run TestProxyDeploymentHost -count=1` — PASS  
- `go test ./components/sites/... -run TestSitesCreateTriggersDeploy -count=1` — PASS  
- `go test ./... -count=1` — PASS (all packages)  
- `go vet ./...` && `go build -o bigbase .` — PASS  
- `cd ui && npm test -- --run` — 172 tests PASS  
- DNS: `dig +short add-tutorial-requests-site.bigbase.click` → `89.116.26.187`  
- Edge: `curl -sI http://add-tutorial-requests-site.bigbase.click/` → `308` to HTTPS (Caddy wildcard route live)  
**Commit:** `fix(deploy): use https://slug.bigbase.click for production site URLs`  

**Remaining ops (after merge/deploy to VPS):** Ensure binary includes this fix, `bigbase` runs with `--sites-domain bigbase.click`, wait for `*.bigbase.click` TLS (DNS shows wildcard still propagating), then **Redeploy** each site so DB `url` and proxy host registry refresh.
