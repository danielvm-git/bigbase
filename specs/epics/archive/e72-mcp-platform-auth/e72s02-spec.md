# e72s02: Scope-Gated Provisioning — mcp:provision scope for write tools

**type:** feat · **context:** domain · **BCPs:** 2 · **Status:** planned

## Story
**As a** platform operator, **I want** write/provisioning MCP tools to require an `mcp:provision` scope on the presented key, **so that** read-capable keys cannot mutate infrastructure.

## Context
Builds on e72s01. Org API keys carry scopes. Write tools (`create_repo`, `create_site`, `provision_ci_credentials`, and the e72s04 credential issuer) must assert `mcp:provision`; read tools do not.

## Steps
1. Add a `requireScope("mcp:provision")` guard invoked at the start of each write-tool handler. → verify: `go test ./components/mcp/ -run TestRequireScope -v`
2. Tag existing provisioning tools as write-scoped; return `403` with a generic message when the scope is absent. → verify: `go test ./components/mcp/ -run TestProvisionScopeGate -v`
3. Ensure read tools remain reachable without the scope. → verify: `go test ./components/mcp/ -run TestReadToolsNoScope -v`

## Acceptance
- Given a key without `mcp:provision`, when `create_site` is called, then the server responds `403`.
- Given a key with `mcp:provision`, when `create_site` is called, then it proceeds.

## Out of scope
- Issuing scoped keys (e72s04).

## Risks
- Scope naming must match the key store's scope vocabulary (align with e30/e67).
