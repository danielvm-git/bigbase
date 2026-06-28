# e51s04: Console Page Templates

**Story ID:** e51s04 | **Epic:** e51 | **BCPs:** 3 | **Status:** planned
**Type:** feat | **Context:** domain
**Depends on:** e51s03 (Layout Components) | **Blocks:** e57, e59, e61, e62, e63, e64

## §1 — Summary

Create standardized page template components that enforce consistent layout,
spacing, and behavioral patterns across all admin console pages. Currently each
page manually composes `<PageHeader>` + content with ad-hoc CSS class strings.
Templates provide: `ListPage` (for tables with header + filters + pagination area),
`DetailPage` (for entity detail views with tabs + back navigation), and
`SettingsPage` (for configuration forms with sections). Refactor 3 existing
pages to use the new templates as proof of integration.

## §2 — Motivation

Every page in the admin console repeats the same patterns:
- PageHeader with title/subtitle/actions
- Content area with variable layout
- Loading/empty/error states
- Tab navigation for detail pages
- Section-based layout for settings

Standardizing these into templates eliminates ~20% of duplicated layout code
and ensures consistent spacing, loading states, and responsive behavior.

## §3 — Background / Context

- 18 page components in `ui/src/pages/`
- Each page manually renders `<PageHeader>`, handles loading/error states inline
- Existing `PageHeader` component is well-tested and reusable
- Existing `Card`, `Badge`, `Tabs`, `EmptyState` components are available
- Common page patterns observed:
  - List pages: DashboardPage, DeployPage, FunctionsPage, StoragePage, UsersPage, GitReposPage, MessagingPage, ForgePage, CiciPage, MonitoringPage, EventsPage
  - Detail pages: SiteDetailPage, FunctionDetailPage, MessagingDetailPage
  - Settings: SettingsPage
  - Form pages: CreateSitePage
  - Special: LoginPage, NotFoundPage, DataStudioPage, SqlEditorPage

## §4 — Zoom-Out Check

- **Module purpose**: Admin console page routing layer
- **Callers**: `App.tsx` route definitions (each page is a route component)
- **Contracts**: Pages are React components rendered inside `<Layout />` via `<Outlet />`; they fetch data from `/api/*` endpoints

## §5 — Prior Art

- shadcn/ui doesn't have page templates — it's component-level only
- Appwrite Console uses consistent page layouts (header + tabs or header + filter bar + table)
- Pattern: React "render props" or "slots" pattern for template customization

## §6 — Design Decisions

| Decision | Rationale |
|----------|-----------|
| Templates as wrapper components (not HOCs or render props) | Simplest pattern; use children + named slots |
| `Page` as the base shell | Every page wraps in `<Page>` for consistent padding, max-width |
| `ListPage` for table/list views | Header + filter slot + content + pagination |
| `DetailPage` for entity views | Header with back button + tab bar + tab content |
| `SettingsPage` for config forms | Section-based layout with dividers |
| Refactor 3 pages first | Prove the templates work before refactoring all 18 |

## §7 — Architecture / Component Design

```
ui/src/components/
  Page.tsx              ← NEW: base page wrapper (consistent padding, heading area)
  Page.test.tsx         ← NEW
  ListPage.tsx          ← NEW: list/table page template
  ListPage.test.tsx     ← NEW
  DetailPage.tsx        ← NEW: detail view template with tabs
  DetailPage.test.tsx   ← NEW
  SettingsPage.tsx      ← NEW: settings page template with sections
  SettingsPage.test.tsx ← NEW

Pages to refactor for proof (3):
  ui/src/pages/UsersPage.tsx         → use ListPage
  ui/src/pages/SiteDetailPage.tsx    → use DetailPage
  ui/src/pages/SettingsPage.tsx      → use SettingsPage
```

## §8 — Data Model / Types

