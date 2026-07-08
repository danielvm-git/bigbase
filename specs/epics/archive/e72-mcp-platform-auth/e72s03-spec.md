# e72s03: Read-Only Public Tier — list_services, get_ci_template without auth

**type:** feat · **context:** domain · **BCPs:** 1 · **Status:** planned

## Story
**As an** AI agent evaluating BigBase, **I want** discovery/read tools to work without credentials, **so that** onboarding stays frictionless while write tools stay protected.

## Context
The Bearer middleware (e72s01) must allow an explicit allow-list of read-only tools (e.g. `list_services`, `get_ci_template`, education/knowledge tools) to run unauthenticated.

## Steps
1. Define a `publicReadTools` allow-list and skip the Bearer requirement for those tool names. → verify: `go test ./components/mcp/ -run TestPublicReadTier -v`
2. Ensure any tool not on the allow-list falls through to the auth requirement (deny-by-default). → verify: `go test ./components/mcp/ -run TestDenyByDefault -v`

## Acceptance
- Given no auth header, when `get_ci_template` is called, then it returns successfully.
- Given no auth header, when `create_site` is called, then it returns `401`.

## Out of scope
- Rate limiting the public tier (tracked separately).

## Risks
- Allow-list must be reviewed so no mutating tool is accidentally public (deny-by-default mitigates).
