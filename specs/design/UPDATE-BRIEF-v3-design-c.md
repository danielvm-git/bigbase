# Update Brief v3 — Design C: Project-Scoped Architecture

**Posted to:** claude.ai/design prototype project (BigBase Console — Design C / v3)  
**Purpose:** Design the post-e57/e58 IA — project-scoped routing with two-zone nav  
**Scope:** New prototype file (do not overwrite Evolved v2) — this is the e58 target state  
**Prerequisite:** Apply UPDATE-BRIEF-v3-fix.md first so Forge/Realtime/Events exist in the baseline

---

## Prompt to post to claude.ai/design

---

I need you to create a new prototype file called "BigBase Console — Design C" that shows the BigBase admin console after the Project Scoping epic (e57/e58) is implemented. This is a structural IA change — do NOT change the visual design, color scheme, or component style from the Evolved v2 prototype.

**Context:** BigBase is adding project-level isolation. Instead of one global database, each project has its own PostgreSQL branch. The sidebar needs to split into a Project Zone (per-project pages) and a Global Zone (platform-wide pages). This is inspired by the Neon database console.

---

## What is changing (e57/e58 scope)

**Before:** Flat sidebar, all pages global, no project concept  
**After:** Header has a project/branch selector; sidebar has two distinct zones

**NOT changing in this epic:** Visual theme, component library, Global Zone pages (Sites, Functions, Storage, etc.)

---

## New Header Bar

Add a persistent top bar to all pages:

```
[BigBase logo]  [Project ▾ my-db]  [Branch ▾ main]  |  SQL Editor        [Daniel ▾]  [⚙]
```

- **Logo:** BigBase logo on far left, links to global dashboard
- **Project selector dropdown:** Shows selected project name with ▾ caret. Clicking opens a popover list of all projects + "Create new project" option at bottom. When no project selected, shows "Select a project" in muted text.
- **Branch selector dropdown:** Only visible when a project is selected. Shows current branch (default: main). Clicking opens a list of branches for that project + "Create branch" option. When on a global page (not project-scoped), hide the branch selector.
- **Page title:** After the separating | character, the current page name in normal weight
- **User avatar/name:** Far right, with a dropdown for profile/logout
- **Settings icon:** Gear icon linking to settings

**Component:** Use the existing Input and DropdownMenu components; style the selectors as compact interactive chips (border, rounded, hover state).

---

## Two-Zone Sidebar

The sidebar splits into two clearly distinct zones with a visual divider between them.

### Zone 1 — Project Zone (top, accent-colored)

Only visible when a project is selected. When no project is selected, show a collapsed "Select a project" placeholder in this zone.

```
PROJECT                                   (section header, accent color)
  ◉  Dashboard          /project/:id
  ◉  SQL Editor         /project/:id/sql/main
  ◉  Branches           /project/:id/branches   [future badge]
  ◉  Settings           /project/:id/settings
```

- Use a slightly different background or left-border accent to distinguish this zone from the global zone
- Active item: same indigo highlight as current sidebar
- "Branches" item has a small "soon" badge (muted, small) — it's planned but not in e58

### Zone 2 — Global Zone (bottom, unchanged)

Same as current sidebar. Always visible regardless of project selection.

```
──────────────────────────── (divider)
PLATFORM                                  (section header, muted)

Build
  Sites         /sites
  Functions     /functions

Data
  Data Studio   /data
  Storage       /storage

Auth
  Users         /users

Engage
  Messaging     /messaging

DevOps
  Git Repos     /repos
  CI/CD         /cici
  Monitoring    /monitoring
  Forge         /forge
  Realtime      /realtime
  Events        /events

────────────────────────────
Settings        /settings   (footer)
```

---

## Pages to Design (6 new screens)

### Screen 1: Global Dashboard (no project selected)

**URL:** `/` (when no project is active)  
**Shown when:** User first opens the app or has no project selected

Layout:
- Top: Welcome banner — "Welcome to BigBase, Daniel" with a muted subtitle "Select a project to get started or create a new one."
- Row of global stat cards:
  - Total Projects: 3
  - Sites: 5 (linked to /sites)
  - Functions: 8 (linked to /functions)
  - Auth Users: 142 (linked to /users)
- Projects list section (heading "Your projects"):
  - 3 project cards in a responsive grid (2-3 per row):
    - Card: project name, plan badge (Free/Pro), branch count "3 branches", last active "2h ago", "Open" button (goes to /project/:id)
    - Example: "my-db" · Free · 3 branches · 2h ago
    - Example: "staging" · Free · 1 branch · 1d ago  
    - Example: "analytics" · Pro · 5 branches · 3d ago
  - "Create new project" card (dashed border, + icon, muted text)