```typescript
// Base Page
interface PageProps {
  children: ReactNode
  className?: string
}

// ListPage
interface ListPageProps {
  title: string
  subtitle?: string
  actions?: ReactNode          // header action buttons (e.g., "Create")
  filters?: ReactNode          // filter bar above the list
  emptyState?: ReactNode       // shown when data is empty
  loading?: boolean
  error?: string
  children: ReactNode          // the list content (table, grid, etc.)
  pagination?: ReactNode       // pagination controls below the list
}

// DetailPage
interface DetailPageProps {
  title: string
  subtitle?: string
  backTo?: string              // route to navigate back to
  backLabel?: string           // e.g., "Back to Sites"
  actions?: ReactNode
  tabs: { id: string; label: string; content: ReactNode }[]
  defaultTab?: string
  loading?: boolean
  error?: string
}

// SettingsPage
interface SettingsPageProps {
  title: string
  subtitle?: string
  sections: {
    title?: string
    description?: string
    content: ReactNode
  }[]
  loading?: boolean
}
```

## §9 — API / Interface Contract

- All templates render `<PageHeader>` automatically from title/subtitle/actions props
- `loading=true` shows `<Spinner />` (from e51s02)
- `error` string shows error state with retry affordance
- Templates accept `className` for per-page overrides
- Templates do NOT fetch data — pages still own their data fetching

## §10 — State Management

Templates manage only UI state: active tab in DetailPage. All data state stays
in page components.

## §11 — Error Handling

- Error state: template renders error message with optional retry callback
- Loading state: shows spinner (from e51s02)
- Empty state: renders `EmptyState` component

## §12 — Testing Strategy

| Component | Tests |
|-----------|-------|
| Page | renders children, applies className |
| ListPage | renders header/title, shows loading spinner, shows error, shows empty state, renders filters slot, renders pagination slot |
| DetailPage | renders tabs, switches active tab on click, shows back button link, shows loading/error |
| SettingsPage | renders sections with titles and dividers, shows loading |
| Refactored pages | Existing page tests still pass after refactor |

## §13 — Performance Considerations

- Templates are pure composition — no data fetching, no effects
- `React.memo` on template components prevents unnecessary re-renders

## §14 — Security Considerations

N/A — templates are presentational only.

## §15 — Accessibility

| Component | A11y requirements |
|-----------|------------------|
| ListPage | Semantic `<section>` with `aria-label`; loading state has `aria-busy="true"` |
| DetailPage | Tabs follow WAI-ARIA tabs pattern; back link is keyboard accessible |
| SettingsPage | Form sections use `<fieldset>` + `<legend>` where applicable |

## §16 — Internationalization (i18n)

All text (titles, subtitles, tab labels, back labels) passed as props — no hardcoded strings.

## §17 — Acceptance Criteria (Gherkin)

```gherkin
Scenario: ListPage shows loading state
  Given a ListPage with loading=true
  When rendered
  Then a spinner is visible and the content slot is hidden

Scenario: ListPage shows error state
  Given a ListPage with error="Failed to load"
  When rendered
  Then the error message is displayed with an error icon

Scenario: ListPage shows empty state
  Given a ListPage with no children and emptyState content
  When rendered
  Then the empty state content is displayed instead of an empty list

Scenario: DetailPage switches tabs
  Given a DetailPage with 3 tabs
  When the user clicks the second tab
  Then the second tab content is visible and others are hidden

Scenario: SettingsPage renders sections
  Given a SettingsPage with 2 sections
  When rendered
  Then both sections are visible with dividers between them
```

## §18 — Verification Script (Step-by-Step)

1. Run template tests: `cd ui && npx vitest run src/components/Page.test.tsx src/components/ListPage.test.tsx src/components/DetailPage.test.tsx src/components/SettingsPage.test.tsx`
2. Run refactored page tests: `cd ui && npx vitest run src/pages/UsersPage.test.tsx src/pages/SiteDetailPage.test.tsx src/pages/SettingsPage.test.tsx`
3. Run all tests: `cd ui && npm test`
4. Type check: `cd ui && npx tsc --noEmit`
5. Build UI: `cd ui && npm run build`
6. Build Go: `cd .. && go build ./...`

## §19 — Out of Scope

- Refactoring all 18 pages (just 3 for proof)
- Page-level code splitting (lazy loading)
- Breadcrumb auto-generation
- Page transition animations

## §20 — Risks

| Risk | Mitigation |
|------|-----------|
| Templates too rigid for edge-case pages | Keep template props optional; pages can bypass templates for special layouts |
| Refactored page tests break | Run page tests before and after refactor; verify behavioral equivalence |
| Templates add abstraction overhead for simple pages | Base `Page` component is minimal (just padding); only use full templates where they add value |
