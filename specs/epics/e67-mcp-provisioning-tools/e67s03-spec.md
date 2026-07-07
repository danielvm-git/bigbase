# e67s03: MCP provision_ci_credentials Tool

**Story ID:** e67s03 | **Epic:** e67 — MCP Provisioning Tools | **BCPs:** 2 | **Status:** planned

## 1. Type & Context

**type:** feat
**context:** domain
**maturity:** 3 — Countable

## 2. Story Statement

**As an** AI coding agent using the BigBase MCP server,
**I want** to provision scoped CI/CD credentials (a site-scoped deployment key) for a site,
**so that** I can inject them into a GitHub repository's secrets via `gh secret set` without ever needing a human to generate credentials through the UI.

## 3. Context

The Auth component (`components/auth/apikeys.go`) already has a fully operational credential system:

```go
// Already exists — battle-tested:
func generateRawAPIKey() // "bb_" + hex(32 random bytes)
func hashAPIKey(raw)     // SHA-256 hex digest
func CreateAPIKey(...)   // INSERT into org_api_keys, return raw key once
func ResolveAPIKey(raw)  // lookup by hash, return org_id
```

The middleware (`auth.go:431`) already has a fast-path for `bb_` prefixed tokens that skips JWT validation:

```go
if strings.HasPrefix(token, "bb_") {
    orgID, _ := a.ResolveAPIKey(token)
    // sets ctxOrgID, bypasses JWT entirely
}
```

This story does NOT build a new credential system. It extends the existing one by adding **site-scoped keys** alongside the existing org-scoped keys — sharing the same SHA-256 hashing, the same `org_api_keys` table, the same middleware fast-path, and the same `key_hash` lookup. The only difference: a `bb_dep_` prefix instead of `bb_`, and a `site_id` alongside `org_id = 0`.

### Zoom-Out Summary

- **Module purpose:** `components/auth` is extended (not `components/mcp`). The MCP component delegates to auth via a thin `SiteKeyCreator` interface. Zero crypto code in MCP.
- **Callers:** AI clients via MCP SSE/stdio for provisioning; deploy API handlers for validation.
- **Infrastructure reused:** `hashAPIKey()`, `generateRawAPIKey()`, `org_api_keys` table, `bb_` middleware fast-path, `crypto/rand` + `crypto/sha256` imports — all already exist.
- **New code:** ~80 lines — `generateSiteKey()` (one-liner), `CreateSiteKey()` (~25 lines), `ResolveSiteKey()` (~15 lines), middleware extension (~15 lines), context key + deploy handler enforcement (~15 lines), MCP tool registration (~10 lines).

## 4. Domain Model

**No new table.** Extend the existing `org_api_keys` table:

```sql
-- Existing schema (components/auth/apikeys.go:32):
CREATE TABLE IF NOT EXISTS org_api_keys (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id       INTEGER NOT NULL,
    key_hash     TEXT NOT NULL UNIQUE,    -- SHA-256 of raw key
    name         TEXT NOT NULL,
    scopes       TEXT NOT NULL DEFAULT '',
    last_used_at TEXT,
    created_at   TEXT NOT NULL
);

-- Migration (this story adds):
ALTER TABLE org_api_keys ADD COLUMN site_id TEXT NULL REFERENCES sites(id);
ALTER TABLE org_api_keys ADD COLUMN revoked INTEGER NOT NULL DEFAULT 0;
```

A row is either — identified by `org_id` sentinel:

| `org_id` | `site_id` | Meaning |
|----------|----------|---------|
| >0 | NULL | Org-scoped API key (existing behavior) |
| **0** | NOT NULL | Site-scoped deployment key (new) |

`org_id = 0` is the established pattern for "not owned by any org" (e.g. `git_repos.owner_id = 0` in `components/git/git.go:165`). The column remains `NOT NULL` — no schema break.

