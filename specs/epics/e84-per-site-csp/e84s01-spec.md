# e84s01 — Site model + API layer

## Summary
Add `csp_policy` field to `DeployDefaults` so site owners can set a per-site CSP via
`PUT /api/sites/{id}/deploy-defaults`. Wire an `UpdateCSP` callback seam so the proxy
cache is refreshed immediately when deploy_defaults changes.

## Changes

### `components/sites/sites.go`
- Add `CSPPolicy string \`json:"csp_policy,omitempty"\`` to `DeployDefaults` struct
- Add `updateCSP func(siteID, cspPolicy string)` field to `Sites` struct
- Add `UpdateCSP func(siteID, cspPolicy string)` to `Options` struct
- In `New()`: `updateCSP: opts.UpdateCSP`
- In `setSiteDeployDefaults`: after `ExecContext` success, call `s.updateCSP(id, dd.CSPPolicy)` if non-nil

No DB migration needed — `deploy_defaults` is already a JSON blob column.

## Verify
```bash
go test ./components/sites/... -run TestDeployDefaults -v
```
- Set `deploy_defaults` with `csp_policy`; assert it round-trips via GET
- Assert `UpdateCSP` callback fires with the new value
- Assert empty `csp_policy` clears the callback value
