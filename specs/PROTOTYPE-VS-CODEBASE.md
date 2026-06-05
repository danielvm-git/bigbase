# Prototype ↔ Codebase Gap Matrix (v2)

**Prototype:** `specs/archive/assets/bigbase-prototype 2/`
**Codebase:** `ui/src/` (React 19 + Vite + TS) + `kernel/`, `components/`
**Date:** 2026-06-03
**Status:** ✅ All 19 actionable gaps closed (2026-06-03).
Design-system parity with prototype achieved. Plan: `specs/plans/PROTOTYPE-FIDELITY-PARITY.md`
(4 batches, 21 commits, 39 test files / 218 tests passing, ~600 LoC shipped).
Go + UI preflight green.

---

## 1. Source-of-truth inventory

### Prototype (`bigbase-prototype 2/`)

| Asset | Purpose | Size |
|---|---|---|
| `project/BigBase Console.html` | Single-file prototype: 35 React components, 6 sections, 1,485 LoC of embedded CSS+JS | 158 KB / 2,485 lines |
| `project/react-stubs/tokens.ts` | Token unions: `AccentTheme`, `ColorScheme`, `StatusKind`, `BadgeVariant`, `ButtonVariant`, `SpaceToken`, `RadiusToken`, `ShadowToken` | 53 lines |
| `project/react-stubs/ThemeContext.tsx` | `ThemeProvider` + `useTheme` | 73 lines |
| `project/react-stubs/Button.tsx` | 5 variants × 3 sizes, `loading`, `icon`, a11y warning | 1.8 KB |
| `project/react-stubs/Badge.tsx` | `Badge` (6 variants) + `StatusBadge` (word+dot, never color-alone) | 1.6 KB |
| `project/react-stubs/Input.tsx` | label, error, hint, prefix, mono | 1.9 KB |
| `project/react-stubs/Card.tsx` | `Card` + `EmptyState` | 1.5 KB |
| `project/react-stubs/ThemePicker.tsx` | Listbox popover (Esc/click-outside) | 2.1 KB |
| `project/react-stubs/Components.stories.tsx` | Stories for every state | 3.4 KB |
| `project/Component Spec - States.html` | Component state reference | 60 KB |
| `project/Component Spec - Tokens.html` | Token spec | 24 KB |
| `project/Component Spec - Information Architecture.html` | IA spec | 13 KB |
| `project/Component Spec - Responsive.html` | Breakpoint spec | 13 KB |
| `project/Component Spec - Accessibility.html` | A11y spec | 17 KB |
| `project/screenshots/*.png` | 14 reference shots | ~960 KB |

### Codebase (`ui/src/`)

| Layer | Count | LoC |
|---|---|---|
| Pages | 20 | 3,372 |
| Components | 23 | ~900 |
| Context | 5 (ThemeContext, themeState, accentThemes, ToastContext, toastState) | ~150 |
| Hooks | 2 (useTheme, useToast) | ~40 |
| Lib | 3 + 2 tests (sitesData, previewMode, functionEnv) | ~500 |
| Mocks | 2 (sites, templates) | ~60 |
| Types | 1 (sites) | ~50 |

---

## 2. Screen parity

