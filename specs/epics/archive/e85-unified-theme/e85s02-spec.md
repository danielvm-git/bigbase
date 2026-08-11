# e85s02 — Accent ramp drift guard + cross-surface parity test

## Summary

e85s01 duplicates the 13-accent ramp: the canonical source lives in
`ui/src/context/accentThemes.ts` (TypeScript), and a JS object literal copy is
embedded in the landing page's `homeTemplate` (Go string). Without a guard the
two will silently drift — a new accent added to the admin won't appear on the
landing page, or a tweaked RGB will produce a mismatched brand between surfaces.

Add two guards:
1. A Go test that asserts the embedded landing-page ramp equals the canonical
   TypeScript source (parses `accentThemes.ts`, compares id + brand500/600/700
   + rainbow for all 13 entries).
2. A Playwright cross-surface test: set a theme in the admin, navigate to the
   landing page, assert the same theme is applied.

## Requirement delta

- **ADDED**: Automated drift guard preventing accent-ramp divergence between the
  admin (`ui/src/context/accentThemes.ts`) and the landing page
  (`homeTemplate`).
- **ADDED**: End-to-end proof that a theme chosen in `/admin/` is honored on `/`.

## Changes

### `components/proxy/landing_theme_test.go` (extend, or new file)

- `TestAccentRampParity`: read `ui/src/context/accentThemes.ts`, extract the
  `ACCENT_THEMES` entries (regex over the `id:`/`brand500:`/`brand600:`/
  `brand700:`/`rainbow:` lines), and the `ACCENTS` literal in `homeTemplate`;
  assert the two sets are equal (same ids, same RGB triples, same rainbow
  flags). Fail with a diff if either side changes without the other.
- Reuse the canonical extraction so adding/editing an accent in one place
  without the other is a red test.

### `tests/e2e/theme-parity.spec.ts` (new)

- Navigate to `/admin/`, open the sidebar appearance controls, toggle dark mode,
  pick a non-default accent (e.g. October — Pink) via the `ThemePicker`.
- `page.goto('/')` to the landing page.
- Assert `document.documentElement.getAttribute('data-theme') === 'dark'`.
- Assert the computed `--brand-500` on `<html>` equals the selected accent's
  `rgb(...)` value.
- Repeat for one light + default case and the June rainbow case.

## Acceptance criteria (Gherkin)

```gherkin
Scenario: Accent ramp parity test catches drift
  Given ui/src/context/accentThemes.ts and the landing-page ACCENTS literal
  When TestAccentRampParity runs
  Then it passes only if all 13 ids, brand500/600/700 values, and rainbow flags match exactly
  And it fails with a clear diff if a value is changed on one side only

Scenario: Theme chosen in admin is honored on the landing page
  Given the user selects dark mode and "October — Pink" in /admin/
  When the browser navigates to "/"
  Then <html> has data-theme="dark"
  And the computed --brand-500 equals "rgb(236, 72, 153)"

Scenario: Rainbow selection propagates to the landing page
  Given the user selects the June accent in /admin/
  When the browser navigates to "/"
  Then <html> has data-accent-rainbow="true"
```

## Verify
```bash
go test ./components/proxy/... -run TestAccentRampParity -v
cd tests/e2e && npx playwright test theme-parity
```
