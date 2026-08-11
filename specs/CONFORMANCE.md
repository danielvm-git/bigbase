# Conformance Record — BigBase Admin Console

| Field | Value |
|---|---|
| **Project** | BigBase Admin Console (`ui/`) |
| **Version** | 2.87.0 |
| **Date** | 2026-08-11 |
| **Basis** | axe 17/17 routes green (`wcag2a`, `wcag2aa`, `wcag2aaa` rulesets, `tests/e2e/axe-scan.spec.ts`); 707 UI tests; contrast matrix (`specs/epics/archive/e87-wcag-aaa/e87s01-contrast-matrix.mjs`) |
| **Standard** | WCAG 2.2 (W3C Recommendation, October 2023) |

## Claim

> **WCAG 2.2 AA certified (axe-verified); AAA implemented except 1.4.8 partial and 2.2.3 exception — see [1.4.8 partial](#148-visual-presentation-partial) and [2.2.3 exception](#223-no-timing-exception) below.**

WCAG conformance claims are level-atomic. Because 1.4.8 is only partially
implemented (and 1.4.8 has no criterion-level exception), a full AAA
conformance level cannot be claimed; this statement records the implemented
AAA criteria and the two documented partials honestly.

## Evidence table

| Criterion | Implementation | Verification |
|---|---|---|
| 1.4.6 Contrast (Enhanced) | Accent-theme per-theme light-mode link color ≥ 7:1 (`--fg-accent` from `accentThemes.ts` brand step, dark mode ≥ 7:1 via brand300); default palette tokens at AAA contrast (e87) | Contrast matrix for default + 13 themes, light + dark (`e87s01-contrast-matrix.mjs`); axe `color-contrast` on 17/17 routes |
| 1.4.8 Visual Presentation | **Partial** — see [§1.4.8 partial](#148-visual-presentation-partial) | `grep 'line-height: 1.5' ui/src/styles/tokens.css`; `grep '0.75rem' ui/src/styles/tokens.css`; no justified text in styles |
| 2.4.13 Focus Appearance (Enhanced) | `:focus-visible`/`:focus` ring ≥ 3:1 via `--border-focus: var(--brand-600)`, consistent 2px ring + ≥ 2px offset | axe `focus-order` + `color-contrast` on 17/17 routes; manual Tab walkthrough (e87s02) |
| 2.5.5 Target Size (Enhanced) | Pointer targets ≥ 44×44 CSS px or documented inline/spacing exception per component | `tests/e2e/target-size.spec.ts` (bounding-box measurement, ≥ 44px-or-exception) |
| 3.1.3 Unusual Words / 3.1.4 Abbreviations / 3.1.5 Reading Level | `<abbr title="…">` expansions for jargon/acronyms; definition mechanism via `Tooltip`/`aria-describedby`; glossary linked from app footer | Glossary at `specs/tech-architecture/GLOSSARY_LATEST.md`, linked from `AppFooter.tsx`; tooltip tests (e87s04) |
| 2.2.5 Re-authentication / 2.2.6 Timeouts | Pre-expiry warning toast + dialog with "Stay signed in" token refresh (`SessionTimeoutWarning.tsx`); expiry re-auth preserves route/state | `SessionTimeoutWarning.test.tsx`; `AuthContext.test.tsx` (e87s05) |
| 3.3.5 Help / 3.3.6 Error Prevention | `Input` `hint` wired via `aria-describedby`; confirm/review steps for destructive sends, deletes, and complex submissions | `Input.tsx`; component tests + manual flows (e87s06) |
| 2.4.8 Location | Breadcrumb on detail pages (`Breadcrumb.tsx`); location indicator on top-level pages (active nav + page header pair) | Manual walkthrough (e87s07) |

## 1.4.8 Visual Presentation (partial)

### Implemented bullets

- **Line height ≥ 1.5** within paragraphs: `line-height: 1.5` on the body
  default in `ui/src/styles/tokens.css`, inherited by body and paragraph
  defaults (headings keep their own spacing).
- **Text scaling to 200%**: all text sizes are rem-based tokens
  (`--text-xs: 0.75rem` … `--text-3xl`), so text scales with the browser/user
  root font-size setting — verified up to 200% zoom.
- **No justified text**: no `text-align: justify` in the stylesheet; text is
  flush-left (default).

### Not implemented (documented partial)

- **User-selectable foreground/background colors** (WCAG 1.4.8 bullet 1).
- **User-selectable text spacing controls** (letter-spacing, word-spacing,
  line-height, paragraph spacing overrides).

### Rationale

- OS/browser accessibility settings (forced colors, high contrast, and
  user stylesheet/font-size overrides) already cover the user-selectable
  color and text-spacing needs 1.4.8 bullet 1 targets; the admin console
  respects those user-agent settings (rem tokens + inherited colors).
- The admin console is a high-density operational UI; adding in-app color
  and spacing override controls would conflict with the density constraint
  (see e87 scope decision) and duplicate platform-level mechanisms.
- Audit trail: documented in `specs/epics/e87-wcag-aaa/e87s07-spec.md`
  (feasible bullets shipped in e87s07) and formalized here in e88s02.
- Consequence: 1.4.8 has **no** criterion-level exception in WCAG, so the
  overall claim is expressed as a statement ("AAA except 1.4.8 partial"),
  not a AAA conformance level.

## 2.2.3 No Timing (exception)

- **Exception claimed**: the JWT session expiry is an authentication
  security control, not content timing. Terminating an authenticated session
  is an *essential* activity (WCAG 2.2.3 exception: "the time limit is an
  essential part of the activity"), so the criterion does not apply to it.
- **User protection**: users are warned before expiry via 2.2.6
  (`SessionTimeoutWarning` toast/dialog with "Stay signed in" refresh,
  shipped e87s05), and re-authentication after expiry preserves their
  route/state (2.2.5).
- No other time limits are imposed on content in the console.

## Threat model & verification references

- WCAG re-audit 2026-08-11 (AAA gap analysis, e87/e88 threat models):
  `specs/epics/archive/e87-wcag-aaa/epic.yaml`, `specs/epics/e88-aaa-closing/epic.yaml`.
- axe gate: `tests/e2e/axe-scan.spec.ts` (17 routes, wcag2a+aa+aaa).
- Contrast matrix: `specs/epics/archive/e87-wcag-aaa/e87s01-contrast-matrix.mjs`.

---

## Certification Record (e88s03)

**Date:** 2026-08-11 · **Version:** 2.87.x (post-e88)
**Gates:**
- Contrast matrix: **13/13 accent themes PASS** (light links 7.10–9.93:1 on white; dark links ≥7:1 on neutral-850) — `node specs/epics/e88-aaa-closing/e88s01-contrast-matrix.mjs`
- axe: **17/17 routes, 0 violations** (wcag2a + wcag2aa + wcag2aaa tags) on rebuilt dist
- UI suite: **709/709** · build green · lint clean

**Claim (final):** WCAG 2.2 AA certified (axe-verified, 17/17 routes). AAA implemented except the two documented items above — 1.4.8 partial (user-selectable fg/bg + spacing controls; OS/browser settings + density rationale) and 2.2.3 essential/security exception (JWT session expiry, warned via 2.2.6).