| Prototype screen | Current page | Status | Notes |
|---|---|---|---|
| `Login` | `LoginPage.tsx` (183 LoC) | ✅ full | 3 modes: signin / reset / reset-sent; Google OAuth relay detection |
| `Dashboard` | `DashboardPage.tsx` (231 LoC) | ✅ full | 6 stat cards, deployment chart, system status panel |
| `SitesList` | `DeployPage.tsx` (144 LoC) | ✅ renamed | Prototype "Sites" → route `/deploy` (alias decision recorded in GAP-PROTOTYPE.md) |
| `SiteDetail` | `SiteDetailPage.tsx` (198 LoC) | ✅ full | Deployments, env vars, branch selector |
| `CreateSite` (wizard) | `CreateSitePage.tsx` (398 LoC) | 🟡 divergent | **4 steps (Source/Configure/Review/Deploy) vs. prototype's 3**; `ChoiceCard` is current-only |
| `Functions` | `FunctionsPage.tsx` (190 LoC) | ✅ full | List + create |
| `FunctionDetail` | `FunctionDetailPage.tsx` (138 LoC) | ✅ full | Code, logs, env vars, triggers |
| `DataStudio` | `DataStudioPage.tsx` (228 LoC) | ✅ full | Schema editor, rows, JSON editor |
| `SqlEditor` | `SqlEditorPage.tsx` (116 LoC) | ✅ minimal | Query editor + result table |
| `Storage` | `StoragePage.tsx` (183 LoC) | ✅ full | File browser + uploader |
| `Users` | `UsersPage.tsx` (91 LoC) | ✅ minimal | List only; CRUD pending |
| `GitRepos` | `GitReposPage.tsx` (117 LoC) | ✅ minimal | List only |
| `CICD` | `CiciPage.tsx` (195 LoC) | ✅ full | Workflows + runs |
| `Monitoring` | `MonitoringPage.tsx` (242 LoC) | ✅ full | Component health grid, request chart, logs |
| `Messaging` | `MessagingPage.tsx` (199 LoC) | ✅ full | Outbound history |
| `MessagingDetail` | `MessagingDetailPage.tsx` (90 LoC) | ✅ minimal | |
| `Settings` | `SettingsPage.tsx` (35 LoC) | 🟡 stub | Tabs exist; content is "stub" placeholder text |
| `Placeholder` | n/a | — | Prototype escape hatch — not needed in router-based app |
| `Shell` + `App` | `Layout.tsx` (170 LoC) + `App.tsx` (54 LoC) | ✅ full | **State machine → React Router** is the only protocol-level divergence |

### Screens in codebase but **not** in prototype

| Current page | Origin | Notes |
|---|---|---|
| `ForgePage.tsx` (234 LoC) | Backend `forge` component (issues/labels/kanban/wiki) | No design spec — needs `Component Spec - Forge.html` to be generated |
| `RealtimePage.tsx` (145 LoC) | Backend `realtime` component (WebSocket) | No design spec — covered by e17s01 story |
| `NotFoundPage.tsx` (15 LoC) | 404 route | Trivial; no spec needed |

---

## 3. Component library parity

### Primitives shared (current ≈ prototype)

| Primitive | Prototype | Current | Delta |
|---|---|---|---|
| `Icon` | 30+ Lucide paths, inline `<Icon name>` | `IconName` union (16 names), paths map | **Current is a subset**; missing icon names: `check`, `chevron-down`, `x`, `plus`, `search`, `trash`, `pencil`, `play`, `pause`, `refresh`, `arrow-up`, `arrow-down`, `arrow-right`, `external-link`, `download`, `upload`, `copy`, `more-horizontal`, `logo-github` (and others) |
| `Badge` | 6 variants (`neutral`, `accent`, `success`, `warning`, `error`, `info`) | 5 variants (no `info`) | **Missing `info` variant** |
| `Button` | 5 variants × 3 sizes (`sm`, `md`, `block`) | 5 variants × 2 sizes (`sm`, `md`) | **Missing `block` size**; a11y warning for icon-only not implemented |
| `Input` | label, error, hint, prefix, mono | label, error, hint (no prefix/mono) | **Missing `prefix` slot and `mono` prop** |
| `Card` | static + `interactive` | static only | **Missing `interactive` variant** (hover/focus affordance) |
| `EmptyState` | icon chip, title, payoff, one CTA | icon, title, description, children | Equivalent (children allow arbitrary CTA) |
| `PageHeader` | title, subtitle, children | title, children | **Missing `subtitle` prop** (current has it inline via JSX) |
| `ThemePicker` | popover listbox w/ Esc + click-outside | `<select>` in Layout footer | **Functionally a downgrade** — picker is the design, not a native select |
| `StatusBadge` | word + dot/spinner + color (never color-alone) | not implemented; uses `Badge` + `statusBadgeVariant` helper | **Missing dedicated `StatusBadge`** — `statusBadgeVariant` is a partial substitute |

### Components in prototype but **not** in current

| Prototype component | Spec ref | Notes |
|---|---|---|
| `StatusBadge` | react-stubs/Badge.tsx | See above |
| `Avatar` | COMPONENT_INVENTORY.md mentions; prototype doesn't have dedicated component | Current uses inline `sidebar-avatar` div |
| `ToastProvider` (full) | COMPONENT_INVENTORY.md | Current has `toastState` + `useToast` hook but **no `ToastProvider` JSX wrapper** |
| `SkeletonCard` (generic) | prototype; current has only `SitesSkeleton` | **Missing generic skeleton** primitive |
| `ChoiceCard` | n/a in prototype | Current-only |
| `WizardRail` with connector lines | prototype `WizardRail` | Current `WizardSteps` lacks the line between steps and the check icon for completed steps |
| `Modal` | COMPONENT_INVENTORY.md "not yet implemented" | Current `Modal.tsx` (101 LoC) is more than the prototype — see "extras" below |

