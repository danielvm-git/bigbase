# Security Review — Epic e72: MCP Platform Authentication

**Date:** 2026-07-08
**Branch/Diff:** `feat/e72-mcp-platform-auth`
**Threat Model:** `specs/security/epics/e72/THREAT_MODEL.md`

## Scope Resolution
This review covers all changes introduced in Epic e72, including:
- Bearer token middleware (`bearerAuthMiddleware`) and authentication helper (`authenticatePost`) in `components/mcp/auth.go`.
- Scoped tool permissions and public tier mapping.
- Dependency injection of `OrgKeyAuthenticator` into `components/mcp/mcp.go`.
- API key validation enforcing `revoked = 0` in `components/auth/apikeys.go`.
- Main startup composition hook in `main.go`.

## Vulnerability Assessment

### 1. SQL Injection
- All database operations in `components/auth/apikeys.go` use parameterized query placeholders (`?` or `$1`).
- The update to `ResolveAPIKey` correctly appends `AND revoked = 0` to a parameterized query:
  `SELECT id, hash, org_id, scopes, revoked FROM org_api_keys WHERE id = ? AND revoked = 0`

**Verdict:** PASS. Confidence: 10/10.

### 2. Auth Bypass / Privilege Escalation
- `bearerAuthMiddleware` intercepts all POST requests to `/mcp` and checks for tool invocation.
- Write tools (e.g. `create_repo`, `create_site`, `deploy_site`, `provision_ci_credentials`, `list_site_keys`, `revoke_site_key`) are gated at both the middleware and handler-enforcement layers, requiring the `mcp:provision` scope.
- Read-only public tools (e.g. `ping`, `list_services`, `get_service_docs`, `get_code_example`, `list_frameworks`, `get_ci_template`, `deploy_guide`) are mapped explicitly and bypass the auth requirement safely.
- Generic HTTP errors (`StatusUnauthorized` and `StatusForbidden`) are returned, preventing implementation-specific leaks.

**Verdict:** PASS. Confidence: 10/10.

### 3. Case Insensitivity & Tokens Handling
- Authorization header parsing via `bearerToken` uses `strings.EqualFold` for the `"bearer "` prefix, ensuring compatibility with standard clients while preventing auth bypasses.
- Key authentication results (`orgID`, `scopes`) are securely propagated down using contextual typing.

**Verdict:** PASS. Confidence: 10/10.

### 4. Secrets Exposure & Logging
- Log statements via `c.logger.Warn` print only the resolved `tool`, `scopes`, or `err` metrics and the truncated prefix of public fields, never the raw Bearer token values.
- Raw keys are safely validated and never persisted in log output.

**Verdict:** PASS. Confidence: 10/10.

## Conclusion
All security threats identified in `specs/security/epics/e72/THREAT_MODEL.md` (specifically F1, F2, and F3) have been fully addressed and verified by the implementation and its unit tests.

**Verdict:** PASS
