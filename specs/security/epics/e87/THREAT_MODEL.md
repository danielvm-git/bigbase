# Threat Model: e87 — WCAG 2.2 AAA Conformance

**Date**: 2026-08-11
**Epic**: e87 — WCAG 2.2 AAA Conformance
**Nature**: UI/CSS accessibility enhancement (7 stories) + one auth-touching story (s05 session timeout). No new data surfaces, no new endpoints (s05 reuses `POST /api/auth/refresh`).
**Risk Level**: LOW (UI-only); s05 raises it to MEDIUM locally (JWT handling)

## 1. Attack Surface

| Story | Surface | New/Existing | Security Risk |
|-------|---------|--------------|---------------|
| e87s01 | CSS tokens (colors) | Modified (CSS) | NONE |
| e87s02 | CSS focus ring | Modified (CSS) | NONE |
| e87s03 | Component sizing | Modified (UI) | NONE |
| e87s04 | Text/abbr markup | Modified (UI text) | NONE |
| e87s05 | JWT exp parsing + refresh call + route restore via sessionStorage | New client logic | MEDIUM |
| e87s06 | Confirm dialogs / hints | Modified (UI) | NONE |
| e87s07 | rem tokens + breadcrumbs | Modified (CSS/UI) | NONE |

## 2. Trust Boundaries (s05 — the only boundary crossing)

```
[Browser] ──reads JWT exp claim (client-decodable, public)──> [AuthContext]
    │
    ├── POST /api/auth/refresh (existing endpoint, refresh token) ──> [Auth server]
    └── sessionStorage (route restore) ──> [LoginPage redirect]
```

- JWT exp is **not secret** (base64 payload, signed) — client-side expiry countdown is safe.
- The refresh call must use the existing server-validated refresh-token flow — no new trust.
- `sessionStorage` route restore: **origin-scoped** by the browser; only a pathname/search string is stored — must NOT store tokens or sensitive params (query strings could carry e.g. site IDs — acceptable; secrets never).

## 3. Vulnerabilities & Mitigations

| # | Category | Finding | Mitigation | Verify |
|---|----------|---------|-----------|--------|
| 1 | CWE-613 Session Expiration | Client-side countdown could desync from server exp | Countdown is UX only; server still rejects expired JWTs (existing middleware). Never trust client "still valid" | e87s05t4 (E2E with 60s expiry) |
| 2 | CWE-384 Session Fixation | Route-restore redirect could replay a stale route after re-auth | Restore only pathname+search from sessionStorage, clear immediately on read; login always issues fresh tokens | e87s05t3 |
| 3 | CWE-201 Info Exposure | Query-string params in the restored route (e.g. ?siteId=) replayed | Acceptable — same user re-auth, org-scoped server checks still apply; document | e87s05t3 |
| 4 | CWE-79 XSS via abbr/glossary (s04) | New glossary surface could inject HTML | Glossary content is static, developer-authored; render as text/React children, never dangerouslySetInnerHTML | e87s04t4 |

## 4. Risk Assessment

| Dimension | Rating | Rationale |
|-----------|--------|-----------|
| Production Risk | LOW | 6/7 stories are CSS/UI text; no behavior change |
| Auth Risk (s05) | MEDIUM | New client JWT handling + refresh orchestration — reuses existing server endpoints, no new trust boundary |
| Regression Risk | MEDIUM | px→rem conversion (s07) and 44px sizing (s03) can shift layout — e2e + component suites are the gate |
| Density Risk (product) | MEDIUM | s03 44px targets degrade admin density — documented design tradeoff, not a security issue |

## 5. Security Gate

- s05 tasks carry `security: low` — verify steps must include the token-lifecycle E2E (refresh + expiry paths)
- All other stories `security: none` — UI-only, no new findings expected
- `tests/e2e/axe-scan.spec.ts` re-run is the conformance regression gate (17/17 routes, 0 violations)

## 6. CWE Mapping

1 → CWE-613 · 2 → CWE-384 · 3 → CWE-201 · 4 → CWE-79