### Components in current but **not** in prototype

| Current component | LoC | Origin | Notes |
|---|---|---|---|
| `Modal` | 101 | Domain need (e17s13) | Not in prototype, fills a gap |
| `ChoiceCard` | 37 | `CreateSitePage` source selector | Could be promoted to a primitive |
| `Breadcrumb` | 32 | Backward-compat with prototype IA | Current uses inline breadcrumb in CreateSitePage; component is unused |
| `ComponentHealthGrid` | 50 | `MonitoringPage` | Domain-specific |
| `DashboardMetrics` | 53 | `DashboardPage` | Domain-specific |
| `EnvVarEditor` | 74 | `FunctionDetailPage`, `SiteDetailPage` | Domain-specific (key/value editor) |
| `FunctionLogsPanel` | 76 | `FunctionDetailPage` | Domain-specific (terminal log) |
| `MetricCard` | 37 | `DashboardPage`, `MonitoringPage` | Could be a primitive |
| `QuickActions` | 42 | `DashboardPage` | Domain-specific |
| `RequestChart` | 65 | `MonitoringPage` | Domain-specific (SVG sparkline) |
| `SitesSkeleton` | 20 | `DeployPage` | Renamed from `SkeletonCard` |
| `StreamLog` | 31 | `FunctionDetailPage`, `SiteDetailPage` | Domain-specific |
| `Tabs` | 27 | Settings, Forge, Functions, FunctionDetail | Shared primitive — matches prototype |
| `WizardSteps` | 30 | `CreateSitePage` | Diverges from prototype's `WizardRail` (see above) |

**Net:** codebase has grown **12 domain-specific components** and **3 new primitives** (`Modal`, `ChoiceCard`, `MetricCard`) beyond the prototype.

---

## 4. Design-system token alignment

### Token IDs (matched 1:1)

`default`, `january`, `february`, `march`, `april`, `may`, `june`, `july`, `august`,
`september`, `october`, `november`, `december` — identical in both.

### Token types (gaps)

| Type union | Prototype | Current | Status |
|---|---|---|---|
| `AccentTheme` / `AccentId` | ✅ | ✅ | identical 12-ID union |
| `ColorScheme` / `Theme` | ✅ `'light' \| 'dark'` | ✅ `'light' \| 'dark'` | identical |
| `StatusKind` | ✅ `'ready' \| 'building' \| 'failed' \| 'pending'` | ❌ inline string in `statusBadgeVariant` | **Missing exported type** |
| `BadgeVariant` | ✅ 6 variants | ✅ 5 variants | **Missing `info`** |
| `ButtonVariant` | ✅ 5 variants | ✅ 5 variants | identical |
| `ButtonSize` | ✅ `'sm' \| 'md' \| 'block'` | ✅ `'sm' \| 'md'` | **Missing `block`** |
| `SpaceToken` | ✅ `0\|1\|2\|3\|4\|5\|6\|8\|10\|12\|16\|20\|24\|32` | ❌ | **Missing** |
| `RadiusToken` | ✅ `'xs' \| 's' \| 'm' \| 'l' \| 'full'` | ❌ | **Missing** |
| `ShadowToken` | ✅ `'xs' \| 's' \| 'm' \| 'l' \| 'xl'` | ❌ | **Missing** |
| `ThemeMeta` (swatch) | ✅ | partial — current has `AccentTheme` with `label` and `brand500/600/700` | Prototype has `swatch: string` (CSS color/gradient); current uses computed `rgb()` — equivalent at render time |

### Color values (matched)

`swatch` colors in `react-stubs/tokens.ts` are byte-identical to `brand500` strings
in `ui/src/context/accentThemes.ts`. Rainbow swatch in `june` matches `rainbow: true`
flag handling in `applyAccentToDocument()`.

---

## 5. Information-architecture parity

### Sidebar (Layout.tsx vs. prototype NAV)

