# MCP Configuration Inventory

**Model:** single canonical source → generated adapters (e90s05).
**Canonical:** [`.mcp.json`](../.mcp.json) — edit this, nothing else.
**Generator:** [`scripts/gen-mcp-configs.sh`](../scripts/gen-mcp-configs.sh)
(CI enforces sync via `--check`).

---

## Files

| File | Role | Committed | Format |
|------|------|-----------|--------|
| `.mcp.json` | **canonical** (Claude Code project config) | yes | `mcpServers`, `http` / `command`+`args` |
| `.claude/mcp.json` | generated mirror of `.mcp.json` | yes | identical to canonical |
| `opencode.jsonc` | generated adapter | yes | `mcp`, `remote` / `local` (Claude-only servers omitted) |
| `opencode.json` | hand-maintained opencode config; its `mcp` block mirrors the canonical set | yes | opencode |
| `.cursor/mcp.json`, `.vscode/mcp.json` | per-machine, regenerated locally if present | **no** (gitignored) | client-specific |

Regenerate after editing `.mcp.json`:

```bash
scripts/gen-mcp-configs.sh          # write adapters
scripts/gen-mcp-configs.sh --check  # CI: fail on drift
```

---

## Servers (6 universal + 1 Claude-only)

| # | Name | Transport | URL / Command | Scope |
|---|------|-----------|---------------|-------|
| 1 | context7 | HTTP | `https://mcp.context7.com/mcp` | all |
| 2 | bigbase | HTTP | `https://mcp.bigbase.click/mcp` | all |
| 3 | sqz | local | `sqz-mcp` | all |
| 4 | ctxo | local | `npx -y @ctxo/cli@0.11.4` | all |
| 5 | filesystem | local | `npx -y @modelcontextprotocol/server-filesystem@2026.7.10 ${BIGBASE_ROOT:-.} /tmp` | all |
| 6 | sequential-thinking | local | `npx -y @modelcontextprotocol/server-sequential-thinking@2026.7.4` | all |
| 7 | claude_design | HTTP | `https://api.anthropic.com/v1/design/mcp` | **Claude only** |

Notes:

- `claude_design` is not agent-universal — the generator omits it from
  `opencode.jsonc` (and any non-Claude adapter).
- npx-spawned packages are **version-pinned** in the canonical file (no `@latest`).
- The `filesystem` root is portable: `${BIGBASE_ROOT}` (set locally by
  `scripts/setup.sh` in `.envrc`), falling back to the launch directory. No
  machine-specific absolute paths live in committed config.
- New Relic MCP was removed (project no longer uses New Relic).
