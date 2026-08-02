# e84s02 — Proxy per-site CSP override

## Summary
The proxy middleware chain sets `permissiveCSP` before identifying the site. After the site
is resolved in `deploymentHostMiddleware`, override the CSP header with a site-specific policy
when one is configured. Changes are picked up live via the `UpdateCSP`→`SetSiteCSP` seam.

## Changes

### `components/proxy/hosts.go`
- Add `CSPPolicy string` to `hostInfo` struct
- In `RegisterDeploymentHost`: query `deploy_defaults` from `sites` table, unmarshal,
  populate `info.CSPPolicy` and `p.cspPolicies[siteID]`
- In `deploymentHostMiddleware`: after auth check, before `url.Parse(...)`:
  ```go
  if csp := p.getSiteCSP(info.SiteID); csp != "" {
      w.Header().Set("Content-Security-Policy", csp)
  }
  ```

### `components/proxy/proxy.go`
- Add `cspPoliciesMu sync.RWMutex` and `cspPolicies map[string]string` to `Proxy` struct
- Initialize `cspPolicies: make(map[string]string)` in `New()`
- Add `SetSiteCSP(siteID, cspPolicy string)` (mirror of `SetSiteAuthPolicy`)
- Add `getSiteCSP(siteID string) string` with in-memory cache + DB fallback

### `components/proxy/securityheaders.go`
- Remove `TODO(issue #197)` comment (lines 26–27)

### `main.go`
- Add `UpdateCSP: p.SetSiteCSP` to `sites.Options`

## Verify
```bash
go test ./components/proxy/... -run TestCSP -v
go test ./components/proxy/... -run TestSecurityHeaders -v
```
- Registered site with `CSPPolicy` gets that value; not `permissiveCSP`
- Site without `CSPPolicy` still gets `permissiveCSP` fallback
- `SetSiteCSP("")` clears the override; fallback resumes
