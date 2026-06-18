# Design C Prototype Handoff to Claude Design

## Context

BigBase admin console is restructuring from a flat, global information architecture to a **project-scoped** architecture (Neon-inspired). This document hands off the Design C approach to the design agent for visual prototyping and iteration.

## Current State

- **Routing:** Flat (/, /data, /sql, /users, /deploy, /functions, /storage, /forge, /cici, /monitoring, /realtime, /settings)
- **Sidebar:** 6 sections (Overview, Build, Data, Auth, Engage, DevOps) → 20 direct page links
- **Scope:** None. Every page is global. Switching between /data and /sql requires re-fetching schema.
- **Problem:** Users working on a database can't easily keep context (project + branch) when navigating between related pages.

## Design C Target

**URL-canonical project/branch scoping** with minimal state machinery.

### Routing Structure (Post-Refactor)

```
/project/:projectId                          → Dashboard (project-scoped)
/project/:projectId/sql/:branch?             → SQL Editor + Table Browser (branch-scoped)
/project/:projectId/branches                 → Branches list (project-scoped)
/project/:projectId/settings                 → Project Settings (project-scoped)

/users  /deploy  /functions  /storage  /forge  /cici  /messaging  /monitoring  /realtime
  ↑ These stay GLOBAL (unchanged)
```

### Layout Changes

**Header (new):**
- Top-left: BigBase logo
- Breadcrumb/context bar: `[Project: my-db ▼]  [Branch: main ▼]  |  Page Title`
- Right: User avatar, Settings, Logout

**Sidebar (refactored into 2 zones):**

1. **Project Zone** (visible only when a project is selected)
   - Dashboard
   - SQL Editor (unified with Table Browser)
   - Branches
   - Settings

2. **Global Zone** (always visible, unchanged)
   - Sites
   - Functions
   - Storage
   - Users
   - Forge
   - CI/CD
   - Messaging
   - Monitoring
   - Realtime

**At app start (no project selected):**
- Show a global dashboard or project picker
- Project Zone hidden
- Global pages available

## Pages to Design

### 1. Global Dashboard (No Project Selected)

**When:** User opens the app or clicks the BigBase logo
**Show:**
- Welcome banner ("Select a project to get started")
- System health panel (global)
- Stat cards: Sites, Functions, Users (global counts)
- "Create a project" CTA card
- Recent activity across all projects
- Project list or quick-access cards

**Components to use:** Card, Button, Badge, Icon, EmptyState, MetricCard

---

### 2. Project Dashboard (Project Selected)

**When:** User selects a project from the selector
**URL:** `/project/p_123`
**Show:**
- Project header: name, status, branch count, connection string (copyable)
- Quick stats row: tables count, rows total, storage used, queries today
- Connection string card (copyable, with format toggle: psql / URI / env var)
- Recent queries list (click to open in SQL editor)
- Branch list (quick branch cards: name, status, last active, size)
- "Open SQL Editor" CTA

**Components to use:** Card, Badge, StatusBadge, Button, Icon, PageHeader, MetricCard, ChoiceCard

---

### 3. Unified SQL + Data Page

**When:** User selects SQL Editor from sidebar (with project selected)
**URL:** `/project/p_123/sql/main` (main is default branch)
**Layout:** 3-panel
- **Left panel (240px):** Schema tree (tables, columns with types). Click to insert. Right-click menu.
- **Top-right panel:** SQL editor (monospace, resizable). Tab bar for multiple queries. [Run] [Save] [Format] buttons.
- **Bottom-right panel:** Results table. Tabs: Results / Messages / Explain. Inline cell editing. Pagination. Export buttons.

**Components to use:** Card, Input, Tabs, Button, Badge, Icon, Modal (for confirm actions), RequestChart (for perf metrics)

---

### 4. Branches Page

**When:** User clicks Branches in project zone
**URL:** `/project/p_123/branches`
**Show:**
- List or cards of all branches for the project
- Each card: name, status, last commit, last active time, size
- [Switch] button (changes URL to /project/p_123/sql/branch-name)
- [Create branch] button (opens modal)
- [Delete] action (with confirm)

**Components to use:** Card, Button, Badge, StatusBadge, Icon, Modal, EmptyState

---

### 5. Project Settings

**When:** User clicks Settings in project zone
**URL:** `/project/p_123/settings`
**Tabs:**
- **General:** project name, description, delete project button
- **Database:** connection pool size, compute size (if applicable)
- **Environment Variables:** key-value editor (add/edit/delete pairs)
- **Branches:** branch protection rules
- **Danger Zone:** delete project, reset database

**Components to use:** Input, Card, Button, Tabs, Modal (confirm destructive actions), Badge, EnvVarEditor, PageHeader

---

### 6. Project Selector & Branch Selector (Header)

**Location:** Top bar, left side (after logo)
**Project Selector:**
- Dropdown listing all projects
- Click to select → URL changes to `/project/:projectId`
- Option to "Create new project"

**Branch Selector:**
- Dropdown listing branches of the current project
- Only visible when a project is selected
- Click to select → URL changes to `/project/:projectId/sql/:branch`
- Option to "Create new branch"

**Components to use:** Dropdown/select pattern (use Input + Modal or a custom dropdown), Button, Icon

---

## Design Principles (Design C)

1. **URL is canonical.** When you select a project or branch, the URL updates. Refresh, back button, and deep links all work.
2. **One obvious hook.** Pages use `useProjectScope()` → `{ project, branch, setProject, setBranch, projects, branches }`. The 90% case is simple.
3. **Headers auto-injected.** Pages don't thread project/branch into API calls—`useScopedApi()` does it.
4. **Global pages unchanged.** /users, /deploy, /functions, etc. stay exactly as they are; they don't know about projects.
5. **Backward compat.** Legacy flat URLs (/data, /sql, /settings) redirect to /project/:lastProjectId/...

---

## Visual Style (DO NOT CHANGE)

- **Colors:** Use existing design tokens (CSS variables) from ui/src/styles/tokens.css
- **Components:** Use BigBase components from ui/src/components/ (Button, Card, Input, Tabs, Badge, Modal, Icon, etc.)
- **Spacing, typography, radii, shadows:** Keep current
- **Dark mode:** Maintain current theme

**Goal:** Restructure IA + layouts WITHOUT changing the visual theme or component styling.

---

## Next Steps After Design

1. **Review mockups** in Claude Design
2. **Iterate** on the 6 pages above (fix spacing, label clarity, data density)
3. **Copy code** from the designs → ui/src/pages/ and ui/src/components/
4. **Implement scope module** (types, hooks, context)
5. **Wire routes** and test end-to-end

---

## Handoff Questions for Design Agent

- Does the 3-panel SQL editor layout feel balanced?
- Is the project/branch selector placement clear enough in the header?
- Should the Project Zone / Global Zone separation in the sidebar be more visually distinct?
- On the project dashboard, should the connection string be more prominent?
- Any UX flows unclear? (e.g., switching projects, creating branches)

---

**Designed by:** Claude Code (Design C architecture)  
**Prototype by:** Claude Design  
**Implement by:** BigBase team (TDD, scope module → routing → pages)
