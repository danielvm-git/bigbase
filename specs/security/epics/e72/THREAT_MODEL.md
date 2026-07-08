# Threat Model — e72: MCP Platform Authentication

**Generated:** 2026-07-08  
**Epic:** e72 — Security — MCP Platform Authentication  
**Stories:** e72s01–e72s03 (5 BCPs active; e72s04 deferred to 3.1.0 per ADR 0006)  
**Risk Level:** **CRITICAL** (closes live CWE-306 / CWE-862 on `mcp.bigbase.click`)  
**ADR:** `specs/adr/0006-mcp-platform-auth.md`  
**Prior review:** `specs/security/SECURITY_REVIEW_2026-07-08.md` (F1, F2, F3)

---

## Executive Summary

The MCP HTTP server at `mcp.bigbase.click/mcp` exposes write and credential-issuing tools with **zero authentication**. Epic e72 implements a deny-by-default three-tier model (public / authenticated read / provision-scoped write) using existing `bb_` org API keys via `Authorization: Bearer`. This threat model covers the full epic; e72s01 is the first story in the build cycle.

**Verdict:** Proceed with implementation immediately. No HIGH+ unmitigated threats remain once all three active stories ship. e72s04 is out of the security-critical path.

---

## Surface Area

| Story | Component | Endpoints / Tools | Attack Vectors |
|-------|-----------|-------------------|----------------|
| e72s01 | `components/mcp`, `main.go` | HTTP `/mcp` (all tools) | Missing auth (CWE-306), token forgery, wrong credential type accepted |
| e72s02 | `components/mcp` | Write tools: `create_repo`, `create_site`, `deploy_site`, `provision_ci_credentials`, `list_site_keys`, `revoke_site_key` | Privilege escalation via scope-less keys (CWE-862), horizontal privilege across orgs |
| e72s03 | `components/mcp` | Public tier: `ping`, `list_services`, `get_service_docs`, `get_code_example`, `list_frameworks`, `get_ci_template` | Allow-list misconfiguration exposing mutating tools publicly |
| e72s04 (deferred) | `components/mcp`, `components/auth` | `provision_mcp_credentials` | Bootstrap paradox, scope inflation — deferred; bootstrap uses admin/CLI per ADR 0006 §5 |

**Exposure:** Proxy forwards `mcp.bigbase.click` → MCP backend `:3900` (`components/proxy/hosts_test.go`). `DisableLocalhostProtection: true` in streamable handler — intentional; Bearer auth replaces localhost guard.

**Non-exposed:** Stdio MCP transport (`ServeStdio`) — trusted local operator; no HTTP auth layer.

---

## Known Vulnerabilities (Pre-e72 Baseline)

| ID | CWE | Severity | Location | e72 Mitigation |
|----|-----|----------|----------|----------------|
| F1 | CWE-306 Missing Authentication | **CRITICAL** | `mcp.go:820-844` — no auth on `/mcp` | e72s01 Bearer middleware; e72s03 public allow-list |
| F2 | CWE-862 Missing Authorization | **CRITICAL** | `provision_ci_credentials` → `bb_dep_*` issuance | e72s01 auth + e72s02 `mcp:provision` scope gate |
| F3 | CWE-129 Improper Array Index | LOW | `get_deploy_status` — `deployID[:8]` panic | Fix in e72s01 pass or quick-fix; length guard |

---

## Threat Analysis by Story

### e72s01 — Bearer Middleware (CRITICAL surface)

