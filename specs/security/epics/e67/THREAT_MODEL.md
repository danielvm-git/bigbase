# Threat Model — e67: MCP Provisioning Tools

**Generated:** 2026-07-07  
**Epic:** e67 — MCP Provisioning Tools — Zero-to-Production from AI Agents  
**Stories:** e67s01–e67s04 (7 BCPs)  
**Risk Level:** MEDIUM (elevated by e67s03 site-scoped deployment keys)

---

## Surface Area

| Story | Component | Endpoints / Tools | Attack Vectors |
|-------|-----------|-------------------|----------------|
| e67s01 | `components/git`, `components/mcp` | MCP `create_repo` | Repo name injection, unauthorized repo creation |
| e67s02 | `components/sites`, `components/mcp` | MCP `create_site` | Site provisioning for repos attacker doesn't own |
| e67s03 | `components/auth`, `components/deploy`, `components/mcp` | MCP `provision_ci_credentials`; `POST /api/deploy` with `bb_dep_` token | Cross-site deploy, key theft, privilege escalation |
| e67s04 | `components/mcp` | MCP `get_ci_template` | None (static public templates) |

MCP server listens on port 3900 by default — internal/trusted network assumption. Not proxied publicly in default config.

---

## Threat Analysis by Story

### e67s01 — create_repo (LOW)

| Threat | Severity | Mitigation |
|--------|----------|------------|
| Empty/malicious repo names | LOW | Non-empty validation; UNIQUE constraint on name |
| MCP without Git component | NONE | Nil `GitCreator` returns "not configured" |
| Resource exhaustion (unlimited repos) | LOW | Out of scope — existing Git API has same surface |

### e67s02 — create_site (LOW)

| Threat | Severity | Mitigation |
|--------|----------|------------|
| Site for nonexistent repo | LOW | `SELECT name FROM git_repos WHERE id = ?` before INSERT |
| Auto-deploy abuse | LOW | Requires wired `Deployer`; same as existing sites HTTP path |

### e67s03 — provision_ci_credentials (HIGH — hard gate)

| Threat | Severity | Mitigation |
|--------|----------|------------|
| **Cross-site deployment** | **HIGH** | `kernel.SiteIDFromContext` + deploy `HandleCreate` 403 on mismatch |
| **bb_dep_ misrouted to org ResolveAPIKey** | **HIGH** | Check `bb_dep_` prefix **before** generic `bb_` in middleware |
| Raw token stored in DB | HIGH | SHA-256 via existing `hashAPIKey()`; raw returned once |
| Revoked key reuse | MEDIUM | `WHERE revoked = 0` in `ResolveSiteKey` |
| Token logged | MEDIUM | Log `key_id` + `site_id` only, never raw token |
| MCP exposes token to agent | MEDIUM | Agent responsibility to `gh secret set`; MCP internal-only |

### e67s04 — get_ci_template (NONE)

Static embedded YAML with GitHub Actions `${{ secrets.* }}` placeholders. No secrets, no user input persistence.

---

## Mitigation Summary

1. Site-scoped keys use `bb_dep_` prefix with dedicated middleware branch.
2. Deploy handler enforces `req.SiteID == ctxSiteID` when site key authenticated.
3. Parameterized SQL for all key lookups and site validation.
4. Generic client errors ("invalid site key") — details in server logs only.

---

## Out of Scope (Accepted Risks)

- MCP port 3900 exposure if operator binds publicly without auth
- Rate limiting on MCP tool calls
- Key rotation/revocation MCP tool (revoked column exists; UI/tool deferred)

**Verdict:** Proceed with implementation. e67s03 requires integration tests for cross-site 403 before marking story done.
