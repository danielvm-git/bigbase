# Security Review — Epic e49 (Security: Auth Hardening)

**Date:** 2026-06-28
**Branch/Diff:** `e49` commits vs `origin/main`

## Scope Resolution
This review covers all changes introduced in Epic e49 including:
- Switch to typed `Claims` for anonymous auth and method restriction bypass in `components/auth/auth.go`
- Host header poisoning mitigation using `PublicURL` in `components/auth/auth.go`
- Path traversal checks in `components/storage/storage.go` for download, thumbnail, and delete operations.

## Context Research & Vulnerability Assessment
1. **Auth Bypass / Privilege Escalation (Anonymous Tokens):**
   - Originally, anonymous tokens could bypass the middleware and make write requests (POST, PATCH, DELETE) to collection endpoints under `org_id = 0`.
   - **Mitigation:** The middleware now strictly rejects non-safe HTTP methods (allowing only GET, HEAD, OPTIONS) for anonymous roles.
   - **Finding:** No unresolved findings. Confidence: 10/10.

2. **Host Header Poisoning (OAuth redirect URI):**
   - OAuth redirect URIs are now constructed using the configured `PublicURL` when set, instead of the client-provided `Host` header.
   - If not set, it logs a startup warning to warn operators. Trailing slashes are cleaned at initialization.
   - **Finding:** No unresolved findings. Confidence: 10/10.

3. **Path Traversal (Storage files):**
   - `handleFileDownload`, `handleThumbnail`, and `handleFileDelete` enforce path validation using `filepath.Abs` and prefix checks to ensure all operations stay within the storage directory boundary.
   - Errors from `filepath.Abs` are checked and handled cleanly.
   - **Finding:** No unresolved findings. Confidence: 10/10.

## Conclusion
All identified security concerns have been addressed. No unresolved HIGH or MEDIUM findings remain.

**Verdict:** PASS
