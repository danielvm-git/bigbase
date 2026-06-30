# Update Brief — e58: Project Scoping UI (Design C)

**Prototype file:** `BigBase Console.html`  
**Project:** BigBase Prototype (`ec1480a1`)  
**Effort:** Large — new sidebar zones, header bar, 5 new screens  
**Epic:** e58-project-scoping-ui  
**Source:** `specs/DESIGN_C_HANDOFF.md`

---

## Summary

Add a **project-scoped zone** to the sidebar, a **project/branch header bar**, and **5 new screens** for the project layer. All 20 existing screens remain unchanged. New screens are added as navigable sections reachable from the new sidebar zone.

---

## 1. Sidebar — add Project Zone above Global Zone

Replace the current single sidebar with a two-zone sidebar. The Project Zone appears at the top when a project is selected; the existing Global Zone stays below unchanged.

```
┌─────────────────────────────────────────────────┐
│ [B] BigBase                                      │  ← logo row (unchanged)
│                                                  │
│ PROJECT                                          │  ← new accent-colored section header
│ [◈ my-db          ▾]                            │  ← project selector button (full width)
│   ● Dashboard                                    │  ← project-scoped nav items
│   ○ SQL Editor                                   │
│   ○ Branches                                     │
│   ○ Settings                                     │
│                                                  │
│ ─────────────────────────────────────────────── │  ← thin divider
│                                                  │
│ Overview                                         │  ← global zone starts here (unchanged)
│   ○ Dashboard                                    │
│ Build                                            │
│   ○ Sites                                        │
│   ○ Functions                                    │
│ Data                                             │
│   ○ Data Studio                                  │
│   ○ Storage                                      │
│ Auth                                             │
│   ○ Users                                        │
│ Engage                                           │
│   ○ Messaging                                    │
│ DevOps                                           │
│   ○ Git Repos                                    │
│   ○ CI/CD                                        │
│   ○ Monitoring                                   │
│   ○ Forge                                        │
│   ○ Realtime                                     │
│   ○ Events                                       │
│ ─────────────────────────────────────────────── │
│ APPEARANCE / Settings / User / Logout            │  ← footer (unchanged)
└─────────────────────────────────────────────────┘
```

**Project Zone styling:**
- Section header "PROJECT" in same uppercase muted style as existing section labels
- Project selector: secondary sm button, full width, icon `layers` left + project name + chevron right
- Active nav item: same indigo left-border style as global zone
- When no project selected: show only the selector button with placeholder "Select a project…", nav items hidden

**Initial state (no project):**
```
PROJECT
[◈ Select a project…  ▾]   ← placeholder, click opens picker
```

---

## 2. Context Bar — add below top logo, above content on project pages

A slim bar appears between the sidebar and the page content **only on project routes** (`/project/*`). On global pages (sites, functions, etc.) it is hidden.

```
┌────────────────────────────────────────────────────────────────┐
│ [◈ my-db  ▾]   [⎇ main  ▾]   │   SQL Editor                   │
└────────────────────────────────────────────────────────────────┘
```

- Left: project selector (same as sidebar, 160px wide) + branch selector (same style, 140px)
- Separator `│` (vertical rule, muted)
- Right: current page title (same `page-title` style, but smaller — 16px not 24px)
- Bar height: 44px, `bg-surface`, thin bottom border
- Dropdowns open inline (same style as existing `<select>` elements)

---

## 3. New Screen — Projects List (`/projects`)

**Sidebar trigger:** Click "Select a project…" → shows this page, or direct navigation.

```
Projects                                          [+ New project]
Your database projects

┌──────────────────────────────────────────────────────────────────────┐
│ Name           Slug         Tables  Storage    Last active           │
│──────────────────────────────────────────────────────────────────────│
│ my-db          my-db        12      48 MB      2h ago         [Open] │
│ blog-db        blog-db      4       12 MB      1d ago         [Open] │
│ staging-db     staging-db   12      44 MB      3d ago         [Open] │
└──────────────────────────────────────────────────────────────────────┘
```

Empty state (no projects):
```
[◈]
No projects yet
Create a project to start working with your database.
[+ Create project]
```

**New project modal:**
```
┌──────────────────────────────┐
│ New project               ✕  │
│                              │
│ [Name *                    ] │
│ [Slug *                    ] │
│ (auto-fills from name)       │
│                              │
│ [Cancel]  [Create project]   │
└──────────────────────────────┘
```

---

## 4. New Screen — Project Dashboard (`/project/:id`)

Header (context bar shows project + branch):

```
Project Dashboard
Overview for my-db

┌──────────────────────────────────────────────────────────────────┐
│ Connection string                                          [psql ▾]│
│                                                                   │
│ postgres://daniel@bigbase.local/my-db?sslmode=disable  [Copy]    │
└──────────────────────────────────────────────────────────────────┘

[12]         [1.2M]        [48 MB]       [548]
Tables       Rows          Storage       Queries today

Recent queries                                         [Open editor →]
─────────────────────────────────────────────────────────────────────
SELECT * FROM users WHERE created_at > ...             2h ago
SELECT COUNT(*) FROM posts GROUP BY status             5h ago
CREATE INDEX idx_posts_user ON posts(user_id)          1d ago

Branches (2)
─────────────────────────────────────────────────────────────────────
● main        12 tables  48 MB  active     [Open SQL editor]
○ dev         12 tables  44 MB  2d ago     [Open SQL editor]
                                            [+ Create branch]
```

- Connection string card: full-width, monospace value, [Copy] icon button, format dropdown `[psql ▾]` with options: psql / URI / env var
- Stat row: 4 equal stat cards (same pattern as existing dashboard)
- Recent queries: muted monospace list, each row is a link → opens SQL editor with query pre-filled
- Branches card: list with status dot (green=active, gray=inactive), table count, size, last active time

