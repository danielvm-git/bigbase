# MCP Multi-Agent Configuration Plan

## Goal
Have all 7 MCP servers working across all 8 agents with the correct config format for each.

---

## MCP Servers (7 total)

| # | Name | Type | URL/Command |
|---|------|------|-------------|
| 1 | context7 | HTTP | `https://mcp.context7.com/mcp` |
| 2 | bigbase | HTTP | `https://mcp.bigbase.click/mcp` |
| 3 | seal | HTTP | `https://api-stage.35.253.75.95.nip.io/mcp` |
| 4 | sqz | Local | `sqz-mcp` |
| 5 | ctxo | Stdio | `npx -y @ctxo/cli` |
| 6 | filesystem | Stdio | `npx -y @modelcontextprotocol/server-filesystem ~/Developer/bigbase /tmp` |
| 7 | sequential-thinking | Stdio | `npx -y @modelcontextprotocol/server-sequential-thinking` |

**Note:** `claude_design` is Claude Code-specific and NOT included in other agents.

---

## Agent Config Formats (from source code analysis)

### 1. Claude Code (`.mcp.json`)
```json
{
  "mcpServers": {
    "context7": {
      "type": "http",
      "url": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "command": "sqz-mcp", "args": [] },
    "ctxo": { "command": "npx", "args": ["-y", "@ctxo/cli"] }
  }
}
```
**File:** `.mcp.json` (project scope) or `~/.claude.json` (user scope)

### 2. agy CLI / Antigravity IDE (`mcp_config.json`)
```json
{
  "mcpServers": {
    "context7": {
      "serverUrl": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "command": "sqz-mcp", "args": [] },
    "ctxo": { "command": "npx", "args": ["-y", "@ctxo/cli"] }
  }
}
```
**File:** `~/.gemini/config/mcp_config.json` (global) or `~/.gemini/antigravity-cli/mcp_config.json`

### 3. MiMo Code / OpenCode (`opencode.jsonc`)
```json
{
  "mcp": {
    "context7": {
      "type": "remote",
      "url": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "type": "local", "command": ["sqz-mcp"] },
    "ctxo": { "type": "local", "command": ["npx", "-y", "@ctxo/cli"] }
  }
}
```
**File:** `opencode.jsonc` (project root) or `~/.config/opencode/config.json` (global)

### 4. Cursor (`.cursor/mcp.json`)
```json
{
  "mcpServers": {
    "context7": {
      "serverUrl": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "command": "sqz-mcp", "args": [] },
    "ctxo": { "command": "npx", "args": ["-y", "@ctxo/cli"] }
  }
}
```
**File:** `.cursor/mcp.json`

### 5. VS Code / Cline / Roo Code (`.vscode/mcp.json`)
```json
{
  "servers": {
    "context7": {
      "serverUrl": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "command": "sqz-mcp", "args": [] },
    "ctxo": { "command": "npx", "args": ["-y", "@ctxo/cli"] }
  }
}
```
**File:** `.vscode/mcp.json`

### 6. Pi (`.pi/mcp.json`)
```json
{
  "mcpServers": {
    "context7": {
      "url": "https://mcp.context7.com/mcp",
      "headers": { "CONTEXT7_API_KEY": "..." }
    },
    "sqz": { "command": "sqz-mcp", "args": [] },
    "ctxo": { "command": "npx", "args": ["-y", "@ctxo/cli"] }
  }
}
```
**File:** `.pi/mcp.json`

---

## Implementation Plan

### Phase 1: Test with zero MCPs
1. Remove all MCP configs from all agent files
2. Verify each agent starts without MCP errors
3. Confirm baseline works

### Phase 2: Add context7 (HTTP server)
1. Add to each agent with correct format
2. Test each agent can connect to context7
3. Verify tools are available

### Phase 3: Add bigbase (HTTP server)
1. Add to each agent with correct format
2. Test each agent can connect to bigbase
3. Verify tools are available

### Phase 4: Add seal (HTTP server)
1. Add to each agent with correct format
2. Test each agent can connect to seal
3. Verify tools are available

### Phase 5: Add sqz (local binary)
1. Add to each agent with correct format
2. Test each agent can run sqz-mcp
3. Verify tools are available

### Phase 6: Add ctxo (npm package)
1. Add to each agent with correct format
2. Test each agent can run ctxo
3. Verify tools are available

### Phase 7: Add filesystem (npm package)
1. Add to each agent with correct format
2. Test each agent can run filesystem server
3. Verify tools are available

### Phase 8: Add sequential-thinking (npm package)
1. Add to each agent with correct format
2. Test each agent can run sequential-thinking
3. Verify tools are available

### Phase 9: Add claude_design (Claude Code only)
1. Add to Claude Code `.mcp.json` only
2. Test Claude Code can connect
3. Verify tools are available

---

## File Locations Summary

| Agent | Config File | Scope |
|-------|-------------|-------|
| Claude Code | `.mcp.json` | Project |
| agy CLI | `~/.gemini/config/mcp_config.json` | Global |
| Antigravity IDE | `~/.gemini/antigravity-cli/mcp_config.json` | Global |
| MiMo Code | `opencode.jsonc` | Project |
| OpenCode | `opencode.jsonc` | Project |
| Cursor | `.cursor/mcp.json` | Project |
| VS Code/Cline/Roo | `.vscode/mcp.json` | Project |
| Pi | `.pi/mcp.json` | Project |

---

## Key Differences Between Agents

| Feature | Claude Code | agy | MiMo/OpenCode | Cursor | VS Code | Pi |
|---------|-------------|-----|---------------|--------|---------|-----|
| HTTP key | `url` + `type` | `serverUrl` | `url` + `type` | `serverUrl` | `serverUrl` | `url` |
| Stdio key | `command` + `args` | `command` + `args` | `command` (array) | `command` + `args` | `command` + `args` | `command` + `args` |
| Top key | `mcpServers` | `mcpServers` | `mcp` | `mcpServers` | `servers` | `mcpServers` |
| Headers | `headers` | `headers` | `headers` | `headers` | `headers` | `headers` |

---

## Critical Learning

**agy CLI bug:** The `context7` MCP server with `npx -y @upstash/context7-mcp` format causes agy to hang during startup. Must use `serverUrl` format instead.

**Root cause of agy hang:** The `.agents/mcp_config.json` had `context7` configured with `npx` format, which agy couldn't connect to. Fixed by using `serverUrl` format.
