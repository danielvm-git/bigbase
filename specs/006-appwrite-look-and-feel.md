# 006 — Port Appwrite Look & Feel to BigBase Admin UI

**status:** done

## Analysis Summary

| Aspect | Current (BigBase) | Target (Appwrite) |
|--------|-------------------|-------------------|
| Accent | Indigo `#4f46e5` | Pink `#FD366E` |
| Neutral scale | None (single `--border: #ddd`) | 0–900 gray scale |
| Design tokens | 8 flat vars | 60+ semantic role tokens |
| Typography | `system-ui` | Inter (UI) + Aeonik Pro (brand) |
| Spacing | Ad-hoc px values | 4px-base scale (base-0…base-48) |
| Motion | None | 5 durations + 4 easing curves |
| Border radius | Single `8px` | xs(4) / s(8) / m(12) / l(16) |
| Components | None (inline per page) | Button, Card, Input, Badge, Toast |
| Sidebar | Plain `<a>` links | Appwrite-style nav with icons |
| CSS approach | Single 743-line file | CSS variables + component classes |

## Plan

### Phase 1: Design Token Migration
**File:** `ui/src/index.css` (rewrite `:root` block)
- Replace existing 8 CSS vars with Appwrite's full design token system
- Keep BigBase's indigo accent (don't switch to pink — brand identity)
- Add neutral color scale (0–900 grays)
- Add semantic role tokens (bgcolor-*, fgcolor-*, border-*)
- Add spacing scale, motion tokens, border radius tokens
- Add Inter font via `@import` (or CDN link in `index.html`)
- Add dark mode support via `prefers-color-scheme`

### Phase 2: Shared UI Components
**New directory:** `ui/src/components/`
- `Button.tsx` — primary / secondary / danger variants, sizes, disabled
- `Card.tsx` — card container with optional header
- `Input.tsx` — text/email/password with label, error, focus ring
- `Badge.tsx` — status pill (ok/err/warn/neutral)
- `Tabs.tsx` — tab bar component (extracted from current inline patterns)
- `EmptyState.tsx` — "no data" placeholder
- `PageHeader.tsx` — consistent h1 + action button row

### Phase 3: Layout & Navigation
**Files:** `Layout.tsx`, `index.css`
- Redesign sidebar: Appwrite-style with logo, nav sections, divider
- Active nav state with accent indicator
- Consistent content area padding
- Smooth page transitions

### Phase 4: Page Migration
Update pages one by one to use shared components:
- LoginPage → Appwrite-style centered card
- DashboardPage → token-aware cards and stats
- DataStudioPage → consistent tables
- SQL Editor → monospace styling
- UsersPage → shared table + badges
- GitReposPage → shared forms + buttons
- DeployPage → status badges
- MessagingPage → tabs + forms
- StoragePage → upload form
- FunctionsPage → code editor styling
- ForgePage → board + cards
- CiciPage → log viewer
- MonitoringPage → alerts + metrics

### Phase 5: Polish
- Toast/notification system
- Loading skeletons instead of "Loading..." text
- Responsive breakpoints
- Focus-visible rings for accessibility

## Verification

After each phase:
```
cd ui && npm run build   # verify build succeeds
cd .. && go run . serve  # verify app works end-to-end
```

## Appwrite Design Token Reference

### Color Palette (adapted for BigBase indigo accent)
```css
:root {
  /* Neutral grays */
  --neutral-0: rgba(255,255,255,1);
  --neutral-25: rgba(250,250,251,1);
  --neutral-50: rgba(237,237,240,1);
  --neutral-100: rgba(228,228,231,1);
  --neutral-200: rgba(216,216,219,1);
  --neutral-300: rgba(173,173,176,1);
  --neutral-400: rgba(151,151,155,1);
  --neutral-500: rgba(129,129,134,1);
  --neutral-600: rgba(108,108,113,1);
  --neutral-700: rgba(86,86,92,1);
  --neutral-800: rgba(45,45,49,1);
  --neutral-900: rgba(25,25,28,1);

  /* Brand accent (indigo — BigBase identity) */
  --brand-500: rgba(79,70,229,1);
  --brand-600: rgba(67,56,202,1);
  --brand-700: rgba(55,48,163,1);

  /* Semantic */
  --success: rgba(16,185,129,1);
  --warning: rgba(245,158,11,1);
  --error: rgba(239,68,68,1);

  /* Semantic role mappings (light) */
  --bg-default: var(--neutral-25);
  --bg-surface: var(--neutral-0);
  --bg-surface-hover: var(--neutral-50);
  --bg-accent: var(--brand-500);
  --bg-accent-hover: var(--brand-600);
  --bg-accent-active: var(--brand-700);

  --fg-primary: var(--neutral-800);
  --fg-secondary: var(--neutral-600);
  --fg-tertiary: var(--neutral-400);
  --fg-accent: var(--brand-500);
  --fg-on-accent: var(--neutral-0);

  --border-default: var(--neutral-50);
  --border-strong: var(--neutral-200);
  --border-focus: var(--neutral-300);
  --border-accent: var(--brand-500);

  /* Spacing */
  --space-0: 0;   --space-1: 2px;  --space-2: 4px;
  --space-3: 6px;  --space-4: 8px;  --space-5: 10px;
  --space-6: 12px; --space-8: 16px; --space-10: 20px;
  --space-12: 24px; --space-16: 32px; --space-20: 40px;

  /* Radius */
  --radius-xs: 4px;
  --radius-s: 8px;
  --radius-m: 12px;
  --radius-l: 16px;

  /* Motion */
  --duration-fast: 150ms;    --ease-standard: ease;
  --duration-short: 160ms;   --ease-emphasized: cubic-bezier(0.32,0.72,0,1);
  --duration-medium: 200ms;  --ease-in-out: ease-in-out;
  --duration-slow: 300ms;    --ease-out: ease-out;
}
```

### Typography Scale
```
--font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
--font-mono: 'Fira Code', 'SF Mono', Monaco, 'Cascadia Code', Consolas, monospace;

--text-xs: 12px;  --text-s: 14px;  --text-m: 16px;  --text-l: 20px;
```
