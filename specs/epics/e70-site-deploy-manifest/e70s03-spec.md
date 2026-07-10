# e70s03 — Parameterized CI Templates

## Summary
Extend get_ci_template MCP tool to include deploy defaults from site record and bigbase.toml. CI templates auto-populate build commands, env vars, and framework-specific configurations.

## Changes
- Read deploy_defaults from site record in get_ci_template
- Merge with bigbase.toml defaults if present
- Generate correct CI template (GitHub Actions, GitLab CI) with defaults
- Add test coverage for each template variant

## Verify
```
go test ./components/mcp/... -run "CITemplate"
```
