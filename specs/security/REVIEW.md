# Security Review — Epic e85: Unified Theming (Landing Page Parity)

**Date:** 2026-08-11
**Branch/Diff:** `feat/e85-unified-theming` (vs main)
**Threat Model:** `specs/security/epics/e85/THREAT_MODEL.md` (refreshed for this diff; e71's review archived under `specs/security/epics/e71/`)

## Scope Resolution
This review covers all changes in epic e85:
- `components/proxy/themes.go` (new): `landingAccents` data table + `landingThemeScript()` inline-`<script>` generator.
- `components/proxy/proxy.go`: `{{.ThemeScript}}` injected in `<head>` of `homeTemplate`; dark overrides refactored from `@media (prefers-color-scheme: dark)` to `[data-theme="dark"]` + `:root:not([data-theme])` no-JS fallback; rainbow (June) gradient CTA.
- `components/proxy/themes_test.go` + `tests/e2e/theme-parity.spec.ts`: tests only, no runtime surface.

## Vulnerability Assessment

### CWE-79 (XSS) — inline script
**Not applicable (mitigated).** The script reads same-origin `localStorage` (`bigbase-theme`, `bigbase-accent`) and writes only **enum-validated** values to the DOM:
- `bigbase-theme` accepted only if exactly `'light'|'dark'`, else derived from `matchMedia`.
- `bigbase-accent` accepted only if a key of the hardcoded 13-entry `ACCENTS` table, else `'default'`.
No value reaches an HTML sink (`innerHTML`/`document.write`/`eval` — asserted absent by `TestLandingThemeScript`). Only `setAttribute` + CSS custom-property writes from a hardcoded data table. **Confidence of any XSS path < 8 — suppressed.**

### CWE-639 / auth — not applicable
No auth boundary touched. Landing page is public; the script adds no privileged access, no new endpoint.

### CSP
`permissiveCSP` already permits `script-src 'self' 'unsafe-inline'` on `/` (`securityheaders.go:27`); the inline script adds no new CSP surface, no `unsafe-eval`. `frame-ancestors 'none'` unchanged.

## Findings
**No HIGH findings (confidence ≥ 8).** No EXCEPTIONS required.

## Verification
- `go test ./components/proxy/ -run 'TestLanding|TestAccentRampParity'` — PASS (5 tests, incl. no-HTML-sink invariant + enum validation).
- `golangci-lint run ./components/proxy/` — 0 issues.
- Threat model: LOW risk.

---

# Security Review — WCAG 2.2 AAA batch (21 defects + e86 + e87)

**Date:** 2026-08-11
**Branch/Diff:** main vs origin/main (40 commits: WCAG audit registration, 21 a11y defect fixes, e86 monitoring logs, e87 AAA conformance)
**Threat Models:** `specs/security/epics/e86/THREAT_MODEL.md` (MEDIUM — monitoring logs org scoping), `specs/security/epics/e87/THREAT_MODEL.md` (LOW; s05 auth MEDIUM)

## Scope Resolution

- **WCAG 21-defect batch (T000001–T000021):** UI/CSS/TSX only — ARIA, focus, contrast, labels. No auth/network boundary except T000019 (document.title) and the axe e2e harness.
- **e86 (monitoring logs):** new SSE endpoint + `org_id` column + deploy-event ingestion. Reviewed in e86 threat model: org isolation via `kernel.OrgIDFromContext` (BUG-143 pattern), SSE subscriber org gating, parameterized queries. No HIGH findings.
- **e87 (AAA):** token contrast, focus rings, target sizes, abbr/glossary, session-timeout warning, help/error-prevention, rem scaling. Only auth-touching story is e87s05 (JWT exp client decode + refresh + sessionStorage route restore): exp decode is public signed data; refresh reuses the existing validated endpoint; sessionStorage holds pathname+search only, never tokens; cleared on read. CWE-613/384/201/79 reviewed — mitigated (see e87 THREAT_MODEL §3).

## Findings

**No HIGH findings (confidence ≥ 8).** No EXCEPTIONS required. e86/e87 threat models carry LOW/MEDIUM residual risk with mitigations recorded.

## Verification

- `bash scripts/check-coverage.sh` — 65.0% PASS (repo gate 60%)
- UI: 707/707 tests, build green, lint clean, axe 17/17 routes 0 violations (wcag2a+aa+aaa tags)
- Token lifecycle E2E 4/4 (auth refresh paths)
