# e51s01: Design Tokens & Theme System

**Story ID:** e51s01 | **Epic:** e51 | **BCPs:** 4 | **Status:** planned
**Type:** feat | **Context:** domain
**Depends on:** none | **Blocks:** e51s02, e51s03, e51s04

## §1 — Summary

Formalize the existing CSS-custom-property design token system with TypeScript
constants for type safety and IDE autocomplete. Add `prefers-reduced-motion`
OS-level respect. Add missing semantic tokens for surface elevation tiers,
interaction states (disabled, readonly), and common layout values. Write
contract tests proving CSS vars match the TS constants — catching drift between
`tokens.css` and the component code before it causes visual regressions.

## §2 — Motivation

The current token system (`ui/src/styles/tokens.css` + `theme.css`) works but
has no type safety. Components reference CSS vars by string, with no autocomplete
or compile-time verification. Adding TS token constants gives designers and
developers a single source of truth, makes refactoring tokens safe (rename
symbol → all usages update), and enables IDE autocomplete for design values.

## §3 — Background / Context

- Existing: `ui/src/styles/tokens.css` (light mode + base tokens)
- Existing: `ui/src/styles/theme.css` (dark mode overrides)
- Existing: `ui/src/context/ThemeContext.tsx` (light/dark toggle + accent picker)
- Existing: `ui/src/context/accentThemes.ts` (5 accent color presets + rainbow)
- Existing: `ui/src/index.css` (globals import tokens.css and theme.css)
- Token source: `specs/epics/e17-enhanced-admin-ui/SYSTEM_DESIGN.md`
- Stack: React 19, TypeScript 6, Vite 8, Vitest 4

## §4 — Zoom-Out Check

- **Module purpose**: Admin console SPA — UX layer for the BigBase platform
- **Callers**: Browser users (all admin console traffic), Go admin component via `//go:embed`
- **Contracts**: CSS custom properties defined in `:root` / `[data-theme="dark"]`; `ThemeContext` React context; `accentThemes.ts` provides `AccentId` type and `getAccentTheme()` function

## §5 — Prior Art

- The token system was designed in e17 and follows the Appwrite design system conventions
- CSS custom properties are the standard approach for theming in modern SPAs (vs. CSS-in-JS runtime)
- Recharts/chart components already consume these tokens via CSS variables

## §6 — Design Decisions

| Decision | Rationale |
|----------|-----------|
| TS constants as `as const` objects | Maximum type narrowing, zero runtime cost |
| Keep CSS custom properties as the runtime source | Existing code consumes CSS vars; breaking that is out of scope |
| Contract test: read computed styles from DOM | Proves TS ↔ CSS alignment without mocking getComputedStyle |
| Add motion tokens + prefers-reduced-motion | WCAG 2.2 SC 2.3.3 (Animation from Interactions) |

## §7 — Architecture / Component Design

```
ui/src/styles/
  tokens.css          ← CSS custom properties (unchanged structure)
  theme.css           ← dark mode overrides (unchanged)
ui/src/tokens/
  tokens.ts           ← NEW: TS constants mirroring CSS vars
  tokens.test.ts      ← NEW: contract tests
  motion.ts           ← NEW: motion tokens + reduced-motion hook
  motion.test.ts      ← NEW: motion hook tests
  types.ts            ← NEW: shared token types
```

## §8 — Data Model / Types

