---
id: e17s12
title: Twelve accent themes
status: pending
wsjf: 2.4
tasks:
  - desc: Extend ThemeContext with accent id (default + 12 months) persisted to localStorage
    verify: "grep -q accent ui/src/context/ThemeContext.tsx ui/src/context/themeState.ts"
  - desc: Apply --brand-500/600/700 and accent-derived tokens on documentElement
    verify: "grep -q setProperty ui/src/context/ThemeContext.tsx || grep -q brand-500 ui/src/context/ThemeContext.tsx"
  - desc: Add accent selector in sidebar footer Appearance block
    verify: "grep -q accent ui/src/Layout.tsx"
  - desc: June rainbow theme uses gradient accent on primary buttons
    verify: "grep -qi rainbow ui/src/context/ThemeContext.tsx ui/src/styles/tokens.css ui/src/index.css"
  - desc: ThemeContext unit tests or build passes
    verify: "cd ui && (npx vitest run src/context/ThemeContext.test.tsx 2>/dev/null || npm run build)"

acceptance: |
  Given the console shell is loaded
  When the user selects an accent from the footer dropdown
  Then primary actions and --fg-accent update without reload
  And the choice persists across refresh via localStorage
  And light and dark mode both respect the selected accent

context: |
  Source: Component Spec - States.html accent dropdown; PROMPT_01_UPDATE_DESIGN_SYSTEM.md month RGB values.
  Storage: prefer `bigbase-accent` key separate from `bigbase-theme` (light/dark).
---