- System health panel at bottom (same as current DashboardPage health grid)

Components: Card, Button, Badge, MetricCard, ComponentHealthGrid, EmptyState (for zero projects)

---

### Screen 2: Project Dashboard (project selected)

**URL:** `/project/p_123`  
**Header state:** Project selector shows "my-db", Branch selector shows "main"  
**Sidebar:** Project Zone is expanded showing Dashboard (active), SQL Editor, Branches, Settings

Layout:
- Page header: "my-db" (project name) with a StatusBadge "Active" and a "Connection details" button
- **Connection string card** (prominent):
  - Tab bar: psql | URI | .env
  - Code block (monospace, copyable): `postgres://daniel:••••••••@bb.example.com:5432/my-db`
  - CopyButton on the right
  - Small note: "Never expose connection strings in client-side code"
- **Quick stats row** (4 metric cards):
  - Tables: 12
  - Total rows: 48,392
  - Storage: 2.4 MB
  - Queries today: 1,203
- **Recent queries** (collapsible list, last 5):
  - Each item: SQL snippet (monospace, truncated) · timestamp · duration badge · "Open in SQL Editor" link
  - Example: `SELECT * FROM users WHERE created_at > '2026-06-28'` · 2m ago · 12ms
- **Branches** (2-column cards):
  - main branch card: name "main", badge "protected", "3 ahead of origin", last active "2h ago", "Open SQL Editor" button
  - dev branch card: name "dev", "up to date", last active "5h ago", "Open SQL Editor" button
  - "Create branch" card (dashed border)

Components: Card, Button, Badge, StatusBadge, CodeBlock, CopyButton, MetricCard, Tabs

---

### Screen 3: Unified SQL + Data Page

**URL:** `/project/p_123/sql/main`  
**Header state:** Project "my-db", Branch "main" (branch selector is prominent here)

Layout — 3 panel split:

**Left panel (280px, scrollable):**
- Header: "Schema" with a search input (filter tables)
- Tree list of tables:
  ```
  ▾ users (5 cols)
      id          uuid PK
      email       text
      name        text
      created_at  timestamptz
      role        text
  ▾ posts (4 cols)
      id          uuid PK
      title       text
      content     text
      user_id     uuid FK
  ▸ comments (3 cols)
  ▸ attachments (4 cols)
  ```
- Click on table name → inserts `SELECT * FROM users LIMIT 50;` into editor
- Right-click → context menu: Copy name, Copy SELECT, Copy INSERT template

**Top-right panel (flexible height):**
- Tab bar: "Query 1" + button to add new tab (like a browser)
- SQL editor: monospace, dark background, syntax highlighting, line numbers
  ```sql
  SELECT u.id, u.email, COUNT(p.id) as post_count
  FROM users u
  LEFT JOIN posts p ON p.user_id = u.id
  WHERE u.created_at > '2026-06-01'
  GROUP BY u.id, u.email
  ORDER BY post_count DESC
  LIMIT 25;
  ```
- Toolbar below editor: [▶ Run] [Save] [Format] buttons + query duration badge "12ms" (shown after run)
- Resize handle between top and bottom right panels

**Bottom-right panel (flexible height):**
- Tab bar: Results | Messages | Explain
- Results tab: A table with the query results
  - Sortable column headers
  - Cell values (numbers right-aligned, text left-aligned, nulls in italic muted)
  - Pagination: "Showing 25 of 142 rows" + Next/Prev buttons
  - Export button: "Export CSV"
  - Inline cell edit on double-click (shows save/cancel buttons)
- Messages tab (empty when no errors): "Query completed successfully in 12ms"
- Explain tab: Shows query plan as a tree (placeholder text: "Run EXPLAIN ANALYZE to see the query plan")

Components: Input, Tabs, Button, Badge, CodeBlock, DropdownMenu

---

### Screen 4: Branches Page

**URL:** `/project/p_123/branches`  
**Status:** Mark this as a "coming soon" feature (e59s02 scope, not e58)

Layout:
- Page header: "Branches" + "Create branch" button (disabled/muted with a "Coming soon" tooltip)
- **Banner at top:** A muted info banner — "Branch management is coming in a future release. You'll be able to create isolated database branches for development, testing, and preview environments."
- Below the banner, show a grayed-out mockup (opacity 0.4) of what the branch list will look like:
  - Branch cards: main (protected), dev, feature/auth-refactor
  - Each card: branch name, base branch, commits ahead/behind, last updated, status badge
  - Buttons: Switch | Merge | Delete (all disabled)
