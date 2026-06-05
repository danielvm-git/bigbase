# BigBase Design System — React stubs

Production-shaped React + TypeScript component stubs for the BigBase Admin Console.
They are **thin, typed wrappers** over the design-system CSS (`colors_and_type.css` +
`components.css`) — all visual states live in CSS; these components compose class
names and wire behavior, theming, and accessibility.

## Install / wire up

```tsx
import '@bigbase/ds/styles.css';            // colors_and_type.css + components.css
import { ThemeProvider, Button } from '@bigbase/ds';

export default function App() {
  return (
    <ThemeProvider>
      <Button variant="primary">Create site</Button>
    </ThemeProvider>
  );
}
```

## Files

| File | What it is |
|------|------------|
| `tokens.ts` | Token type unions (`AccentTheme`, `BadgeVariant`, `ButtonVariant`, spacing/radius/shadow) + `THEME_META` for the 12 month themes |
| `ThemeContext.tsx` | `ThemeProvider` + `useTheme()` — writes `data-accent`/`data-theme` to `<html>`, persists to `localStorage` (`bigbase-theme`, `bigbase-dark`) |
| `Button.tsx` | `Button` — 5 variants × 3 sizes, `loading`, `icon`, dev-time icon-only a11y warning |
| `Badge.tsx` | `Badge` (6 variants) + `StatusBadge` (word + dot/spinner + color, never color-alone) |
| `Input.tsx` | `Input` — label, inline `error` (aria-invalid + role="alert"), `hint`, `prefix`, `mono` |
| `Card.tsx` | `Card` (static / `interactive`) + `EmptyState` (icon chip, title, payoff, one CTA) |
| `ThemePicker.tsx` | Sidebar-footer accent-theme dropdown (listbox/option roles, Esc + click-outside) |
| `index.ts` | Barrel — re-exports every component + its props type |
| `Components.stories.tsx` | Examples covering each state (default/error/disabled/loading/empty); also valid Storybook CSF |

## Conventions

- **Theme context:** every component reads CSS role tokens, so changing the accent or
  scheme via `useTheme()` re-themes the whole tree instantly — no prop drilling, no
  per-component overrides. Brand/color overrides happen at the token layer.
- **Props interfaces:** every component exports a named `*Props` interface and extends
  the matching DOM attributes (`ButtonHTMLAttributes`, etc.) so native props pass through.
- **Accessibility:** icon-only `Button`s require `aria-label` (warned in dev); `Input`
  wires `aria-invalid` + `aria-describedby`; `StatusBadge` never relies on color alone.
- **Production icons:** examples use Lucide. Swap the placeholder nodes for
  `lucide-react` imports (`import { Plus } from 'lucide-react'`).

> These are **stubs for handoff** — typed contracts + behavior, ready to drop into the
> React 19 + Vite app. They intentionally contain no bespoke styling; the CSS is the
> single source of truth.