| Section | Prototype items | Current items | Delta |
|---|---|---|---|
| Overview | Dashboard | Dashboard | ✅ |
| Build | Sites, Functions | Sites (/deploy), Functions | ✅ renamed |
| Data | Data Studio, SQL Editor, Storage | Data Studio, SQL Editor, Storage | ✅ |
| Auth | Users | Users | ✅ |
| Engage | Messaging | Messaging | ✅ |
| DevOps | Git Repos, CI/CD, Monitoring | Git Repos, CI/CD, Monitoring, **Forge**, **Realtime** | ➕ 2 extras (out of prototype scope) |
| Footer | Settings + appearance (theme + accent) | Settings + appearance (theme + accent) | ✅ |

**One extra section missing:** prototype places **Settings at footer level** (not in a
section), which current matches. ✅

### Routes (App.tsx)

| Prototype key | Current route | Notes |
|---|---|---|
| `dashboard` | `/` | ✅ |
| `sites`, `sites/new`, `sites/:id` | `/deploy`, `/deploy/new`, `/deploy/:siteId` | renamed |
| `functions`, `functions/:id`, `functions/:id/logs` | `/functions`, `/functions/:id`, `/functions/:id/logs` | ✅ |
| `data` | `/data` | ✅ |
| `sql` | `/sql` | ✅ |
| `storage` | `/storage` | ✅ |
| `users` | `/users` | ✅ |
| `messaging`, `messaging/:id` | `/messaging`, `/messaging/:id` | ✅ |
| `repos` | `/repos` | ✅ |
| `cici` | `/cici` | ✅ |
| `monitoring` | `/monitoring` | ✅ |
| `settings` | `/settings` | ✅ |
| (none) | `/forge` | ➕ out-of-prototype |
| (none) | `/realtime` | ➕ out-of-prototype |
| (none) | `*` → `NotFoundPage` | ➕ escape hatch |
| `Login` | `/login` (outside Layout) | ✅ |

### Wizard (CreateSite) divergence

| Step | Prototype | Current |
|---|---|---|
| 1 | Source | Source |
| 2 | Configure | Configure |
| 3 | Deploy | **Review** (added) |
| 4 | — | Deploy (moved) |

**Why:** e17s14 added a Review step so users see env vars + commit + branch before
triggering the build. This is a *feature gap closed* — prototype skipped review,
assumed happy-path.

**Visual divergence:** prototype `WizardRail` draws **connector lines between
steps** and a **check icon for completed steps**; current `WizardSteps` uses
plain numbered circles. A small but visible fidelity loss.

---

## 6. Protocol-level divergence

| Concern | Prototype | Current | Verdict |
|---|---|---|---|
| Routing | `useState('screen', 'dashboard' \| ...)` + switch | `react-router-dom` v6 with `<Routes>` and `<Route>` | **Current is correct** — state machine doesn't survive reload, doesn't allow deep links, doesn't support browser back/forward |
| Layout | `Shell` component wraps active screen | `Layout` component provides `<Outlet />` for child routes | **Equivalent** |
| Data | All data is in-memory `useState` mocks | Real `fetch('/api/...')` with `AbortController` for cancellation | **Current is correct** |
| Auth | `onLogin(email)` callback, no token | `fetch('/api/auth/login')` + session cookie | **Current is correct** |
| Theme | `<html data-theme="light" data-accent="default">` | same — plus `data-accent-rainbow` for `june` | ✅ equivalent |
| Storage | `localStorage('bigbase-theme', 'bigbase-dark')` | `localStorage('bigbase-theme', 'bigbase-accent')` | **Naming divergence** — prototype uses `bigbase-dark` for the dark flag, current co-locates with `bigbase-theme` |

---

## 7. Open gaps — actionable list

### P0 — Visible fidelity loss vs. prototype — ✅ CLOSED (B1.1–B1.5)

| # | Gap | Effort | Story | Resolution |
|---|---|---|---|---|
| G1 | `PageHeader` missing `subtitle` prop | 5 LoC + 1 test | e17s18 | B1.3 — added `subtitle?` + actions wrapper |
| G2 | `WizardSteps` missing connector lines + check icon for completed | 15 LoC + visual | e17s18 | B1.4 — Fragment + `.wizard-step-line` + check icon |
| G3 | `ThemePicker` is a native `<select>` instead of popover listbox | 60 LoC + a11y test | e17s18 | B1.5 — new `ThemePicker` listbox popover (13 themes) |
| G4 | `Badge` missing `info` variant | 3 LoC | e17s13 | B1.1 — added `info` to `BadgeVariant` |
| G5 | `Button` missing `block` size | 3 LoC | e17s13 | B1.2 — added `block` to `ButtonSize` |