```typescript
// ui/src/tokens/types.ts
export interface DesignTokens {
  colors: {
    neutral: Record<string, string>  // neutral-0 through neutral-900
    brand: Record<string, string>    // brand-50 through brand-700
    semantic: {
      success: string; successBg: string; successFg: string
      warning: string; warningBg: string; warningFg: string
      error: string; errorBg: string; errorFg: string
      info: string; infoBg: string; infoFg: string
    }
    background: Record<string, string>  // bg-default, bg-surface, etc.
    foreground: Record<string, string>  // fg-primary, fg-secondary, etc.
    border: Record<string, string>
    overlay: Record<string, string>
  }
  typography: {
    fontSans: string; fontMono: string
    sizes: Record<string, string>   // text-xs through text-3xl
  }
  spacing: Record<string, string>   // space-0 through space-24
  radii: Record<string, string>     // radius-xs through radius-full
  shadows: Record<string, string>   // shadow-xs through shadow-xl
  motion: {
    durations: Record<string, string>  // duration-fast, etc.
    easings: Record<string, string>    // ease-standard, etc.
  }
}
```

## §9 — API / Interface Contract

**tokens.ts** exports:
- `TOKENS: Readonly<DesignTokens>` — the canonical token object
- `token(path: string): string` — lookup helper, e.g. `token('colors.semantic.success')`
- `cssVar(name: string): string` — returns `var(--name)` for inline style use

**motion.ts** exports:
- `usePrefersReducedMotion(): boolean` — hook wrapping `matchMedia('(prefers-reduced-motion: reduce)')`
- `MOTION_TOKENS` — durations/easings constants

## §10 — State Management

No state changes. `usePrefersReducedMotion` is a read-only reactive hook.

## §11 — Error Handling

N/A — pure data module, no async ops.

## §12 — Testing Strategy

| Test type | What | Tool |
|-----------|------|------|
| Contract tests | CSS custom properties in DOM = TS constants | vitest + jsdom |
| Motion hook | `usePrefersReducedMotion` returns true/false based on media query | vitest + jsdom `matchMedia` mock |
| Type tests | `expectTypeOf` for token shape | vitest |

## §13 — Performance Considerations

- TS constants are compile-time, zero runtime cost
- `usePrefersReducedMotion` uses `matchMedia` with passive listener — negligible overhead

## §14 — Security Considerations

N/A — no user data, no network calls.

## §15 — Accessibility

The `prefers-reduced-motion` hook is the accessibility deliverable. Components
built on e51s02+ will consume this hook to disable animations when the user's
OS preference is set.

## §16 — Internationalization (i18n)

N/A — design tokens have no translatable content.

## §17 — Acceptance Criteria (Gherkin)

```gherkin
Scenario: Token constants match CSS custom properties
  Given the design tokens TS module defines all token values
  When the contract test reads computed styles from the DOM
  Then every TS constant matches its corresponding CSS custom property exactly

Scenario: Reduced motion preference is detected
  Given the user's OS has "Reduce motion" enabled
  When usePrefersReducedMotion() is called
  Then it returns true

Scenario: Reduced motion preference change is reactive
  Given usePrefersReducedMotion() returns false
  When the user changes their OS motion preference to "Reduce"
  Then the hook re-renders with true

Scenario: Token lookup helper works
  Given TOKENS.colors.neutral['900'] is 'rgba(25, 25, 28, 1)'
  When token('colors.neutral.900') is called
  Then it returns the same value
```

## §18 — Verification Script (Step-by-Step)

1. Run contract tests: `cd ui && npx vitest run src/tokens/`
2. Verify all token CSS vars exist in DOM: `cd ui && npx vitest run src/tokens/tokens.test.ts`
3. Verify motion hook works: `cd ui && npx vitest run src/tokens/motion.test.ts`
4. Verify full UI still builds: `cd ui && npm run build`
5. Verify Go embed still works: `cd .. && go build ./...`

## §19 — Out of Scope

- Removing or renaming existing CSS custom properties (backward compat)
- CSS-in-JS migration
- Design token documentation website
- Figma plugin or design tool integration

## §20 — Risks

| Risk | Mitigation |
|------|-----------|
| Token drift: TS constants get out of sync with CSS | Contract test runs in CI on every push |
| Motion hook performance with many listeners | Single matchMedia listener per app (hook uses context internally if needed) |
| Token object too large for tree-shaking | Use `as const` + direct property access; no tree-shaking issue for constants |
