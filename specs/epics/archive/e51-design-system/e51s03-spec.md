# e51s03: Layout Components

**Story ID:** e51s03 | **Epic:** e51 | **BCPs:** 4 | **Status:** planned
**Type:** feat | **Context:** domain
**Depends on:** e51s02 (core components) | **Blocks:** e51s04

## §1 — Summary

Extract the existing Layout structure (`Layout.tsx`) into reusable, exportable
sub-components: `Sidebar`, `SidebarSection`, `SidebarItem`, `AppShell` (the
outer layout frame), `Content` (scrollable main area), and `Page` (standard
page wrapper). Add responsive mobile drawer behavior to the sidebar — currently
the sidebar collapses to icons at 768px but lacks a proper slide-over drawer
pattern for small screens. Make the footer a standalone `AppFooter` component.

## §2 — Motivation

The current `Layout.tsx` is a single 190-line file mixing concerns: nav structure,
theme picker, user info, auth check, tutorial trigger, and footer. Extracting
sub-components makes them testable independently and reusable across future
layout variants (e.g., a "full-width" layout without sidebar for landing pages).

## §3 — Background / Context

- Existing: `Layout.tsx` (190 lines) — combines sidebar, content area, footer
- Existing: `Layout.test.tsx` — tests basic rendering
- Existing CSS classes: `.layout`, `.layout-body`, `.sidebar`, `.content`, `.app-footer`, `.sidebar-toggle`, `.sidebar-open`, `.sidebar-section`, `.sidebar-nav`, etc.
- Current responsive: sidebar shrinks to icons at 768px, toggle button appears
- Missing: proper mobile drawer (slide-over) for sub-480px screens

## §4 — Zoom-Out Check

- **Module purpose**: Admin console SPA shell with sidebar navigation
- **Callers**: `App.tsx` (renders `<Layout>` as route wrapper), every page via `<Outlet />`
- **Contracts**: Layout renders `<Outlet />` for child routes; fetches `/api/auth/me` and `/api/version`; uses `ThemeContext`, `ToastContext`, `HashRouter`

## §5 — Prior Art

- shadcn/ui Sidebar pattern: collapsible sidebar with mobile sheet
- Appwrite Console sidebar (icon + label, sections, footer with user)
- Our existing `Layout.tsx` is the functional reference

## §6 — Design Decisions

| Decision | Rationale |
|----------|-----------|
| Keep Layout.tsx as the composition layer | Pages use `<Layout />` in routes; don't break that contract |
| Extract sub-components to new files | Each gets its own test file |
| Mobile drawer as CSS-only slide-over | No extra JS library; use `<dialog>` or CSS `transform` with state |
| `AppShell` as the outermost frame | Separates layout structure from content |
| Sidebar items use `NavLink` from react-router-dom | Already the pattern; just extract to reusable components |

## §7 — Architecture / Component Design

```
ui/src/components/
  AppShell.tsx          ← NEW: outer layout frame (layout-grid)
  AppShell.test.tsx     ← NEW
  Sidebar.tsx           ← EXTRACT: from Layout.tsx — nav container
  Sidebar.test.tsx      ← NEW
  SidebarSection.tsx    ← EXTRACT: nav section with title
  SidebarItem.tsx       ← EXTRACT: single nav link with icon
  AppFooter.tsx         ← EXTRACT: footer bar
  AppFooter.test.tsx    ← NEW

ui/src/
  Layout.tsx            ← REFACTOR: compose from new components
```

## §8 — Data Model / Types

```typescript
// AppShell
interface AppShellProps {
  sidebar: ReactNode
  children: ReactNode
  footer?: ReactNode
}

// Sidebar
interface SidebarProps {
  children: ReactNode
  /** Controlled open state for mobile drawer */
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

// SidebarSection
interface SidebarSectionProps {
  title: string
  children: ReactNode
}

// SidebarItem
interface SidebarItemProps {
  to: string
  icon: IconName
  label: string
  end?: boolean  // exact match for NavLink
}

// AppFooter
interface AppFooterProps {
  version?: string
  showTutorialButton?: boolean
  onTutorialClick?: () => void
}
```