---

## 5. New Screen — SQL Editor (`/project/:id/sql/:branch?`)

3-panel layout (replaces current single-panel SqlEditorPage for project scope):

```
┌──────────────────────────────────────────────────────────────────────┐
│  [⎇ main ▾]    [Run (⌘⏎)]  [Save]  [Format]               [Export] │  ← toolbar
├─────────────────┬──────────────────────────────────────────────────  │
│ Schema          │  SELECT * FROM users                               │
│ ─────────────── │  WHERE created_at > '2026-01-01'                  │
│ ▾ users         │  LIMIT 100                                         │
│    id  integer  │                                                    │
│    email  text  │  (monospace textarea, 10 rows)                    │
│    created_at   │                                                    │
│ ▾ posts         ├────────────────────────────────────────────────── │
│    id  integer  │  [Results]  [Messages]  [Explain]                  │
│    title  text  │  ──────────────────────────────────────────────── │
│    user_id int  │  3 rows returned  ·  12ms                          │
│ ▾ sessions      │  ┌────────┬──────────────────────┬───────────┐   │
│    (3 cols)     │  │ id     │ email                │ created   │   │
│                 │  ├────────┼──────────────────────┼───────────┤   │
│ [+ Add table]   │  │ 1      │ daniel@gmail.com     │ 2026-01   │   │
│                 │  │ 2      │ team@co.com          │ 2026-01   │   │
│                 │  └────────┴──────────────────────┴───────────┘   │
└─────────────────┴────────────────────────────────────────────────── │
```

- Left panel (240px, fixed): schema tree. `▾` expands table showing columns + types. Clicking a table name inserts `SELECT * FROM table_name` into editor. `[+ Add table]` ghost button at bottom.
- Top-right panel: monospace textarea, 10 rows, dark bg (same `.bb-terminal` style but editable). Toolbar above: branch selector + Run/Save/Format buttons left, Export right.
- Bottom-right panel: tabs Results / Messages / Explain. Results shows table with row count + latency. Empty state: "Run a query to see results."

---

## 6. New Screen — Branches (`/project/:id/branches`)

```
Branches                                           [+ Create branch]
my-db  ·  2 branches

┌──────────────────────────────────────────────────────────────────────┐
│ ● main (default)    12 tables  48 MB   active       [Open SQL editor]│
├──────────────────────────────────────────────────────────────────────┤
│ ○ dev               12 tables  44 MB   2d ago    [Switch]  [Delete]  │
└──────────────────────────────────────────────────────────────────────┘
```

- Each branch is a card row: status dot + name + badge "(default)" if main + table count + size + last active time + actions
- `main`: no Delete button (protected). `[Open SQL editor]` links to `/project/:id/sql/main`
- Non-main branches: `[Switch]` (changes context bar branch to this) + `[Delete]` (danger, confirm modal)

**Create branch modal:**
```
┌──────────────────────────────────┐
│ Create branch                  ✕ │
│                                  │
│ [Branch name *               ]   │
│ From: [main ▾]                   │
│                                  │
│ [Cancel]  [Create]               │
└──────────────────────────────────┘
```

---

## 7. New Screen — Project Settings (`/project/:id/settings`)

```
Settings
my-db

[General]  [Database]  [Env Vars]  [Danger Zone]
```

#### General tab
```
┌─────────────────────────────────────┐
│ Project name    [my-db            ] │
│ Slug            [my-db            ] │  ← auto-slug, editable
│                         [Save]      │
└─────────────────────────────────────┘
```

#### Database tab
```
┌─────────────────────────────────────┐
│ Compute size      [1 vCPU / 256 MB] │
│ Connection pool   [10             ] │
│                         [Save]      │
└─────────────────────────────────────┘
Note: Changes take effect on next cold start.
```

#### Env Vars tab
Same env var editor as Site Detail Env Vars tab:
```
Environment Variables                             [+ Add variable]

KEY              VALUE              Actions
DATABASE_URL     ••••••••  [👁]    [✏] [🗑]
API_SECRET       ••••••••  [👁]    [✏] [🗑]
```

#### Danger Zone tab
```
┌─────────────────────────────────────────────────────────┐
│ Delete project                                          │
│ This permanently deletes the project and all its data. │
│ This action cannot be undone.                           │
│                              [Delete project]  ← danger │
└─────────────────────────────────────────────────────────┘
```

Delete confirmation modal:
```
┌────────────────────────────────────────────┐
│ Delete project?                          ✕ │
│                                            │
│ Type the project name to confirm:          │
│ [                                        ] │
│                                            │
│ [Cancel]   [Delete permanently]            │
│                 ↑ red/danger               │
└────────────────────────────────────────────┘
```

---

## Routing summary

Add these routes to `app.jsx`:

| Route | Screen |
|-------|--------|
| `#/projects` | Projects List |
| `#/project/:id` | Project Dashboard |
| `#/project/:id/sql` | SQL Editor (default branch) |
| `#/project/:id/sql/:branch` | SQL Editor (specific branch) |
| `#/project/:id/branches` | Branches |
| `#/project/:id/settings` | Project Settings |

---

## Constraints

- Do NOT modify any of the 20 existing screens
- Use same CSS tokens and component classes as current prototype
- The 3-panel SQL editor left column uses a fixed 240px width; the rest is flex
- Context bar only renders on `/project/*` routes; global pages keep their existing `page-header` layout
- Project selector in sidebar and context bar should stay in sync (selecting one updates the other)
- All mock data goes in `bb/data.js` under a `projects` key
