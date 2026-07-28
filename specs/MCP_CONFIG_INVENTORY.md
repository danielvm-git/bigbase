# MCP Configuration Inventory

**Generated:** 2026-07-27  
**Root:** `~/Developer/bigbase`  
**Status:** ✅ All MCPs configured correctly

---

## Summary

| Agent | Config File | MCPs | Status |
|-------|-------------|------|--------|
| Claude Code | `.mcp.json` | 8 | ✅ Correct |
| agy CLI | `~/.gemini/config/mcp_config.json` | 7 | ✅ Correct |
| Antigravity IDE | `~/.gemini/config/mcp_config.json` | 7 | ✅ Correct |
| Cursor | `.cursor/mcp.json` | 7 | ✅ Correct |
| VS Code/Cline/Roo | `.vscode/mcp.json` | 7 | ✅ Correct |
| Pi | `.pi/mcp.json` | 7 | ✅ Correct |
| MiMo/OpenCode | `opencode.jsonc` | 7 | ✅ Correct |

---

## MCP Servers (7 total + 1 Claude-only)

| # | Name | Type | URL/Command | Agents |
|---|------|------|-------------|--------|
| 1 | context7 | HTTP | `https://mcp.context7.com/mcp` | All |
| 2 | bigbase | HTTP | `https://mcp.bigbase.click/mcp` | All |
| 3 | seal | HTTP | `https://api-stage.35.253.75.95.nip.io/mcp` | All |
| 4 | sqz | Local | `sqz-mcp` | All |
| 5 | ctxo | Stdio | `npx -y @ctxo/cli` | All |
| 6 | filesystem | Stdio | `npx -y @modelcontextprotocol/server-filesystem` | All |
| 7 | sequential-thinking | Stdio | `npx -y @modelcontextprotocol/server-sequential-thinking` | All |
| 8 | claude_design | HTTP | `https://api.anthropic.com/v1/design/mcp` | Claude Code only |

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
