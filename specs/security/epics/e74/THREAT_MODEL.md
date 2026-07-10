# Threat Model: e74 — Self-Service Deploy Tokens

## Epic Scope

Two stories:
- **e74s01**: REST endpoints for site-scoped deploy key CRUD (`bb_dep_*`)
- **e74s02**: Deploy Keys tab on Site Detail page (React UI)

## Surface Area

### Story 1 — REST Endpoints

| Endpoint | Method | Description | New/Existing |
|----------|--------|-------------|-------------|
| `/api/sites/{id}/deploy-keys` | POST | Generate `bb_dep_` key, return raw token once | New |
| `/api/sites/{id}/deploy-keys` | GET | List key metadata (no raw tokens) | New |
| `/api/sites/{id}/deploy-keys/{keyID}` | DELETE | Soft-delete (set revoked=1) | New |

### Story 2 — UI Tab

- New `SiteDeployKeysTab.tsx` component with list, generate modal, revoke confirmation
- Tab registration in `SiteDetailPage.tsx` (after "Domains", before "Cache")
- Raw token displayed once in modal with copy button, cleared on modal close/unmount

## Trust Boundaries

```
[User Browser] → [auth.Middleware (JWT/bb_ key)] → [ProtectedHandler] → [Site Key Handlers] → [DB]
```

- `bb_dep_` deploy tokens CANNOT call these endpoints (MCP rejects them)
- Only JWT-authenticated users or `bb_` org admin keys can access

## Vulnerability Assessment

| # | Category | Finding | Confidence | Severity | Status |
|---|----------|---------|------------|----------|--------|
| 1 | Secrets Exposure | Raw `bb_dep_` token returned once on POST — must never be logged or cached server-side | 10 | HIGH | Mitigated by design (spec requires log redaction) |
| 2 | Secrets Exposure | Raw token exposed in browser DOM on generate — client-side XSS could steal it | 9 | HIGH | Mitigated by ephemeral state: cleared on modal close/unmount; React auto-escapes |
| 3 | IDOR | GET/DELETE on `/api/sites/{id}/deploy-keys` — no per-site ownership check (sites table has no `org_id` column until e57) | 8 | MEDIUM | Accepted: same protection level as existing sites API (JWT/bb_ key gate only) |
| 4 | Auth Bypass | Routes must be registered in `ProtectedHandler()` (wrapped by `auth.Middleware`) — accidental registration in `Handler()` would expose them without auth | 9 | HIGH | Mitigated by design: code review gate ensures correct registration |
| 5 | Rate Limiting | POST endpoint can be abused to generate unlimited deploy keys — each creation hits DB | 8 | MEDIUM | Mitigated by spec: 10 POSTs/site/hour/user via rate-limit middleware |
| 6 | Audit Integrity | Audit events are fire-and-forget goroutines — audit loss possible on shutdown | 7 | LOW | Accepted: consistent with existing audit pattern across auth component |
| 7 | Token Leak (Logs) | Raw token could leak via server error logs if handler prints it in error messages | 9 | HIGH | Mitigated by spec: log only `key_id` and `site_id`; grep gate in tasks |
| 8 | Token Leak (List) | GET list endpoint must NOT include raw token column — only metadata | 9 | HIGH | Mitigated by spec: `SiteKeyEntry` has no raw token field (verified by test) |
| 9 | Token Leak (Browser) | Clipboard API access on generate modal — other browser extensions/tabs can read clipboard | 6 | LOW | Accepted: standard browser clipboard behavior; user accepts when clicking "Copy" |
| 10 | CSRF | POST/DELETE via non-browser clients (curl, scripts) — no CSRF token | 5 | LOW | N/A: API does not use cookie-based auth; JWT/API-key in Authorization header |

## Risk Assessment

| Dimension | Rating | Rationale |
|-----------|--------|-----------|
| Overall Risk | **MEDIUM** | No vulnerability bypasses auth boundary. Token exposure is the primary concern |
| Likelihood of Exploit | LOW | Attacker must have JWT or `bb_` key to access endpoints |
| Impact of Exploit | MEDIUM | Leaked deploy token allows unauthorized deploys to that site |
| Attack Complexity | MEDIUM | Requires authenticated session + targeted site |

## Mitigation Guidance

### Critical (implement in e74s01)

1. **Log redaction**: Never log raw token in any handler. Use structured logging with only `key_id` and `site_id`. Verify with grep gate as specified in tasks.
2. **Route registration**: Register all site key endpoints in `ProtectedHandler()` only, never in `Handler()`.
3. **Rate limiting**: Wire rate-limit middleware on POST endpoint — 10 per site per hour. Use existing `rl.Middleware` token-bucket.

### Important (implement in e74s01)

4. **GET returns metadata only**: `SiteKeyEntry` struct must not include raw token field. List handler returns `key_id`, `name`, `prefix`, `last_used_at`, `created_at`.
5. **Audit events**: Emit `auth.site_key_created` on POST success, `auth.site_key_revoked` on DELETE success.
6. **Input validation**: Validate `name` (max 100 chars, alphanumeric + hyphens), `scopes` (known values only - `["deploy"]`), `site_id` (non-empty, checked against DB).

### Important (implement in e74s02)

7. **Ephemeral token state**: Clear raw token from React state on modal close and component unmount.
8. **Copy-to-clipboard**: Use `navigator.clipboard.writeText` (not `execCommand`).
9. **Warning text**: "This key will not be shown again. Copy it now." before showing raw token.
10. **Error messages**: Generic only — no internal details exposed in UI (consistent with existing error patterns).

### Accepted Risks (no action required)

- **No per-site ownership check** (e74s01): Mitigated by existing sites API protection level. e57 will add `org_id` column for org-level scoping.
- **Fire-and-forget audit** (e74s01): Consistent with existing pattern. Acceptable for audit-grade logging.
- **Clipboard reader access** (e74s02): Standard browser security model. User accepts when clicking copy.
- **No CSRF protection** (e74s01): API uses Bearer token auth, not cookies.

## CWE Mapping

| Finding | CWE |
|---------|-----|
| Secrets Exposure (raw token) | CWE-200 (Information Exposure) |
| IDOR (no per-site ownership) | CWE-639 (Authorization Bypass) |
| Auth Bypass (route mis-registration) | CWE-306 (Missing Authentication) |
| Rate Limiting (resource exhaustion) | CWE-770 (Allocation of Resources) |

## Confidence Rubric

All findings above confidence 8 are actionable. Findings 7 (audit shutdown) and 10 (clipboard) and 11 (CSRF) are below threshold and included for awareness only.
