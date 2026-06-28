# Threat Model — e49: Auth Hardening (Anonymous, OAuth, Path Traversal)

**Generated:** 2026-06-28  
**Epic:** e49 — Security: Auth Hardening  
**Stories:** 3 (e49s01, e49s02, e49s03)  
**Total BCPs:** 5  
**Risk Level:** MEDIUM  

---

## Surface Area

| Story | Component | Endpoints | Attack Vectors |
|-------|-----------|-----------|----------------|
| e49s01 | `components/auth/` | `POST /api/auth/anonymous`, all protected routes via `Middleware()` | JWT claims manipulation, anonymous privilege escalation, token replay |
| e49s02 | `components/auth/` | `GET /api/auth/oauth/google`, `GET /api/auth/oauth/google/callback`, `GET /api/auth/oauth/google/popup` | Host header poisoning, OAuth state CSRF, redirect URI manipulation |
| e49s03 | `components/storage/` | `GET /api/storage/files/{id}` | Path traversal via DB corruption, directory escape via `../` sequences |

---

## Story-by-Story Threat Analysis

### e49s01 — Fix Anonymous Tokens (add org_id context)

**Current vulnerability:** Anonymous tokens minted with `jwt.MapClaims` lack `org_id` field. When parsed by `verifyJWT` into the `Claims` struct, `OrgID` defaults to `0` (zero-value), triggering the "fail closed" rejection at `auth.go:373`:
```go
if claims.OrgID == 0 {
    writeJSON(w, http.StatusForbidden, map[string]string{"error": "no organization"})
    return
}
```
**Impact:** Anonymous auth feature is completely broken — all anonymous tokens return 403.

**Fix approach:** Switch from `jwt.MapClaims` to `Claims` struct with `Role: "anonymous"`, `OrgID: 0`. Add bypass in `Middleware()` before org/email checks:
```go
if claims.Role == "anonymous" {
    ctx = context.WithValue(ctx, ctxUserRole, "anonymous")
    next.ServeHTTP(w, r.WithContext(ctx))
    return
}
```

**Threat assessment of the fix:**

| # | Threat | Severity | Confidence | Notes |
|---|--------|----------|------------|-------|
| 1 | **Anonymous privilege escalation** — anonymous token used on write endpoints | MEDIUM | 9 | Fix adds role-based bypass, but downstream handlers use `UserRoleFromContext` to enforce read-only; this is defense-in-depth, not a new vulnerability. Existing write handlers already reject anonymous. |
| 2 | **Token downgrade via JWT manipulation** — attacker changes `role` from "user" to "anonymous" to bypass email verification | LOW (no exploit) | 5 suppressed | `verifyJWT` verifies the HMAC signature — the `role` field is part of the signed payload, so it cannot be modified without invalidating the token. |
| 3 | **Org isolation bypass** — anonymous token sets no org context, could access cross-org data if downstream doesn't check | LOW | 7 suppressed | Existing behavior for API keys also sets only `ctxOrgID` without user context. Downstream is responsible for org-scoping queries based on context values. Same pattern, not a new risk. |
| 4 | **Inflight token invalidation** — switching from `MapClaims` to `Claims` struct changes JSON field names (e.g., `sub` → `user_id`), invalidating existing tokens | LOW (operational) | 8 | Anonymous tokens have 1-hour TTL. Any token minted before the deploy becomes invalid. Mitigation: accept the break, document it. |

**Verdict:** The fix is a net security improvement. The only residual risk is inflight token invalidation (operational disruption, not a vulnerability). Downstream org isolation is an existing concern shared with API key auth — not introduced by this fix.

---

### e49s02 — Fix OAuth Redirect URI (Host Header Poisoning)

**Current vulnerability (score: 8/10, HIGH):** Four code locations construct OAuth `redirect_uri` from `r.Host`, which is attacker-controlled via the HTTP Host header:

