# e88s01 — Accent-theme light-mode link contrast ≥7:1 (1.4.6)

**type:** feat
**risk:** P1
**context:** domain
**BCPs:** 3

## Summary

The last 1.4.6 gap: 13 accent themes' light-mode link color (per-theme `brand600` used for `--fg-accent` in light mode) runs **4.23–7.90:1** on white — several themes fail AAA. Add a per-theme `brandLink` step (darker than brand600, ≈ Tailwind 800-level) verified ≥7:1 on white, and use it for light-mode `--fg-accent`.

## Context

`ui/src/context/accentThemes.ts` — each `AccentTheme` has `brand500/600/700/300` (RGB comma-strings). e87 set light `--fg-accent: var(--brand-600)` and dark `--fg-accent: var(--brand-300)` (dark links verified ≥7:1 via ThemeContext.applyAccentToDocument). Known failing light values (computed): july 251,146,60 (~3.1:1), august 107,114,128 (~4.6:1), september 202,138,4 (~4.2:1), may 139,92,246 (~4.5:1), june/december 67,56,202 / 185,28,28 (pass), others mixed.

## Requirements

#### ADDED: Per-theme AAA link step (1.4.6)
Each `AccentTheme` gains a `brandLink` field — an RGB step darker than `brand600` — verified ≥7:1 against white. Light-mode `--fg-accent` uses `brandLink`; dark-mode keeps `brand300` (already ≥7:1). Theme switching, hover/active states, and the rainbow (june) gradient keep working.

## Implementation Steps

1. Compute `brandLink` per theme: start from brand700 and darken until ≥7:1 on white (Tailwind 800-level values are the target; verify each with the luminance formula). Record the 13 values + ratios in the spec. → verify: `node -e "<luminance check: assert all 13 brandLink >=7:1 on white>"`
2. Add `brandLink: '<r, g, b>'` to each `AccentTheme` in accentThemes.ts. → verify: `grep -c "brandLink" ui/src/context/accentThemes.ts` (13)
3. ThemeContext: light-mode `--fg-accent` → per-theme `brandLink` (default theme = indigo brandLink); keep dark `brand300`. Update `applyAccentToDocument` + `accentThemes.ts` types. → verify: `cd ui && npm run build`
4. Update/extend `e87s01-contrast-matrix.mjs` (in specs/epics/archive/e87-wcag-aaa/) or a new e88 matrix script covering default + all 13 themes × light link on white + dark link on neutral-850, all ≥7:1. → verify: `node specs/epics/e88-aaa-closing/e88s01-contrast-matrix.mjs` prints 14/14 themes PASS
5. Tests: ThemeContext.test.tsx — update october `--fg-accent` expectation to the new brandLink value; add a per-theme ≥7:1 assertion loop. → verify: `cd ui && npx vitest run src/context/ThemeContext.test.tsx`

## Risks

- Darkening link colors shifts the brand feel for light themes (links become darker/less saturated) — the tradeoff inherent to AAA 7:1; keep hue, adjust lightness.
- Rainbow (june) theme: links follow the default indigo brandLink; the gradient applies to buttons only, not links — no change needed.
- ThemeContext test expectations hardcode colors — update with the new values (B1's e87 note flagged october specifically).

## Acceptance Criteria

- [ ] All 13 themes have brandLink ≥7:1 on white (matrix PASS)
- [ ] Light links use brandLink; dark links unchanged (brand300)
- [ ] ThemeContext tests green; `npm run build` green
