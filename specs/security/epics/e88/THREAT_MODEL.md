# Threat Model: e88 — WCAG AAA Closing

**Date**: 2026-08-11
**Epic**: e88 — WCAG AAA Closing (accent-theme link contrast + formal conformance)
**Nature**: UI data-table change (13 accent theme colors) + documentation. No new endpoints, no auth/network surface, no new data.
**Risk Level**: LOW

## 1. Attack Surface

| Story | Surface | New/Existing | Security Risk |
|-------|---------|--------------|---------------|
| e88s01 | `accentThemes.ts` per-theme `brandLink` colors + ThemeContext light-mode `--fg-accent` | Modified (UI data/CSS var) | NONE |
| e88s02 | `specs/CONFORMANCE.md` claim document | New (docs) | NONE (reputational only — overclaim risk) |
| e88s03 | Re-certification (matrix + axe + release) | Verification | NONE |

## 2. Trust Boundaries

None crossed. `brandLink` values are hardcoded developer-authored RGB strings applied via the existing `applyAccentToDocument` path (enum-validated AccentId → hardcoded table, no user input reaches a sink — the same invariant e85's review asserted for the landing theme script).

## 3. Vulnerabilities & Mitigations

| # | Category | Finding | Mitigation | Verify |
|---|----------|---------|-----------|--------|
| 1 | N/A (XSS) | brandLink string could in principle reach a style sink | Values are compile-time literals from a typed table; only setAttribute + CSS custom-property writes (existing invariant) | e88s01t2/3 |
| 2 | Reputational overclaim | CONFORMANCE.md could overstate conformance | e88s02 spec mandates honest wording ("AA certified; AAA except documented partials"); 1.4.8 has no WCAG exception, so no level claim | e88s02t2 |

## 4. Risk Assessment

| Dimension | Rating | Rationale |
|-----------|--------|-----------|
| Production Risk | LOW | Color values + docs only; no behavior/network change |
| Regression Risk | LOW | ThemeContext test expectations must track new values (single test file); matrix script is the guard |
| Reputational Risk | LOW | Wording is pinned by the spec; reviewable in diff |

## 5. Security Gate

- All stories `security: none` — UI data + docs, no new findings expected.
- Regression gate: `ui/src/context/ThemeContext.test.tsx` (updated expectations) + the 14/14 contrast matrix script (fails loudly if any theme misses 7:1).

## 6. CWE Mapping

1 → N/A · 2 → CWE-1021 (Improper Restriction of Rendered UI Layers) is not applicable — no overlay injection; overclaim is a process risk, not a CWE-classifiable code flaw.
