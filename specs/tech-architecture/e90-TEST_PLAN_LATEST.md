# Test Design: e90 Dev-Environment Hardening

**type:** test-plan
**context:** domain
**epic:** e90
**risk profile:** LOW/MEDIUM; Dev-Environment & CI Gates

## 1. Risk Matrix & Scenarios

| Scenario ID | Behavior Description | Risk | Test Level | Target Area |
|---|---|---:|---|---|
| SC-e90s01-P0-01 | Missing required environment variables fail deployment preflight before binary swap. | P0 | CI / Bash | `.github/workflows/deploy.yml` |
| SC-e90s02-P1-01 | Git conflict markers in source or config fail CI verify job. | P1 | CI / Shell | `scripts/check-conflict-markers.sh` |
| SC-e90s02-P1-02 | Single `.gosec.json` ruleset governs all security scans without shadow exclusions. | P1 | Mechanical | `.gosec.json`, `package.json` |
| SC-e90s03-P1-01 | AGENTS.md documents ctxo accurately as graph oracle without unviable mandatory claims. | P1 | Docs | `AGENTS.md` |
| SC-e90s04-P1-01 | Go race detector flags concurrency violations in deploy, functions, and realtime. | P1 | CI / Go | `test-build-release.yml` |
| SC-e90s04-P1-02 | Frontend lint and TypeScript compilation violations block CI merge. | P1 | CI / Node | `test-build-release.yml` |
| SC-e90s05-P1-01 | Generated client MCP configs match `.mcp.json` without path drift or missing tools. | P1 | Shell | `scripts/gen-mcp-configs.sh` |
| SC-e90s06-P1-01 | AGENTS.md serves as single source of engineering truth; CLAUDE.md/GEMINI.md are thin symlinks. | P1 | Docs / Audit | `AGENTS.md`, `CLAUDE.md` |
| SC-e90s07-P2-01 | Go and Node version claims across all docs match toolchain declarations. | P2 | Shell | `scripts/check-version-drift.sh` |
| SC-e90s08-P2-01 | Dependabot group configurations group dependency updates predictably. | P2 | Config | `.github/dependabot.yml` |
| SC-e90s09-P2-01 | Async goroutine tests use polling helpers instead of bare sleeps. | P2 | Unit / Race | `components/...` |
