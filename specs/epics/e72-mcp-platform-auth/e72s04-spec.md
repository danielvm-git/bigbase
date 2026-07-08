# e72s04: provision_mcp_credentials — issue scoped MCP access keys for agents

**type:** feat · **context:** domain · **BCPs:** 1 · **Status:** planned

## Story
**As a** platform operator, **I want** an authenticated tool that mints scoped MCP keys, **so that** I can hand an agent a least-privilege `bb_` key bound to `mcp:provision` without sharing the org root key.

## Context
Depends on e72s01/s02. Adds a write tool `provision_mcp_credentials` that creates a new org-scoped key with a caller-specified scope set (default `mcp:provision`) and returns the raw token once.

## Steps
1. Add `provision_mcp_credentials` tool that requires `mcp:provision` and creates a scoped key via the org key store. → verify: `go test ./components/mcp/ -run TestProvisionMCPCredentials -v`
2. Return the raw token exactly once; persist only the hash (reuse site-key hashing pattern). → verify: `go test ./components/mcp/ -run TestProvisionMCPCredentialsHashOnly -v`

## Acceptance
- Given a caller with `mcp:provision`, when the tool runs, then a new scoped key is created and the raw token returned once.
- Given a caller without `mcp:provision`, when the tool runs, then it returns `403`.

## Out of scope
- Key rotation/revocation UI (reuse existing key lifecycle from e69).

## Risks
- Raw token must never be logged or re-returned (generic-error + hash-only storage per CONVENTIONS.md).
