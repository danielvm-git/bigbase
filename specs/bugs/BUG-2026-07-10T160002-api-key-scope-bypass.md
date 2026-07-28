---
bug_id: BUG-2026-07-10T160002
status: fixed
severity: high
scope: auth
title: "API key scopes not enforced in auth middleware — full org access (CWE-287)"
---

## Problem

`ResolveOrgKey` resolves a `bb_` API key to `org_id + scopes`, and the middleware
stores scopes in the request context via `ctxOrgKeyScopes`. However,
`OrgKeyScopesFromContext()` had **zero callers** — scopes were resolved but never
enforced by any handler. A scoped API key with `scopes: ["sites:read"]` could
perform unrestricted CRUD on org resources (create/delete sites, manage API keys,
etc.).

## Root Cause

The enforcement half of the scope fix was never implemented. The middleware
stashed scopes in context, but no middleware or handler ever checked them.

## Fix

Added `RequireScopes(required ...string)` middleware in
`components/auth/middleware.go` that:
- Extracts scopes from context via `OrgKeyScopesFromContext`
- Returns 403 "insufficient scopes" when no required scope matches
- Passes through requests with no scopes (unscoped keys = full access, backward compat)
- Passes through JWT-authenticated requests (no scopes in context)

Wired into `ProtectedHandler` on all write routes:
- `POST /api/orgs` → `RequireScopes("orgs:write")`
- `PATCH /api/orgs/{id}` → `RequireScopes("orgs:write")`
- `DELETE /api/orgs/{id}` → `RequireScopes("orgs:write")`
- `POST /api/orgs/{id}/invites` → `RequireScopes("orgs:write")`
- `POST /api/orgs/{id}/api-keys` → `RequireScopes("orgs:write")`
- `DELETE /api/orgs/{id}/api-keys/{keyID}` → `RequireScopes("orgs:write")`
- `POST /api/sites/{id}/deploy-keys` → `RequireScopes("sites:write", "deploy")`
- `DELETE /api/sites/{id}/deploy-keys/{keyID}` → `RequireScopes("sites:write")`

Read-only routes remain unscoped for backward compatibility.

## Regression Guards

- `TestRequireScopes_MatchingScopeAllowed` — scoped key with matching scope → 200
- `TestRequireScopes_MissingScopeDenied` — scoped key without matching scope → 403
- `TestRequireScopes_UnscopedKeyAllowed` — unscoped key → 200 (backward compat)
- `TestRequireScopes_JWTAuthBypassesScopeCheck` — JWT auth bypasses scope check

## Scope Model

| Scope | Grants access to |
|-------|-----------------|
| `orgs:write` | Create/update/delete orgs, invites, API keys |
| `sites:write` | Create/revoke deploy keys |
| `deploy` | Create deploy keys (CI/CD use case) |

Unscoped keys (empty scopes) get full access for backward compatibility.
