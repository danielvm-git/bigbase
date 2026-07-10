# e70s02 — bigbase.toml Manifest Parsing

## Summary
Parse bigbase.toml from repository root during deploy. Merge configuration with site defaults and request body overrides in the three-layer hierarchy.

## Changes
- Add TOML parsing (BurntSushi/toml) to LoadManifest or parallel path
- bigbase.toml in repo root detected during build step
- Three-layer merge: toml → site defaults → request body
- Validate TOML schema with clear error messages

## Verify
```
go test ./components/deploy/... -run "TOML\|bigbase.toml"
```
