### Story e71s04: MCP set_site_auth_policy tool for agent configuration — Implementation Steps

**type:** feat
**risk:** P2
**context:** domain
**Context**: To make it easy for AI coding agents to configure route security policies, we must expose a Model Context Protocol (MCP) tool. This story adds the `set_site_auth_policy` tool to the MCP component, enabling agents to declare the default route policy, protected paths, public paths, and accepted credential types for any site.

## Steps

1. In `components/mcp/mcp.go` `NewMCPServer()`, register the `set_site_auth_policy` tool at the `tierWrite` access tier:
   - Name: `set_site_auth_policy`
   - Description: "Set the authentication and routing policy for a deployed site."
   - Input Schema:
     - `site_id` (string, required): The ID of the site to secure.
     - `policy` (object, required): The policy definition containing:
       - `default` (string, optional): "public" or "protected".
       - `protected_paths` (array of strings, optional): Paths requiring auth.
       - `public_paths` (array of strings, optional): Paths bypassing auth.
       - `accept` (array of strings, optional): Accepted authentication types (e.g. `["jwt", "site_key"]`).
   Tag the Go MCP SDK package as `[OK]`. → verify: `go test -v -run TestNewMCPServer ./components/mcp/...`
2. Add `UpdateAuthPolicy func(siteID string, policyJSON string)` callback to `mcp.Options` and `mcp.Component`. Implement the tool handler:
   - Check if `c.db` is initialized. If nil, return an error.
   - Deserialize the policy arguments, format/validate fields, and marshal to a JSON string.
   - Run `ExecContext` on `c.db` to update the `auth_policy` column for the site in the database.
   - Invoke the `UpdateAuthPolicy` callback to notify the proxy to update its in-memory policy cache immediately.
   - Return a success status result indicating the policy was updated.
   → verify: `go test -v -run TestMCPSetSiteAuthPolicy ./components/mcp/...`
3. Write unit tests in `components/mcp/mcp_test.go` or a new test file asserting that calling the tool with valid arguments successfully persists the policy to the database and correctly triggers the callback. → verify: `go test -v -run TestMCPSetSiteAuthPolicyAPI ./components/mcp/...`

## Verification Script (Step-by-Step)

1. Run `go test ./components/mcp/...` to ensure all MCP server tools and registrations work.
2. Run `go build -o bigbase .` to verify compilation.

## Out of scope

- UI components for auth policy management (future epic).

## Risks

- SQL syntax errors when updating the `sites` table. Mitigated by using standard parameter binding (`UPDATE sites SET auth_policy = ? WHERE id = ?`).
- Stale state on cluster restarts. Mitigated by persisting the policy to the shared DB.