| Location | File | Line |
|----------|------|------|
| `handleGoogleOAuth` | `auth.go:757` | `fmt.Sprintf("%s://%s/api/auth/oauth/google/callback", scheme, r.Host)` |
| `handleGoogleCallback` | `auth.go:805` | `realGoogleVerifier.redirectURI = fmt.Sprintf(..., r.Host)` |
| `handlePopupCallback` | `anonymous.go:80` | `realGoogleVerifier.redirectURI = fmt.Sprintf(..., r.Host)` |
| `handleLinkIdentity` | `usermgmt.go:109` | `realGoogleVerifier.redirectURI = fmt.Sprintf(..., r.Host)` |

**Exploit scenario:**
```bash
# Attacker sends forged Host header
curl -H "Host: evil.com" https://bigbase.click/api/auth/oauth/google
# → Google redirects back to https://evil.com/api/auth/oauth/google/callback?code=AUTH_CODE
# → Attacker captures authorization code on evil.com
# → Attacker exchanges code for victim's JWT token
```

**Exploitability conditions:**
- Attacker must convince victim to click a link that includes a forged `Host` header (possible via HTTP/1.1 request smuggling, or if BigBase is behind a misconfigured reverse proxy that forwards the Host header)
- Google validates `redirect_uri` against registered URIs — attacker would need a Google Cloud Console app registered for `evil.com`, OR exploit a wildcard registration like `*.bigbase.click`
- The OAuth state cookie provides CSRF protection, but the state is not bound to the redirect URI

**Confidence:** 8/10 — well-known Host header poisoning pattern, the OAuth redirect URI is a classic target. Exploitability depends on reverse proxy configuration and Google Cloud Console URI registration.

**Fix approach:** Add `PublicURL` config field that takes precedence over `r.Host`. Fallback to `r.Host` with warning log for backward compatibility.

**Threat assessment of the fix:**

| # | Threat | Severity | Confidence | Notes |
|---|--------|----------|------------|-------|
| 1 | **PublicURL misconfiguration** — admin sets `BIGBASE_PUBLIC_URL` to wrong value, OAuth breaks | LOW (operational) | 6 suppressed | Misconfiguration is an operational risk, not a code vulnerability. |
| 2 | **Fallback still uses Host header** — when PublicURL is not set, existing behavior is preserved (vulnerable) | MEDIUM | 9 | After the fix, deployments without `BIGBASE_PUBLIC_URL` are STILL vulnerable to Host header poisoning. The fix introduces a warning log but doesn't remove the vulnerable code path. Mitigation: document that production deployments MUST set `BIGBASE_PUBLIC_URL`. |
| 3 | **PublicURL validation** — if PublicURL is not validated, could accept invalid/malicious values (e.g., `javascript:alert(1)`) | LOW | 7 suppressed | PublicURL is set via env var or CLI flag — trusted source. An attacker with control of env vars or CLI args already has full compromise. |
| 4 | **Redirect URI in handleLinkIdentity** (usermgmt.go:109) — not in spec scope for e49s02 | MEDIUM | 8 | Same Host header poisoning vulnerability exists in a fourth location not covered by the current story. Should be addressed in this story or a follow-up. |

**Verdict:** The fix is a strong security improvement. However, two residual concerns: (1) the vulnerable fallback code path remains when PublicURL is not configured, and (2) the `handleLinkIdentity` location (usermgmt.go:109) is not covered by the fix's 3 stated locations. These should be tracked as hardening items.

---

### e49s03 — Path Traversal Defense for File Downloads

**Current vulnerability (score: 8/10, MEDIUM):** `handleFileDownload` (storage.go:343) joins DB-stored path with storage directory without `filepath.Clean` or prefix boundary check:
```go
fullPath := filepath.Join(s.dir, filePath)  // ← no Clean, no boundary check
if _, err := os.Stat(fullPath); ...         // ← Stat happens before validation
http.ServeFile(w, r, fullPath)
```

**Exploit scenario:**
- If a corrupted DB record stores `path = "../../../etc/passwd"`:
  - `filepath.Join("data/storage", "../../../etc/passwd")` → `"data/storage/../../../etc/passwd"`
  - `http.ServeFile` resolves the relative path → serves `/etc/passwd`
- Requires DB corruption or SQL injection — this is a secondary attack (the primary attack gets DB write access, then path traversal enables reading arbitrary files)

