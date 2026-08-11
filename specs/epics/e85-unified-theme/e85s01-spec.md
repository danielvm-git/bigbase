# e85s01 — Landing page theme bootstrap

## Summary

The landing page (`homeTemplate` in `components/proxy/proxy.go`) is a zero-JS,
server-rendered HTML page. Dark mode is driven purely by
`@media (prefers-color-scheme: dark)` and the brand color is a hardcoded indigo
literal — it ignores the React admin console's theme/accent selection entirely.

Give the landing page **full theme parity** with the admin: light/dark **and**
the 13 accent colors. Because both surfaces are same-origin (`bigbase.click`),
they already share `localStorage`. Add a blocking inline `<head>` script that
reads the admin's existing keys (`bigbase-theme`, `bigbase-accent`), resolves a
final theme (localStorage → `prefers-color-scheme` fallback for first-time
visitors), and applies it before paint by mirroring the admin's
`applyAccentToDocument` logic. Refactor the landing CSS to key dark overrides
off `[data-theme="dark"]` while preserving the OS-driven behavior for no-JS
visitors.

No backend changes, no new endpoint, no DB migration. The landing route's CSP
already permits `unsafe-inline` for scripts/styles
(`components/proxy/securityheaders.go:20-23`), so the inline script is allowed.

## Requirement delta

- **ADDED**: Landing page respects the admin console's light/dark choice.
- **ADDED**: Landing page respects the admin console's 13-accent selection.
- **MODIFIED**: Landing page dark-mode trigger.
  - before: `@media (prefers-color-scheme: dark) { :root { … } }`
  - after:  `[data-theme="dark"] { … }` + `:root:not([data-theme])` media-query
    fallback (preserves current OS-driven behavior when JS is disabled).

## Changes

### `components/proxy/proxy.go` — `homeTemplate` `<head>`

Add a synchronous inline `<script>` (before `<style>` so it sets the attribute
before first paint; must run before body renders to avoid FOUC):

1. Embed an `ACCENTS` object literal — the 13 entries from
   `ui/src/context/accentThemes.ts` (`id`, `brand500`, `brand600`, `brand700`,
   `rainbow?`). Use the same `r, g, b` string format the admin uses.
2. Resolve theme:
   - `t = localStorage.getItem('bigbase-theme')`; accept only `'light'|'dark'`.
   - If absent, derive from `window.matchMedia('(prefers-color-scheme: dark)').matches`.
3. Resolve accent:
   - `a = localStorage.getItem('bigbase-accent')`; accept only a key present in
     `ACCENTS`; else `'default'`.
4. `document.documentElement.setAttribute('data-theme', t)`.
5. Apply brand custom props on `documentElement.style` — the **same set and
   formulas** as `ui/src/context/ThemeContext.tsx::applyAccentToDocument`:
   `--brand-500/600/700`, `--border-accent`, `--bg-accent`,
   `--bg-accent-hover`, `--bg-accent-active`, `--fg-accent`, `--brand-tint`,
   `--focus-ring`.
6. Rainbow: if `ACCENTS[a].rainbow`, set `data-accent-rainbow="true"`; else
   remove the attribute.

### `components/proxy/proxy.go` — `<style>` block

- Replace the two `@media (prefers-color-scheme: dark) { :root { … } }` blocks
  (lines ~605 and ~965) with `[data-theme="dark"] { … }` selectors carrying the
  identical dark token values.
- Add a no-JS fallback so first-time dark-OS visitors without JS still get dark:
  ```css
  @media (prefers-color-scheme: dark) {
    :root:not([data-theme]) { /* same dark values as [data-theme="dark"] */ }
  }
  ```
  Rationale: when JS runs it always sets `data-theme`, so `:not([data-theme])`
  excludes JS users and prevents a light-chose/light-OS user from inheriting the
  dark media query. No-JS users never get the attribute, so the media query
  applies.
- Add a rainbow CTA treatment to match the admin's rainbow button fill, e.g.
  `[data-accent-rainbow="true"] .hero-cta-primary,
  [data-accent-rainbow="true"] .nav-cta { background: <gradient>; }`.
- Keep `--brand-500: #4f46e5` etc. as the `:root` defaults — they remain the
  fallback for no-JS / `default` accent; the inline script overrides them at
  runtime.

## Acceptance criteria (Gherkin)

```gherkin
Scenario: Landing page reflects admin dark + accent choice
  Given the admin console has set localStorage bigbase-theme="dark" and bigbase-accent="march"
  When the user navigates to "/" (landing page)
  Then <html> has attribute data-theme="dark"
  And the computed --brand-500 on <html> equals "rgb(124, 58, 237)" (March — Purple)
  And all elements using var(--brand-500) — hero CTA, nav CTA, links, mockup stats — render purple

Scenario: First-time visitor follows OS preference
  Given no bigbase-theme in localStorage and the OS is in dark mode
  When the landing page loads with JS enabled
  Then <html data-theme="dark"> is set via the matchMedia fallback
  And the default indigo accent is applied

Scenario: First-time light-OS visitor
  Given no bigbase-theme in localStorage and the OS is in light mode
  When the landing page loads
  Then <html data-theme="light"> and the default indigo accent are applied

Scenario: No-JS dark-OS visitor still gets dark mode
  Given JavaScript is disabled and the OS is in dark mode
  When the landing page loads
  Then the :root:not([data-theme]) media-query fallback renders dark mode

Scenario: Rainbow accent on the landing page
  Given bigbase-accent="june" in localStorage
  When the landing page loads
  Then <html> has attribute data-accent-rainbow="true"
  And primary CTAs display the rainbow gradient

Scenario: No flash of un-themed content
  Given bigbase-theme="dark" and bigbase-accent="october"
  When the landing page loads
  Then the dark background and pink brand are present on first paint (no light→dark flash)
```

## Verify
```bash
go test ./components/proxy/... -run TestLandingTheme -v
```
- Render `homeTemplate` output, inject `<script>`-simulated localStorage, assert
  `data-theme` attribute + brand custom props for: dark+march, light+default,
  june (rainbow), and the no-attribute (no-JS) path.
