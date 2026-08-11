# Threat Model — e85: Unified Theming (Landing Page Parity)

**Risk level: LOW**
**Generated:** build-epic Step 0 (security-review)

## Scope

Add a blocking inline `<head>` script to the server-rendered landing page
(`homeTemplate`, `components/proxy/proxy.go`) that reads the admin console's
`localStorage` keys (`bigbase-theme`, `bigbase-accent`) and applies light/dark
+ 13 accent colors via `data-theme` attribute + brand CSS custom properties.
Refactor landing CSS dark overrides to `[data-theme="dark"]` with a no-JS
`prefers-color-scheme` fallback. No backend, no DB, no auth, no new endpoint.

## Surface area

| Asset | Exposure |
|-------|----------|
| Landing page HTML (`/`) | Public, unauthenticated, served by proxy |
| `localStorage` (`bigbase-theme`, `bigbase-accent`) | Same-origin, shared with `/admin/` |
| Inline `<script>` in `<head>` | Runs before paint on every landing load |

## Vulnerability assessment

### CWE-79 — XSS via inline script
**Verdict: NOT APPLICABLE (mitigated).** The script reads `localStorage` values
but never injects them as HTML/markup. Both inputs are validated before use:
- `bigbase-theme` accepted only if exactly `'light'` or `'dark'`; otherwise
  derived from `matchMedia`.
- `bigbase-accent` accepted only if it is a key of the hardcoded `ACCENTS`
  object (13 known ids); otherwise defaults to `'default'`.
Only the validated enum value reaches the DOM, and only as:
1. `document.documentElement.setAttribute('data-theme', t)` — `t` ∈ {light, dark}.
2. `setAttribute('data-accent-rainbow', 'true')` — boolean literal.
3. `style.setProperty('--brand-500', 'rgb('+v.b+')')` etc. — `v.b/c/d` are
   hardcoded RGB triples from the `ACCENTS` table, never user input.

No `innerHTML`, no `document.write`, no eval, no template interpolation of
untrusted data. **Confidence of any XSS path: < 8 — suppressed.**

### CWE-79 — Stored XSS via `localStorage`
**Verdict: low / out of scope.** A malicious `localStorage` value written by a
third-party same-origin script could be read here, but (per above) it is
enum-validated and never rendered as markup. The admin writer
(`ui/src/context/ThemeContext.tsx`) only ever stores enum values. No escalation.

### CSP considerations
`permissiveCSP` (applied to all non-API routes incl. `/`) already permits
`script-src 'self' 'unsafe-inline'` (`securityheaders.go:27`). The inline script
adds **no new CSP surface** — inline scripts were already allowed on this route
(docs page uses HTMX inline handlers). No `unsafe-eval`. `frame-ancestors 'none'`
unchanged.

### Data drift (accent ramp duplication)
**Verdict: low / quality, not security.** The accent RGB table is duplicated
between `ui/src/context/accentThemes.ts` (TS) and `landingAccents` (Go). Drift
would cause a visual mismatch, not a vulnerability. Mitigated by
`TestAccentRampParity` (e85s02).

## Mitigations enforced by this epic
1. Enum validation on both `localStorage` reads (theme + accent) before any DOM write.
2. No untrusted string ever reaches an HTML sink — only attributes and CSS custom properties.
3. No new CSP permissions introduced.
4. Parity test prevents silent accent-ramp drift between surfaces.

## Risk reduction / opportunity
Unifies two divergent theme systems (reduces maintenance/architecture drift);
no new attack surface.

## Conclusion
**No HIGH findings (confidence ≥ 8).** Net change to XSS/auth/crypto surface:
none. The only DOM writes are enum-validated attribute/CSS-property sets from a
hardcoded data table. Proceed to implementation without a grill-me gate
(assess-impact risk score well below 7 — net-new client-side code, no existing
dependents on the new script).