The `revoked` column enables soft-delete for both key types (currently API keys are hard-deleted — this is a forward-looking improvement).

```
SiteKey {
    ID:        int64    // auto-increment
    SiteID:    string   // FK to sites.id
    Name:      string   // human-readable label (default "ci-bot")
    Scopes:    []string // e.g. ["deploy"]
    Key:       string   // raw token — returned once at creation
    CreatedAt: string   // RFC3339
}
```

## 5. Contract / Interface

### New functions in `components/auth/apikeys.go` (no new file)

```go
// generateSiteKey creates a site-scoped key with "bb_dep_" prefix.
// Reuses the same crypto as generateRawAPIKey (32 random bytes, hex-encoded).
func generateSiteKey() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return "bb_dep_" + hex.EncodeToString(b), nil
}

// CreateSiteKey generates a site-scoped deployment key.
// Uses hashAPIKey (already exists) — same SHA-256 as org keys.
// Validates site exists before inserting. Stores org_id = 0 as sentinel.
// The raw key is returned once; only the hash is stored.
func (a *Auth) CreateSiteKey(ctx context.Context, siteID, name string, scopes []string) (*SiteKeyCreated, error)

// ResolveSiteKey looks up the site_id for a bb_dep_ prefixed key.
// Uses hashAPIKey (same hash lookup as org keys).
// Returns error when not found or revoked.
func (a *Auth) ResolveSiteKey(rawKey string) (string, error)
```

### New context key in `kernel/context.go` (or `kernel/kernel.go`)

```go
// In kernel/ — neutral package imported by both auth and deploy (no ECC violation).
// Add alongside existing shared kernel types (DBer, Logger, etc.).

type contextKey string

const CtxSiteID contextKey = "site_id"

func SiteIDFromContext(ctx context.Context) (string, bool) {
    id, ok := ctx.Value(CtxSiteID).(string)
    return id, ok
}
```

Both components already import `kernel` — no new imports, no component coupling.

**Reason for Depth:** A neutral context helper keeps the site-scope authorization signal shared between auth and deploy without making either component import the other or duplicating stringly typed context keys.

### Middleware extension (`auth.go:431`)

```go
// Existing fast-path — extended with one branch:
if strings.HasPrefix(token, "bb_") {
    if strings.HasPrefix(token, "bb_dep_") {
        siteID, err := a.ResolveSiteKey(token)
        if err != nil {
            writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid site key"})
            return
        }
        r = r.WithContext(context.WithValue(r.Context(), kernel.CtxSiteID, siteID))
        next.ServeHTTP(w, r)
        return
    }
    // Existing org-scoped path — unchanged
    orgID, err := a.ResolveAPIKey(token)
    // ...
}
```

### Deploy handler enforcement (`components/deploy/deploy.go:HandleCreate`)

When a request arrives with a site-scoped key (ctxSiteID set), the deploy handler MUST reject cross-site deployments:

```go
// In HandleCreate, after parsing req:
if siteID, ok := kernel.SiteIDFromContext(r.Context()); ok && siteID != "" {
    if req.SiteID != "" && req.SiteID != siteID {
        writeJSON(w, http.StatusForbidden, map[string]string{"error": "site key not authorized for this site"})
        return
    }
    // Force req.SiteID to match the token's site scope
    req.SiteID = siteID
}
```

This is the **authorization** half of the `bb_dep_` token — the middleware handles **authentication** (who you are), the deploy handler enforces **authorization** (you can only deploy the site your key is scoped to).

### MCP interface (in `mcp/mcp.go`)

```go
// SiteKeyCreator provisions site-scoped deployment credentials.
// Implemented by the auth component; thin delegator — no crypto in MCP.
type SiteKeyCreator interface {
    CreateSiteKey(ctx context.Context, siteID, name string, scopes []string) (rawToken, keyID string, err error)
}

type Options struct {
    // ... existing fields ...
    SiteKeyCreator SiteKeyCreator // optional; nil disables provision_ci_credentials
}
```