## §9 — API / Interface Contract

- `AppShell` renders CSS grid: sidebar | (content + footer stacked)
- `Sidebar` renders `<nav>` with `aria-label="Main navigation"`; supports `open` prop for mobile drawer
- `SidebarSection` renders a labeled `<div>` grouping `SidebarItem` children
- `SidebarItem` renders `<NavLink>` with icon + label
- `AppFooter` renders the sticky bottom bar with version, copyright, links
- All export from `ui/src/components/index.ts`

## §10 — State Management

Sidebar open/close state: currently in `Layout.tsx` as `useState`. Moves to
`Sidebar` component with controlled (`open`/`onOpenChange`) OR uncontrolled
(internal state) mode.

## §11 — Error Handling

- UI gracefully handles missing user data (shows "?" avatar initial)
- Auth redirect on 401 remains in `Layout.tsx` (not in sub-components)

## §12 — Testing Strategy

| Component | Tests |
|-----------|-------|
| AppShell | renders sidebar + content + footer slots |
| Sidebar | renders nav links, toggles open/closed, mobile drawer overlay |
| SidebarSection | renders title + children |
| SidebarItem | renders NavLink with icon, applies active class |
| AppFooter | renders version, link bar, tutorial button when enabled |

## §13 — Performance Considerations

- No lazy-loading changes — all components are in the initial bundle
- Mobile drawer uses CSS transform (GPU-accelerated)

## §14 — Security Considerations

- All links use `react-router-dom` `<NavLink>` (client-side routing)
- Footer external links use `target="_blank" rel="noopener noreferrer"`

## §15 — Accessibility

| Component | A11y requirements |
|-----------|------------------|
| Sidebar | `aria-label="Main navigation"`, mobile drawer has `aria-expanded`, focus trap when open |
| SidebarItem | Focus-visible ring, `aria-current="page"` from NavLink |
| AppFooter | Semantic `<footer>` element |
| Mobile toggle | `aria-expanded`, `aria-controls="sidebar-nav"` |

## §16 — Internationalization (i18n)

Nav labels and section titles passed as props — no hardcoded English in sub-components.

## §17 — Acceptance Criteria (Gherkin)

```gherkin
Scenario: AppShell renders all slots
  Given an AppShell with sidebar, children, and footer props
  When rendered
  Then sidebar, content, and footer are all visible

Scenario: Sidebar mobile drawer opens and closes
  Given a narrow viewport (< 768px)
  When the hamburger button is clicked
  Then the sidebar slides in from the left with an overlay

Scenario: SidebarItem highlights active route
  Given the user is on the /deploy page
  When SidebarItem with to="/deploy" renders
  Then it has the active class and aria-current="page"

Scenario: AppFooter shows version
  Given AppFooter with version="2.49.0"
  When rendered
  Then "v2.49.0" is visible
```

## §18 — Verification Script (Step-by-Step)

1. Run layout component tests: `cd ui && npx vitest run src/components/AppShell.test.tsx src/components/Sidebar.test.tsx src/components/AppFooter.test.tsx`
2. Run existing Layout test: `cd ui && npx vitest run src/Layout.test.tsx`
3. Run all tests: `cd ui && npm test`
4. Type check: `cd ui && npx tsc --noEmit`
5. Build UI: `cd ui && npm run build`
6. Build Go: `cd .. && go build ./...`
7. Visual check: run dev server, verify sidebar works at 320px, 768px, 1024px

## §19 — Out of Scope

- Breadcrumb auto-generation from route tree
- Keyboard shortcut (⌘K) command palette
- Multiple layout variants (just the existing sidebar layout)
- Collapsible sidebar sections (accordion pattern)

## §20 — Risks

| Risk | Mitigation |
|------|-----------|
| Existing Layout test breaks after refactor | Run Layout.test.tsx after each extraction step |
| Mobile drawer animation jank | Use CSS `transform: translateX()` with `transition` — GPU layer |
| Sidebar re-renders cause flash | Memoize `SidebarSection`/`SidebarItem` with `React.memo` |