| Threat | Severity | Mitigation |
|--------|----------|------------|
| **Anonymous write tool invocation** | **CRITICAL** | Reject missing/invalid Bearer with `401` before tool dispatch |
| **bb_dep_ site key accepted as org key** | **HIGH** | Reject non-`bb_` org keys at MCP layer; `ResolveSiteKey` path is deploy-only (ADR 0006 §4) |
| **Revoked key reuse** | **HIGH** | Key store lookup must honor `revoked = 0` (extend `ResolveAPIKey` or new `KeyAuthenticator` seam) |
| **Timing side-channel on key lookup** | LOW | Use constant-time compare on hash (existing `hashAPIKey` + DB lookup pattern) |
| **Token logged in access/error paths** | MEDIUM | Log `org_id` / key id only; never raw Bearer token |
| **Stdio transport accidentally gated** | MEDIUM | Middleware wraps HTTP handler only; `ServeStdio` unchanged |
| **ECC import violation (MCP→auth)** | MEDIUM | `KeyAuthenticator` interface injected from `main.go`; MCP does not import `auth` (ADR 0006) |
| **Cross-org resource access post-auth** | HIGH | Attach `org_id` to context (e57s01 helpers); downstream tools must scope queries by org |

### e72s02 — Scope Gate (CRITICAL for F2)

| Threat | Severity | Mitigation |
|--------|----------|------------|
| **Scope-less key invokes write tools** | **CRITICAL** | `requireScope("mcp:provision")` on every write handler; `403` if absent |
| **Grandfather empty-scope keys into write** | **HIGH** | Deny-by-default — empty scopes = read only (ADR 0006 §3) |
| **Scope string mismatch** | MEDIUM | Align with `org_api_keys.scopes` vocabulary (comma-separated in DB, parsed to `[]string`) |
| **Read tools blocked for valid keys** | LOW | Read tier requires valid key but not `mcp:provision` |

### e72s03 — Public Read Tier

| Threat | Severity | Mitigation |
|--------|----------|------------|
| **Mutating tool on public allow-list** | **CRITICAL** | Explicit allow-list only; deny-by-default for all others |
| **Data leak via misclassified "read" tool** | HIGH | Infra-read tools (`list_repos`, `list_sites`, `get_deploy_logs`, etc.) stay **authenticated**, not public |
| **Public tier enumeration/abuse** | LOW | Static doc content only; rate limiting out of scope |

---

## Vulnerability Categories (Scan Checklist)

| Category | Applicable? | Notes |
|----------|-------------|-------|
| Auth bypass | **YES** | Primary epic goal |
| IDOR / cross-tenant | **YES** | org_id must flow to all DB queries in authenticated tools |
| SQL injection | Checked clean | Bound parameters throughout MCP DB access |
| Secrets exposure | **YES** | Bearer tokens, `bb_dep_*` in responses — never log or persist raw |
| SSRF / command injection | N/A | No exec/filesystem from MCP handlers |
| Crypto flaws | LOW | Reuse existing SHA-256 key hashing from `auth/apikeys.go` |
| Deserialization | N/A | JSON tool args only |

---

## Mitigation Summary

1. **Three-tier deny-by-default** (ADR 0006): public docs → authenticated read → `mcp:provision` write.
2. **`KeyAuthenticator` injection** — MCP defines interface; `main.go` wires auth component.
3. **Reject `bb_dep_` at MCP** — deploy keys cannot drive MCP tools.
4. **Bootstrap outside MCP** — first `mcp:provision` key via admin/CLI `CreateAPIKey`.
5. **Integration tests** — HTTP handler tests for 401/403/200 matrix per tier; proxy route smoke optional.
6. **F3 quick-fix** — safe truncation before `[:8]` slice in `get_deploy_status`.

---

## Out of Scope (Accepted Risks)

- Rate limiting on public or authenticated MCP tiers
- e72s04 `provision_mcp_credentials` (deferred 3.1.0)
- Per-org MCP audit log (e56 covers session audit elsewhere)
- Disabling public HTTP MCP pending e72 (operational mitigation — see SECURITY_REVIEW §Recommended actions)

---

## Implementation Guidance for Developers

| Step | Security requirement |
|------|---------------------|
| Middleware order | Bearer validation → scope check → tool handler |
| Error responses | Generic `401`/`403` messages; details in server logs only |
| Test matrix | No auth + write = 401; valid key no scope + write = 403; valid key + scope + write = 200; public tool no auth = 200 |
| Regression | Existing stdio MCP tests must remain green |

**Verdict:** ✅ Proceed — threat surface is well understood; mitigations map 1:1 to stories e72s01–e72s03.