- No "Create branch" modal (since it's disabled)

Components: Alert (info variant), Card, Badge, Button (disabled state)

---

### Screen 5: Project Settings

**URL:** `/project/p_123/settings`

Layout:
- Page header: "Project Settings" with project name breadcrumb
- **Tab bar:** General | Database | Environment Variables | Danger Zone

**General tab (default):**
- Form field: Project name (Input, prefilled with "my-db", [Save] button)
- Form field: Description (Textarea, optional)
- Project ID (read-only Input with CopyButton): `p_123abc456`
- Region (read-only): "US East (Virginia)"

**Database tab:**
- Connection pool size (Input type=number, default 10, range 1-100)
- Compute size dropdown: "Shared (0.25 vCPU)" / "Small (0.5 vCPU)" / "Medium (1 vCPU)" — show as ChoiceCard group
- Auto-suspend after inactivity (Switch, default on, duration select)
- [Save] button

**Environment Variables tab:**
- EnvVarEditor component (add/edit/delete key-value pairs)
- Keys shown, values masked (•••••) with a reveal toggle per row
- [Add variable] button at bottom
- Note: "These are injected into your project's deployed functions and sites as environment variables."

**Danger Zone tab:**
- Section: Reset Database — "Permanently delete all data in this project. Cannot be undone." → [Reset Database] button (danger/red, opens confirmation modal)
- Section: Delete Project — "Permanently delete this project and all its data." → [Delete Project] button (danger/red, opens confirmation modal with type-to-confirm input)
- Both buttons use the danger Button variant (red)

Components: Input, Textarea, Tabs, ChoiceCard, Switch, EnvVarEditor, Button (danger), Dialog (modal), CopyButton

---

### Screen 6: Project Selector + Branch Selector (Header Popover)

**URL:** Overlay/popover, appears on top of any page

**Project selector popover (when Project chip is clicked):**
```
┌─────────────────────────────┐
│  Switch project              │
│  ─────────────────────────  │
│  ● my-db          (current) │
│    staging                   │
│    analytics                 │
│  ─────────────────────────  │
│  + Create new project        │
└─────────────────────────────┘
```
- Current project has a dot indicator
- Clicking a project → URL changes to `/project/:id`, sidebar Project Zone updates
- "Create new project" → opens a modal with: Project name input, [Create] button

**Branch selector popover (when Branch chip is clicked):**
```
┌─────────────────────────────┐
│  Switch branch               │
│  ─────────────────────────  │
│  ● main           (current) │
│    dev                       │
│  ─────────────────────────  │
│  + Create branch  (disabled, soon) │
└─────────────────────────────┘
```
- Clicking a branch → URL changes to `/project/:id/sql/:branchName`
- "Create branch" is disabled with a "soon" tooltip

**"Create new project" modal:**
- Title: "Create project"
- Input: Project name (alphanumeric + hyphens, e.g. "my-db")
- Note: "Your project will start with an empty PostgreSQL database on the main branch."
- [Create project] button (primary) · [Cancel] button (ghost)

Components: DropdownMenu or custom Popover, Input, Button, Dialog

---

## Constraints

- Keep the exact same visual theme as Evolved v2: dark background, indigo accent, same spacing, same typography
- All components used must be from the BigBase component library (Button, Card, Input, Tabs, Badge, StatusBadge, Modal/Dialog, Icon, EmptyState, MetricCard, EnvVarEditor, DropdownMenu, CopyButton, ChoiceCard, CodeBlock, Alert)
- The Global Zone pages (Sites, Functions, etc.) are unchanged — do not redesign them
- Dark mode only — do not add light mode variants
- The prototype should be interactive: clicking the Project selector updates the sidebar state, clicking a branch updates the SQL editor URL, tab switching works within pages
- Mark the Branches page clearly as "coming soon" — it's e59s02 scope, not part of e58

## Deliverable

A new prototype file "BigBase Console — Design C.html" with:
- The new header bar (project + branch selectors)
- Two-zone sidebar (Project Zone + Global Zone)
- 6 new screens: Global Dashboard, Project Dashboard, SQL+Data, Branches, Project Settings, Project Selector popovers
- Clicking the project selector and navigating between project-scoped pages works interactively
- The Branches page shows the "coming soon" treatment
- All Global Zone pages remain accessible and unchanged
