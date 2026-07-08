# e72s01: MCP HTTP Bearer Middleware — validate bb_ org API keys

**type:** feat · **context:** domain · **BCPs:** 2 · **Status:** planned

## Story
**As a** platform operator, **I want** the MCP HTTP server to require a valid `bb_` organization API key on write tools, **so that** `mcp.bigbase.click` is no longer an unauthenticated remote-control surface.

## Context
`components/mcp/mcp.go` serves the MCP protocol over HTTP with no authentication. Provisioning tools (`create_repo`, `create_site`, …) are publicly reachable. This story adds a Bearer-token middleware that validates the `Authorization: Bearer bb_…` header against the existing org API-key store (reused from e30/e67) and attaches the resolved org to the request context.

## Steps
1. Add `bearerAuthMiddleware` to `components/mcp/mcp.go` that extracts `Authorization: Bearer <token>`, validates `bb_`-prefixed keys via the org key store, and rejects with `401` on missing/invalid tokens. → verify: `go test ./components/mcp/ -run TestBearerAuthMiddleware -v`
2. Attach resolved `org_id` to request context using the kernel scoping helpers (e57s01). → verify: `go test ./components/mcp/ -run TestBearerAuthContext -v`
3. Wire the middleware into the MCP HTTP handler chain (before tool dispatch). → verify: `go build ./... && go test ./components/mcp/ -run TestMCP -v`

## Acceptance
- Given no `Authorization` header, when a write tool is called, then the server responds `401`.
- Given a valid `bb_` key, when a write tool is called, then the request proceeds with `org_id` in context.

## Out of scope
- Scope enforcement per-tool (e72s02); public read tier (e72s03).

## Risks
- Must not break existing local stdio transport (only HTTP transport is gated).