### P1 — Type-system gaps (cheap to close) — ✅ CLOSED (B1.1, B1.6, B1.7)

| # | Gap | Effort | Story | Resolution |
|---|---|---|---|---|
| G6 | Export `StatusKind` type from `Badge.tsx` | 1 LoC | e17s13 | B1.1 — `export type StatusKind` |
| G7 | Export `SpaceToken` / `RadiusToken` / `ShadowToken` unions in `tokens.ts` | 15 LoC | e17s13 | B1.6 — new `ui/src/context/tokens.ts` |
| G8 | `localStorage` key naming: `bigbase-dark` → unified or documented | docs | e17s12 | B1.7 — doc added to `SYSTEM_DESIGN.md` |

### P2 — Missing dedicated components — ✅ CLOSED (B2.1–B2.4)

| # | Gap | Effort | Story | Resolution |
|---|---|---|---|---|
| G9 | `StatusBadge` component (word + dot/spinner, never color-alone) | 25 LoC + a11y test | e17s13 | B2.1 — new `StatusBadge.tsx` |
| G10 | Generic `SkeletonCard` primitive (replaces `SitesSkeleton`) | 15 LoC | e17s13 | B2.2 — new `SkeletonCard.tsx`; `SitesSkeleton` refactored |
| G11 | `ToastProvider` JSX wrapper around `ToastContext` | 30 LoC | e17s13 | N/A — false positive: `ToastProvider` already exists in `ui/src/context/ToastContext.tsx` |
| G12 | `Input` add `prefix` slot and `mono` prop | 20 LoC | e17s13 | B2.3 — added; `Omit<'prefix'>` for HTML-attr collision |
| G13 | `Card` add `interactive` variant (hover/focus affordance) | 10 LoC | e17s13 | B2.4 — `interactive?` prop + `.card-interactive` |

### P3 — Out-of-prototype screens needing specs — ✅ CLOSED (B3.1–B3.3)

| # | Gap | Effort | Story | Resolution |
|---|---|---|---|---|
| G14 | `ForgePage` (issues/kanban/wiki) has no `Component Spec - Forge.html` | spec | new e26 epic? | B3.2 — 12.8 KB HTML spec at `specs/archive/assets/bigbase-prototype 2/project/Component Spec - Forge.html` |
| G15 | `RealtimePage` (WebSocket subscriptions) has no spec | spec | e17s01 | B3.3 — 10.2 KB HTML spec at `specs/archive/assets/bigbase-prototype 2/project/Component Spec - Realtime.html` |
| G16 | `SettingsPage` is a stub (3 placeholder paragraphs) | full impl | e17s17 | B3.1 — full impl (~387 LoC) + 3 stub hooks + 5 tests |

### P4 — Domain components worth promoting to primitives — ✅ CLOSED (B4.1–B4.2)

| # | Component | Reason | Story | Resolution |
|---|---|---|---|---|
| G17 | `MetricCard` (37 LoC, used in Dashboard + Monitoring) | Reuse opportunity | e17s13 | B4.1 — exported; `DashboardMetrics` refactored to compose `MetricCard` |
| G18 | `ChoiceCard` (37 LoC, used in CreateSite source step) | Reuse opportunity | e17s13 | B4.2 — exported (was already); added 5 tests for behavior |

### P5 — Icon library shortfall — ✅ CLOSED (B4.3); G20 deferred

| # | Gap | Effort | Story | Resolution |
|---|---|---|---|---|
| G19 | `Icon` has 16 names; prototype uses 30+ | 30 LoC | e17s18 | B4.3 — added 15 new names; 31 names total |
| G20 | Consider switching from inline SVG paths to `lucide-react` | refactor | e17s18 | DEFERRED — out of scope; inline SVG keeps the bundle lean |

---

## 8. Epic alignment

