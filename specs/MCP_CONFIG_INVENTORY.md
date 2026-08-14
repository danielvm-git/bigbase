# MCP Configuration Inventory

**Generated:** 2026-08-13
**Root:** repository-relative paths only
**Status:** credentials use `${CONTEXT7_API_KEY}`; local MCP packages are pinned

---

## Summary

| Agent | Config File | MCPs | Status |
|-------|-------------|------|--------|
| Claude Code | `.mcp.json` / `.claude/mcp.json` | 7 | ✅ Current |
| agy / Antigravity | `.agents/mcp_config.json` | 6 | ✅ Current |
| Cursor | `.cursor/mcp.json` | 6 | ✅ Current |
| VS Code/Cline/Roo | `.vscode/mcp.json` | 6 | ✅ Current |
| Pi | `.pi/mcp.json` | 6 | ✅ Current |
| OpenCode | `opencode.json` / `opencode.jsonc` | 6 | ✅ Current |

---

## MCP Servers (6 shared + 1 Claude-only)

| # | Name | Type | URL/Command | Agents |
|---|------|-------------|-------------|--------|
| 1 | context7 | HTTP/local | `https://mcp.context7.com/mcp` | All |
| 2 | bigbase | HTTP | `https://mcp.bigbase.click/mcp` | All |
| 3 | sqz | Local | `sqz-mcp` | All |
| 4 | ctxo | Stdio | `npx -y @ctxo/cli@0.11.4` | All |
| 5 | filesystem | Stdio | `npx -y @modelcontextprotocol/server-filesystem@2026.7.10 . /tmp` | All |
| 6 | sequential-thinking | Stdio | `npx -y @modelcontextprotocol/server-sequential-thinking@2026.7.4` | All |
| 7 | claude_design | HTTP | `https://api.anthropic.com/v1/design/mcp` | Claude Code only |

---

## Config Format Reference

### Claude Code (`.mcp.json`)
```json
{
  "mcpServers": {
    "context7": {
      "type": "http",
      "url": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "command": "sqz-mcp", "args": [] }
  }
}
```

### agy CLI / Antigravity IDE (`mcp_config.json`)
```json
{
  "mcpServers": {
    "context7": {
      "serverUrl": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "command": "sqz-mcp", "args": [] }
  }
}
```

### MiMo/OpenCode (`opencode.jsonc`)
```json
{
  "mcp": {
    "context7": {
      "type": "remote",
      "url": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "type": "local", "command": ["sqz-mcp"] }
  }
}
```

### Cursor (`.cursor/mcp.json`)
```json
{
  "mcpServers": {
    "context7": {
      "serverUrl": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "command": "sqz-mcp", "args": [] }
  }
}
```

### VS Code/Cline/Roo (`.vscode/mcp.json`)
```json
{
  "servers": {
    "context7": {
      "serverUrl": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "command": "sqz-mcp", "args": [] }
  }
}
```

### Pi (`.pi/mcp.json`)
```json
{
  "mcpServers": {
    "context7": {
      "url": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "command": "sqz-mcp", "args": [] }
  }
}
```

---

## Key Differences Between Agents

| Feature | Claude Code | agy | MiMo/OpenCode | Cursor | VS Code | Pi |
|---------|-------------|-----|---------------|--------|---------|-----|
| HTTP key | `url` + `type` | `serverUrl` | `url` + `type` | `serverUrl` | `serverUrl` | `url` |
| Stdio key | `command` + `args` | `command` + `args` | `command` (array) | `command` + `args` | `command` + `args` | `command` + `args` |
| Top key | `mcpServers` | `mcpServers` | `mcp` | `mcpServers` | `servers` | `mcpServers` |
| Headers | `headers` | `headers` | `headers` | `headers` | `headers` | `headers` |

---

## Root Cause of agy Hang

The `.agents/mcp_config.json` had `context7` configured with `npx -y @upstash/context7-mcp` format, which agy couldn't connect to. Fixed by using `serverUrl` format.

---

## Notes

- `claude_design` is Claude Code-specific and NOT included in other agents
- agy CLI uses `serverUrl` for HTTP servers, not `url` or `type`
- MiMo/OpenCode uses `type: "remote"` and `type: "local"` for HTTP and stdio servers
- VS Code uses `servers` key instead of `mcpServers`
- Pi uses `url` key for HTTP servers
