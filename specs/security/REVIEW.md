# Security Review — Epic e71: Site Route Auth Policy

**Date:** 2026-07-08
**Branch/Diff:** `feat/e71-site-route-auth-policy` (working tree vs main)
**Threat Model:** `specs/security/epics/e71/THREAT_MODEL.md`

## Scope Resolution
This review covers all changes introduced in Epic e71, including:
- Column migration and struct additions for `auth_policy` in `components/sites/sites.go`.
- Policy path matching and JWT/site_key validation in `components/proxy/hosts.go`.
- Strip and inject logic for custom identity headers (`X-BigBase-User-ID`, `X-BigBase-Site-ID`) in `components/proxy/hosts.go`.
- MCP server `set_site_auth_policy` tool implementation in `components/mcp/mcp.go`.

## Vulnerability Assessment

### 1. Route Authorization Bypass (CWE-285)
- **Threat**: Attackers might bypass routing security policies, accessing protected paths without authenticating or accessing site deployment keys improperly.
- **Analysis**: Path checking evaluates public paths first (explicit whitelist), followed by protected paths, and finally defaults to the policy-specified default behavior (deny-by-default if `default == "protected"`). The path matching correctly handles prefix checks with wildcard matching (`/*`).
- **Verdict**: PASS. Confidence: 10/10.

### 2. Header Spoofing and Passthrough Identity Injection (CWE-290 / CWE-306)
- **Threat**: Clients could inject `X-BigBase-User-ID` or `X-BigBase-Site-ID` headers to impersonate other users or sites to backend services.
- **Analysis**: The proxy middleware clears incoming client-provided identity headers (`X-BigBase-User-ID`, `X-BigBase-Site-ID`) immediately upon intercepting the request, prior to performing authentication checks or forwarding requests to downstream handlers. It then reconstructs the headers using the verified authentication context.
- **Verdict**: PASS. Confidence: 10/10.

### 3. SQL Injection (CWE-89)
- **Threat**: Malformed policy JSON injected during MCP write operations could trigger SQL Injection.
- **Analysis**: The database update in both the `sites` component and the `mcp` server tools uses parameter binding (`UPDATE sites SET auth_policy = ? WHERE id = ?`).
- **Verdict**: PASS. Confidence: 10/10.

### 4. Privilege Escalation via MCP Tools (CWE-269 / CWE-862)
- **Threat**: Low-privilege tools or unauthorized agents could configure route policies for other sites.
- **Analysis**: The `set_site_auth_policy` tool is explicitly registered at the `tierWrite` access tier, requiring a valid token with provisioning permissions to execute.
- **Verdict**: PASS. Confidence: 10/10.

## Conclusion
All security threats identified in `specs/security/epics/e71/THREAT_MODEL.md` have been resolved. The code is secure and conforms to security best practices.

**Verdict:** PASS