### MCP tool signature

```
provision_ci_credentials(site_id, name?)
  → { token: "bb_dep_<64 hex chars>", key_id: "42", site_id: "..." }
```

## 6. Implementation Strategy

1. **Migration** — `ALTER TABLE org_api_keys ADD COLUMN site_id TEXT NULL` + `ADD COLUMN revoked INTEGER DEFAULT 0` in `apikeys.go:migrateAPIKeysTable()`.
2. **Context key** — Add `CtxSiteID` + `SiteIDFromContext()` to `kernel/` (neutral package, imported by both auth and deploy — no ECC violation). Auth sets it; deploy reads it.
3. **`generateSiteKey()`** — one-line variant of `generateRawAPIKey()` with `"bb_dep_"` prefix.
4. **`CreateSiteKey()`** — mirrors `CreateAPIKey()`: validate site exists (`SELECT 1 FROM sites WHERE id = ?`), call `generateSiteKey()`, `hashAPIKey()`, INSERT with `site_id` set and `org_id = 0`, return raw key once.
5. **`ResolveSiteKey()`** — mirrors `ResolveAPIKey()`: `hashAPIKey(raw)` → `SELECT site_id FROM org_api_keys WHERE key_hash = ? AND revoked = 0`. Note: the hash comparison is performed by SQLite's `WHERE key_hash = ?` — no additional constant-time comparison in Go is needed (single-row lookup by primary unique key; no timing channel exists).
6. **Middleware extension** — second `strings.HasPrefix(token, "bb_dep_")` check in the existing fast-path block at `auth.go:431`. Sets `kernel.CtxSiteID` on context.
7. **Deploy handler enforcement** — In `HandleCreate`, after parsing the request body: if `kernel.SiteIDFromContext(ctx)` returns a site_id, enforce `req.SiteID == ctxSiteID` (403 on mismatch). If `req.SiteID` is empty, default it to `ctxSiteID`. No import of components/auth — kernel is neutral.
8. **`SiteKeyCreator` interface** — add to `mcp.Options`, wire `auth` component in `main.go`.
9. **MCP tool registration** — `provision_ci_credentials` in `NewMCPServer()`.
10. **Tests** — mock `SiteKeyCreator` in `mcp_test.go`; integration tests for key creation + validation + cross-site rejection in `auth_test.go` and `deploy_test.go`.

## 7. Data Flow

```
AI agent: provision_ci_credentials({"site_id": "04c58b9...", "name": "library-bot"})
  → mcp.NewMCPServer handler
      → c.siteKeyCreator.CreateSiteKey(ctx, "04c58b9...", "library-bot", ["deploy"])
          → auth.CreateSiteKey:
              1. SELECT 1 FROM sites WHERE id = ? (validate site exists)
              2. generateSiteKey() → "bb_dep_<64 hex chars>"
              3. hashAPIKey(raw)    → SHA-256 hex (reuses existing function)
              4. INSERT INTO org_api_keys (org_id=0, site_id, key_hash, name, scopes, ...)
              5. RETURN raw key (one time only)
      → return { token: "bb_dep_...", key_id: "42", site_id: "04c58b9..." }

CI/CD pipeline — valid deployment:
  curl -H "Authorization: Bearer bb_dep_..." -d '{"site_id":"04c58b9..."}' /api/deploy
    → auth.Middleware:
        1. "bb_" prefix → "bb_dep_" prefix → ResolveSiteKey → ctxSiteID="04c58b9..."
    → deploy.HandleCreate:
        1. ctxSiteID="04c58b9...", req.SiteID="04c58b9..." → match → proceed

CI/CD pipeline — cross-site attack (BLOCKED):
  curl -H "Authorization: Bearer bb_dep_A..." -d '{"site_id":"other-site"}' /api/deploy
    → auth.Middleware:
        1. "bb_dep_" prefix → ResolveSiteKey → ctxSiteID="04c58b9..."
    → deploy.HandleCreate:
        1. ctxSiteID="04c58b9...", req.SiteID="other-site" → MISMATCH → 403
```

