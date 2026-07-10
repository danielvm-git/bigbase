# e70s04 — static-sidecar App Profile

## Summary
Add static-sidecar app profile for deploying sidecar-only services (no main application). The sidecar profile skips framework detection and uses a minimal build/start configuration from bigbase.toml.

## Changes
- Add "static-sidecar" to valid app types in manifest/config
- Skip framework auto-detection for sidecar profile
- Minimal build step (custom command only)
- Start command directly from bigbase.toml without framework wrapper

## Verify
```
go test ./components/deploy/... -run "StaticSidecar"
```
