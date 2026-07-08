# ADR 0006: MCP Platform Authentication Model

type: adr
context: |
  The MCP HTTP server (mcp.bigbase.click/mcp) shipped write and credential-issuing
  tools with zero authentication (security review 2026-07-08, findings F1+F2: CWE-306,
  CWE-862). Epic e72 must close this. This ADR fixes the domain model for MCP auth so
  implementation (e72s01–s03) has an unambiguous target.

## Decision

Authenticate MCP HTTP tool calls using the **existing organization API key** credential
(`bb_` prefix, `org_api_keys` table), presented as an **`Authorization: Bearer` token**,
enforced by a **deny-by-default three-tier** access model. No new credential type is
introduced.

### 1. Principal = Organization (reuse `bb_` org keys)

The authenticated principal is the **Organization**, identified by an existing `bb_` org
API key via `auth.ResolveAPIKey`. We do **not** invent a `bb_mcp_` credential type.

- **Rationale (DRY/YAGNI):** `org_api_keys`, `ResolveAPIKey`, the `scopes` column, and the
  admin key-management flow already exist. MCP write tools create org-owned resources
  (repos, sites), so the org is the correct ownership boundary. A third credential shape
  in the already-overloaded `org_api_keys` table would add cost for no benefit.

### 2. Transport: Bearer over HTTP only; stdio stays trusted

- MCP HTTP requests authenticate via `Authorization: Bearer <bb_...>` (MCP ecosystem
  convention), **not** the REST `X-API-Key` header.
- The **stdio** transport is unauthenticated by design — it is a single-user local process
  (the operator already owns the host). Auth applies to the HTTP handler only.

### 3. Three-tier, deny-by-default access model

| Tier | Requirement | Tools |
|------|-------------|-------|
| **Public** | none | `ping`, `list_services`, `get_service_docs`, `get_code_example`, `list_frameworks`, `get_ci_template` |
| **Authenticated read** | valid non-revoked `bb_` org key | `list_repos`, `list_sites`, `get_site`, `get_deploy_status`, `get_deploy_logs` |
| **Provision (write)** | valid `bb_` org key **with `mcp:provision` scope** | `create_repo`, `create_site`, `deploy_site`, `provision_ci_credentials`, `list_site_keys`, `revoke_site_key` |

- **Deny-by-default:** a key **without** `mcp:provision` cannot call write tools, even if
  otherwise valid. Empty-scope keys (all keys created before e72) get read access only —
  they are **not** grandfathered into write access.
- The public tier stays public because the MCP server's purpose (e38) is agent-native
  platform *education*; doc tools expose nothing sensitive. Infra-read tools move behind
  auth because they leak repo/site/deploy data and build logs (finding F1).

### 4. Site deploy keys (`bb_dep_`) are rejected by MCP

MCP accepts only `bb_` org keys. A `bb_dep_` site deploy key authenticates the
`/api/deploy` path only; it cannot drive MCP tools. This keeps a leaked deploy key's
blast radius to deployment, not platform enumeration/provisioning.

### 5. Bootstrap via existing admin/CLI, not MCP

The first `mcp:provision` key is minted through the **existing** org API-key creation
path (admin UI / `CreateAPIKey`) with the `mcp:provision` scope. This avoids the
bootstrap paradox (you cannot call an authenticated MCP tool to get your first key).

## Consequences

- **e72s04 (`provision_mcp_credentials` MCP tool) is deferred to 3.1.0.** It only lets an
  already-authenticated org mint *additional* scoped keys for sub-agents — a narrow
  convenience that does not close the vulnerability. Per YAGNI it is out of the
  security-critical path. e72 ships as **5 BCPs** (s01=2, s02=2, s03=1). The bootstrap
  path (§5) already exists, so nothing is blocked by the deferral.
- **ECC dependency direction:** MCP must not import `auth`. MCP defines a
  `KeyAuthenticator` interface (`ResolveOrgKey(token) (orgID int64, scopes []string, err error)`);
  `main.go` injects the auth component, mirroring the existing `SiteKeyCreator` injection.
  (Note: `auth` currently imports `components/mcp` for `SiteKeyEntry` — an existing
  backwards coupling. e72 must not deepen it; new seams point MCP→interface, injected.)
- **`DisableLocalhostProtection: true` is retained.** The server is legitimately reached
  from non-localhost via the proxy; Bearer-token enforcement (not localhost checks) is now
  the protection. Cross-origin browser rebinding cannot forge the `Authorization` header.
- Existing automation using scope-less `bb_` keys for write flows (if any) will break until
  the key is reissued with `mcp:provision`. This is the intended secure-default behavior.
- Scope enforcement is introduced to a column that was previously decorative — this
  establishes the pattern for future scoped tools (a benefit beyond MCP).

## Alternatives considered

- **Dedicated `bb_mcp_` credential type** — rejected: third row-shape + third resolve path
  in an overloaded table (violates KISS/DRY) for no ownership benefit.
- **Grandfather empty-scope keys into write access** — rejected: violates deny-by-default;
  would silently preserve the F2 escalation for existing keys.
- **Authenticate all tools including docs** — rejected: kills the zero-friction education
  purpose of the MCP server with no security gain (docs expose nothing).

## Status

Accepted (e72 planning, 2026-07-08). Supersedes the unauthenticated MCP surface shipped
in e67/e69. Implemented by e72s01 (Bearer middleware), e72s02 (scope gate), e72s03
(public tier). e72s04 deferred to 3.1.0.