## 8. Error Handling

| Condition | Tool response |
|-----------|---------------|
| `SiteKeyCreator` not configured | "Credential provisioning requires an Auth component." |
| `site_id` is empty | "site_id is required. Use list_sites or create_site to get one." |
| Site not found | "Site '...' not found." |
| Key generation fails | "Failed to generate key: <err>" |

| Condition | Middleware/Handler response |
|-----------|---------------------------|
| Invalid/revoked site key | 401 "invalid site key" |
| Cross-site deployment | 403 "site key not authorized for this site" |

## 9. Testing Strategy

- **Unit — MCP layer:** Mock `SiteKeyCreator` returns token + key_id. Verify tool response contains both.
- **Unit — nil creator:** Tool returns "not configured."
- **Unit — missing site_id:** Tool returns validation error.
- **Unit — `generateSiteKey()`:** Verify output matches `^bb_dep_[0-9a-f]{64}$`.
- **Unit — site existence:** `CreateSiteKey` with nonexistent site → error "Site '...' not found."
- **Integration — create + validate:** `CreateSiteKey()` → returns raw token → `ResolveSiteKey(raw)` → returns correct `site_id`.
- **Integration — revoked key:** Set `revoked=1`, `ResolveSiteKey()` → returns error.
- **Integration — hash security:** Raw token is nowhere in `org_api_keys` — only `key_hash`.
- **Integration — cross-site rejection:** Site key for site A → deploy to site B → 403 Forbidden.
- **Integration — middleware + handler:** HTTP request with `bb_dep_` token sets `ctxSiteID`; `HandleCreate` accepts matching `site_id` and rejects mismatched.
- **Regression:** Existing `bb_` API keys still work (org-scoped path unchanged).

## 10. Migration / Rollback

Adds two nullable columns. Rollback:

```sql
-- Safe: columns are nullable, no data migration needed.
-- SQLite doesn't support DROP COLUMN directly; create new table without columns.
-- Or: simply stop writing to site_id/revoked and ignore them.
```

Code rollback: remove `ctxSiteID` key + middleware branch, `CreateSiteKey`/`ResolveSiteKey`/`generateSiteKey` functions, `SiteKeyCreator` interface, MCP tool registration, `main.go` wiring, deploy handler enforcement.

## 11. Documentation

Update `specs/tech-architecture/tech-stack.md` MCP section to list `provision_ci_credentials`. Update auth section to document `bb_dep_` prefix, `ctxSiteID`, and the deploy handler authorization enforcement.

## 12. Dependencies

- `crypto/rand`, `crypto/sha256`, `encoding/hex` — all stdlib, already imported in `apikeys.go`.
- `components/auth` — existing component being extended.
- `components/deploy` — deploy handler needs cross-site check.
- `github.com/modelcontextprotocol/go-sdk` — already in `go.mod` [OK].
- No new external packages.

## 13. Observability

```go
// In auth.CreateSiteKey:
a.logger.Info("site key created", "key_id", id, "site_id", siteID)

// In MCP tool:
c.logger.Info("mcp tool", "tool", "provision_ci_credentials", "site_id", siteID)

// In deploy handler cross-site rejection:
d.logger.Warn("deploy rejected: site key mismatch",
    "ctx_site_id", ctxSiteID, "req_site_id", req.SiteID)

// In middleware (Debug level — high frequency):
a.logger.Debug("site key resolved", "site_id", siteID)
```

## 14. Security

**Security level:** medium — keys are deployment credentials with authorization enforcement.

