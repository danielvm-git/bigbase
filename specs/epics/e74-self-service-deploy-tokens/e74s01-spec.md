# e74s01 — REST Endpoints for Site Deploy Keys

## Summary

Add HTTP REST endpoints for site-scoped deploy key (`bb_dep_*`) CRUD. The underlying Go functions (`CreateSiteKey`, `ListSiteKeys`, `RevokeSiteKey`) already exist — this story wraps them with HTTP handlers, input validation, rate limiting, and audit logging.

## Changes

### New endpoints (on the auth or sites component)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `POST` | `/api/sites/{id}/deploy-keys` | `handleCreateSiteKey` | Generate `bb_dep_` key, return raw token once |
| `GET` | `/api/sites/{id}/deploy-keys` | `handleListSiteKeys` | List key metadata (no raw tokens) |
| `DELETE` | `/api/sites/{id}/deploy-keys/{keyID}` | `handleRevokeSiteKey` | Soft-delete (set `revoked=1`) |

### Authorization

- Route behind `authComp.Middleware` (JWT or `bb_` key required) — same protection level as the existing sites API
- No per-site ownership check (sites table has no `org_id` column yet — e57s03 adds it)
- `bb_dep_` tokens themselves cannot call these endpoints (MCP rejects them) — only JWT or `bb_` org keys

### Input validation

- `site_id` — validated as non-empty string; checked against DB via existing `CreateSiteKey`
- `name` — optional, max 100 chars, alphanumeric + hyphens
- `scopes` — default to `["deploy"]`, reject unknown scopes

### Rate limiting

- `POST /api/sites/{id}/deploy-keys` — max 10 creations per site per hour per user
- Use existing rate-limit middleware (`rl.Middleware`)

### Security hardening

- Raw token **never** written to server logs — log only `key_id` and `site_id`
- `GET` list endpoint returns metadata only: `key_id`, `name`, `prefix` (first 10 chars of token e.g. `bb_dep_a1b2`), `last_used_at`, `created_at`
- Audit events: `auth.site_key_created`, `auth.site_key_revoked`

## Verify

```
go test ./components/auth/... -run "SiteKey"
go test ./components/auth/... -run "SiteKeyHandler"
go test ./components/auth/... -run "SiteKeyRateLimit"
```
