# e84s03 — Manifest security.csp field

## Summary
Allow site owners to version-control their CSP in `bigbase.toml` / `bigbase.yaml` under
a `[security]` section. At deploy time, if the manifest declares a CSP, it is written to
`deploy_defaults.csp_policy` in the DB so the proxy picks it up at `RegisterDeploymentHost`.

Priority order: manifest `[security] csp` > site `deploy_defaults.csp_policy` > permissiveCSP.

## Changes

### `components/deploy/manifest.go`
- Add `type ManifestSecurity struct { CSP string \`yaml:"csp" toml:"csp"\` }`
- Add `Security ManifestSecurity \`yaml:"security" toml:"security"\`` to `Manifest` struct
- Add `CSPPolicy string \`json:"csp_policy,omitempty"\`` to `SiteDefaults` struct
- In `MergeManifests`: if `siteDefaults.CSPPolicy != ""` and `base.Security.CSP == ""`,
  set `base.Security.CSP = siteDefaults.CSPPolicy`

### `components/deploy/engine.go`
- After loading the manifest (line ~204), if `manifest.Security.CSP != ""`:
  read `deploy_defaults` from `sites`, unmarshal into `SiteDefaults`,
  set `sd.CSPPolicy = manifest.Security.CSP`, marshal and write back

## Verify
```bash
go test ./components/deploy/... -run TestMergeManifests -v
go test ./components/deploy/... -run TestManifest -v
```
- `[security] csp` in bigbase.toml parses correctly
- `MergeManifests`: manifest CSP > site default CSP
- `MergeManifests`: site default CSP fills in when manifest is silent
- Deploy engine writes manifest CSP to `deploy_defaults.csp_policy`