- **Hashing:** SHA-256 via existing `hashAPIKey()` — raw key never stored.
- **Hash comparison:** Performed by SQLite `WHERE key_hash = ?` on a `UNIQUE` column — single-row index lookup, no timing channel. No additional constant-time comparison needed in Go.
- **Revocation:** `revoked` column enables soft-delete. `ResolveSiteKey` checks `WHERE revoked = 0`.
- **Authorization:** Deploy handler enforces `ctxSiteID == req.SiteID` (403 on cross-site). Site-scoped keys cannot deploy to other sites.
- **Key rotation:** Call `provision_ci_credentials` again → new key created, old key optionally revoked.
- **Prefix-based routing:** `bb_dep_` prefix enables middleware fast-path without DB call to determine key type.
- **Minimal scope:** Default scopes are `["deploy"]` — the key can only trigger deployments, not read data or manage users.
- **Exposure:** MCP server on port 3900, not publicly proxied. Raw key returned once; MCP client is responsible for secure storage (e.g., `gh secret set`).

## 15. Acceptance Criteria

```gherkin
Scenario: provision_ci_credentials generates a deploy key
  Given a site with id "site-1" exists
  When an AI agent calls provision_ci_credentials with {"site_id": "site-1"}
  Then the response contains a token matching ^bb_dep_[0-9a-f]{64}$
  And the response contains a key_id

Scenario: token authenticates deploy API calls for the correct site
  Given a valid site-scoped key for site "site-1"
  When a request is made to POST /api/deploy with "Authorization: Bearer <token>"
  And the body contains {"site_id": "site-1"}
  Then the request is accepted (200)

Scenario: cross-site deployment is rejected
  Given a valid site-scoped key for site "site-1"
  When a request is made to POST /api/deploy with "Authorization: Bearer <token>"
  And the body contains {"site_id": "site-2"}
  Then the response is 403 Forbidden
  And the error message says "site key not authorized for this site"

Scenario: provision_ci_credentials validates site_id
  When an AI agent calls provision_ci_credentials without site_id
  Then the response says "site_id is required"

Scenario: revoked key is rejected by middleware
  Given a key that has been revoked (revoked=1)
  When a request is made with this key
  Then the response is 401 "invalid site key"

Scenario: raw key is not stored in the database
  When a site key is provisioned
  Then the org_api_keys table contains its SHA-256 hash in key_hash
  And the raw token is not present in any column

Scenario: existing org-scoped API keys still work
  Given an existing bb_ prefixed API key (org_id > 0, site_id NULL)
  When a request is made with this key
  Then the request is accepted (org-scoped path unchanged)
```

## 16. Out of Scope

- Token rotation and revocation UI in the admin panel.
- `revoke_site_key` MCP tool (future story — soft-delete by `key_id` is already supported via the `revoked` column, just not exposed as an MCP tool yet).
- Scoped permissions beyond `["deploy"]` (e.g., `["deploy", "read:logs"]`). The `scopes` column supports this — deferred to a future story.
- Email/password bot account provisioning (Option B from alternatives — more complex, not needed given this approach).

## Alternatives Considered

**Option A (chosen) — Extend `org_api_keys`:** Reuse existing SHA-256 hashing, prefix-based middleware fast-path, and `org_api_keys` table. Add `site_id` + `revoked` columns. Two key types, one infrastructure. Matches Appwrite's proven model (one key concept, type prefix, resource-specific scoping).

**Option B — Separate `deployment_tokens` table:** New table with its own hashing, its own validation path, its own middleware branch. Duplicates 80% of `org_api_keys` infrastructure. Rejected — violates DRY, creates maintenance divergence risk, and would have missed the deploy authorization enforcement (GAP-4) that the unified middleware pattern naturally catches.

**Option C — Bot user account:** Register a headless user via `auth.Register()`, generate JWT. More complex (email uniqueness, password hashing, login flow), and JWT expiry means the CI pipeline needs token refresh logic. Rejected — site-scoped keys are stateless (hash lookup, no expiry).
