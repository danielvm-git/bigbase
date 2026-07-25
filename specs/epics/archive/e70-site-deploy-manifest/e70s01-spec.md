# e70s01 — Persist Deploy Defaults on Site Record

## Summary
Add deploy_defaults column to the sites table and CRUD endpoints. Site-level defaults (framework, build command, env vars) serve as the middle layer in the three-layer config merge: bigbase.toml → site defaults → request body.

## Changes
- Add deploy_defaults JSONB column to sites table (SQLite: TEXT, Postgres: JSONB)
- Update site CRUD endpoints (POST/PUT /api/sites) to accept deploy_defaults
- Expose deploy_defaults in site GET responses
- Validate deploy_defaults schema on write

## Verify
```
go test ./components/sites/... -run "DeployDefaults"
```