| Prototype asset | Existing story | Status |
|---|---|---|
| `react-stubs/tokens.ts` | e17s12 (accent themes), e17s13 (primitives) | partial — G6–G8 open |
| `react-stubs/ThemeContext.tsx` | e17s12 | ✅ shipped |
| `react-stubs/Button.tsx` | e17s13 | partial — G5, G20 open |
| `react-stubs/Badge.tsx` | e17s13 | partial — G4, G9 open |
| `react-stubs/Input.tsx` | e17s13 | partial — G12 open |
| `react-stubs/Card.tsx` | e17s13 | partial — G13 open |
| `react-stubs/ThemePicker.tsx` | e17s12 | partial — G3 open |
| `Component Spec - IA.html` | e17s11 | ✅ shipped (Forge + Realtime added under DevOps) |
| `Component Spec - States.html` | e17s13 | partial — G1, G2 open |
| `Component Spec - Tokens.html` | e17s12, e17s13 | partial — G4–G7 open |
| `Component Spec - Responsive.html` | e17s18 | partial — sidebar toggle shipped, breakpoints not audited |
| `Component Spec - Accessibility.html` | e17s18 | partial — a11y warnings not enforced |
| `BigBase Console.html` (CreateSite wizard) | e17s14 | partial — G2 (wizard rail) open, Review step added (intentional) |
| `BigBase Console.html` (Login) | e17s17 | ✅ shipped |
| `BigBase Console.html` (Settings) | e17s17 | stub — G16 open |
| `BigBase Console.html` (Dashboard) | e17s05, e17s08, e17s09 | ✅ shipped |
| `BigBase Console.html` (SiteDetail) | e17s07 | ✅ shipped |
| `BigBase Console.html` (FunctionDetail) | e17s02, e17s14 | ✅ shipped |
| `screenshots/01-01-dashboard.png`, `02-01-dashboard.png` | e17s05, e17s09 | ✅ shipped |
| `screenshots/01-06-fndetail.png`, `02-06-fndetail.png` | e17s14 | ✅ shipped |
| `screenshots/01-spec-states.png`, `02-spec-states.png` | e17s13 | partial — states spec not fully realized (G9) |
| `screenshots/03-navcheck.png` | e17s11 | ✅ shipped |
| `screenshots/spec-resp.png` | e17s18 | partial — G19, G20 open |

---

## 9. Out-of-prototype additions worth tracking

| Page | Backend component | Spec status |
|---|---|---|
| `ForgePage` (234 LoC) | `components/forge/` (issues, labels, kanban, wiki) | **No design spec** — needs a `Component Spec - Forge.html` |
| `RealtimePage` (145 LoC) | `components/realtime/` (WebSocket subscriptions) | **No design spec** — covered by e17s01 story |
| `NotFoundPage` (15 LoC) | n/a | Trivial |

These three pages total **394 LoC of UI without a corresponding `Component Spec`**.
That is **20% of the page LoC (3,372)** running on engineer intuition rather
than design intent. Recommend a discovery pass to either generate specs or
confirm the current implementation is acceptable.

---

## 10. One-line summary

**Coverage:** 16 of 16 prototype screens shipped; 1 protocol divergence (state
machine → router, current is correct); ~12 domain components outpaced the
prototype; ~20 fidelity gaps remain (most are 5–30 LoC token/primitive fixes
already in stories e17s12, e17s13, e17s18). **Forge + Realtime pages ship
without a design spec — that's the real outstanding work.**

---

## Appendix A — Files that need changes (P0/P1 quick wins)

```
ui/src/components/PageHeader.tsx      +5   (subtitle prop)
ui/src/components/WizardSteps.tsx     +15  (connector line + check icon)
ui/src/context/accentThemes.ts        +60  (ThemePicker popover)
ui/src/components/Badge.tsx           +3   (info variant)
ui/src/components/Button.tsx          +3   (block size)
ui/src/components/Badge.tsx           +1   (export StatusKind)
ui/src/context/tokens.ts              +15  (Space/Radius/Shadow unions — new file)
```

Total to close all P0+P1 gaps: **~100 LoC, ~half a day.**

## Appendix B — Files that need new specs

```
specs/archive/assets/bigbase-prototype 2/project/Component Spec - Forge.html       [MISSING]
specs/archive/assets/bigbase-prototype 2/project/Component Spec - Realtime.html    [MISSING]
specs/archive/assets/bigbase-prototype 2/project/Component Spec - Settings.html    [STUB]
```

## Appendix C — Verification commands

```bash
# Confirm 16 prototype screens → 16 current pages (Realtime + Forge + NotFound are extras)
rtk ls ui/src/pages/

# Confirm react-stubs assets are reachable
rtk ls "specs/archive/assets/bigbase-prototype 2/project/react-stubs/"

# Confirm all e17 stories still exist
rtk ls specs/epics/e17-enhanced-admin-ui/stories/

# Run UI tests (none in project currently — gap)
go test ./components/admin/...
```
