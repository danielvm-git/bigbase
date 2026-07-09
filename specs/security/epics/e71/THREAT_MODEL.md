# Threat Model — e71: Host-Level Route Auth Policy

**Generated:** 2026-07-08  
**Epic:** e71 — Security — Host-Level Route Auth Policy  
**Stories:** e71s01–e71s04  
**Risk Level:** **HIGH** (guards deployed sites from unauthorized access to static files and backends)  

---

## Executive Summary

Epic e71 introduces host-level routing security policies for deployed sites. Site operators can configure routing rules (public vs. protected paths) and authentication constraints (accepting JWT tokens or site deploy keys). The proxy layer enforces these policies prior to routing traffic to static assets or backend ports.

---

## Surface Area

| Story | Component | Endpoints / Path | Attack Vectors |
|-------|-----------|------------------|----------------|
| e71s01 | `components/sites` | `GET/POST /api/sites/:id/auth-policy` | SQL Injection, Auth Bypass, privilege escalation, malformed policy storage |
| e71s02 | `components/proxy` | Host routing / middleware | Open access to protected paths, validation bypass, timing side-channels |
| e71s03 | `components/proxy` | Proxy header forwarding | Header spoofing (`X-BigBase-User-ID`), credential leakage |
| e71s04 | `components/mcp` | MCP tool `set_site_auth_policy` | Unauthorized tool execution, SQL injection, cache inconsistency |

---

## Threat Analysis by Story

### e71s01 — Site Auth Policy Schema
- **Threat**: Malformed JSON or policy bypass.
- **Mitigation**: Strict schema validation of the JSON payload. Ensure `ALTER TABLE` is idempotent and safe.

### e71s02 — Proxy JWT Validation
- **Threat**: Access to `/books/*` or other sensitive static content without a valid token.
- **Mitigation**: Deny-by-default logic for protected paths. Verify signature and expiry of the JWT token or the site deploy key.

### e71s03 — Passthrough Auth Injection
- **Threat**: Client injects `X-BigBase-User-ID` directly, spoofing identity to backend functions.
- **Mitigation**: Explicitly strip/override `X-BigBase-User-ID` from all incoming client requests in the proxy.

### e71s04 — MCP Tool
- **Threat**: Unauthorized agent modifies security policy.
- **Mitigation**: Access control at `tierWrite` (`mcp:provision` scope required).

---

## Verdict: ✅ Proceed
Mitigations are mapped 1:1 to stories.