**Existing defense (handleThumbnail, storage.go:319):**
```go
fullPath := filepath.Join(s.dir, filepath.Clean(filePath))
absDir, _ := filepath.Abs(s.dir)
absPath, _ := filepath.Abs(fullPath)
if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) {
    writeJSON(w, http.StatusForbidden, ...)
    return
}
```

**Confidence:** 8/10 — clear path traversal pattern. Requires DB corruption as pre-condition, which limits practical exploitability but defense-in-depth is still warranted.

**Fix approach:** Replicate the three-step `handleThumbnail` pattern: `filepath.Clean` → `filepath.Abs` → prefix check. Move validation BEFORE `os.Stat`.

**Threat assessment of the fix:**

| # | Threat | Severity | Confidence | Notes |
|---|--------|----------|------------|-------|
| 1 | **Race condition (TOCTOU)** — path validated, then file served, path could change between check and serve | LOW | 6 suppressed | `http.ServeFile` follows symlinks by default; but the storage directory is managed by BigBase, and an attacker would need write access to the storage dir to exploit. Excluded per hard exclusion #8 (theoretical race conditions). |
| 2 | **Symlink bypass** — attacker creates symlink inside storage dir pointing outside | LOW | 7 suppressed | `http.ServeFile` resolves symlinks. The prefix check uses `filepath.Abs` which resolves symlinks, so `absPath` would show the real target. However, a symlink created AFTER the check and BEFORE `ServeFile` could bypass. Extremely unlikely — requires concurrent write access. |
| 3 | **handleFileDelete path traversal** — uses `filepath.Join(s.dir, id)` where `id` is hex | LOW | 5 suppressed | `id` is a hex string from the DB (e.g., `abc123`), not attacker-controlled. Not exploitable. |

**Verdict:** The fix is a clear defense-in-depth improvement. No new attack surface introduced. The TOCTOU concern is theoretical and excluded per the false-positive rules.

---

## Epic-Level Risk Summary

| Category | Before Fix | After Fix | Delta |
|----------|-----------|-----------|-------|
| Host header poisoning (e49s02) | HIGH — 4 exploitable endpoints | MEDIUM — 1 remaining fallback path + 1 uncovered endpoint | ↓ |
| Path traversal (e49s03) | MEDIUM — requires DB corruption | NONE — path traversal blocked | ↓ |
| Anonymous auth (e49s01) | NONE — feature is broken (403 always) | LOW — anonymous read-only with token invalidation on deploy | ↑ (feature restored) |
| **Overall** | **HIGH** | **MEDIUM** | **Net improvement** |

---

## Mitigation Guidance

### High Priority
1. **e49s02 — Cover all 4 redirect URI locations:** The `handleLinkIdentity` in `usermgmt.go:109` must also be updated to use `publicURLOrDefault(r)`. Add it to the story scope.
2. **e49s02 — Make PublicURL required in production:** After grace period, remove the `r.Host` fallback and require `BIGBASE_PUBLIC_URL` to be set in production. Log an error (not just warning) on startup if unset.

### Medium Priority
3. **e49s01 — Downstream handler hardening:** Audit all handlers that access org context to ensure they handle `OrgID = 0` gracefully (return 403/404, not panic). This is a defense-in-depth concern shared with API key auth.
4. **e49s01 — Token invalidation grace period:** Consider supporting both old `MapClaims` and new `Claims` formats for 1 hour after deploy, or document the deploy-time impact clearly.

### Low Priority
5. **e49s03 — Shared helper extraction:** After both `handleThumbnail` and `handleFileDownload` use the same pattern, extract into `validatePath(dir, filePath string) (string, error)` helper to prevent future regressions.
6. **Rate limiting on anonymous token minting:** Already tracked by e47 — no additional action needed.

---

## Excluded from Scope (By Design)

- Configurable anonymous token TTL (hardcoded 1 hour)
- Rate limiting on anonymous endpoints (e47)
- Google Cloud Console redirect URI whitelist update (ops)
- PublicURL format validation at startup (follow-up)
- Path traversal in `handleFileDelete` (uses hex IDs, not exploitable)
- Path traversal in `writeFile` (uses `filepath.Base` on filenames)
