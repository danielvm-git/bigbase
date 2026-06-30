# Update Brief v3 — Fix: Add Missing Pages to Prototype

**Posted to:** claude.ai/design prototype project (BigBase Console — Evolved v2)  
**Purpose:** Close 3 known gaps in the Evolved v2 prototype before e57/e58 work begins  
**Scope:** Surgical additions only — do NOT redesign existing pages or change visual style  

---

## Prompt to post to claude.ai/design

---

I need you to update the BigBase Console - Evolved v2 prototype with 3 specific fixes. Do NOT change anything that's already working — only add the missing items listed below.

**Context: BigBase** is a self-hosted BaaS platform (like Supabase/Appwrite). The admin console is a React SPA with a dark sidebar on the left. The design uses a dark theme with indigo (#4F46E5) as the brand accent. The sidebar has named sections: Overview, Build, Data, Auth, Engage, DevOps, plus a footer.

---

### Fix 1: Add Forge to the DevOps section

**What Forge is:** A fully-shipped git-backed project management tool — issues tracker, kanban board, label management, and wiki. It's a complete product surface, not a sub-feature.

**Where to add it:** In the DevOps sidebar section, between CI/CD and Monitoring.

```
DevOps
  Git Repos        ← already there
  CI/CD            ← already there
  Forge            ← ADD THIS  (icon: git-branch or hammer, route: /forge)
  Monitoring       ← already there
  Realtime         ← ADD THIS (see Fix 2)
```

**Forge page to add (new screen):**

Show a page with:
- **Page header:** "Forge" with a "New Issue" button (primary, right side)
- **Three tabs:** Issues | Board | Wiki
- **Issues tab (default):** A filterable list table of issues
  - Columns: # (issue number), Title, Labels (colored badge chips), Assignee (avatar), Status (Open/Closed badge), Created date
  - Filter bar above: "All issues" dropdown, Labels dropdown, Assignee dropdown, Search input
  - A few sample rows: 
    - #42 "Add rate limiting to auth endpoints" [security] [bug] · Open · 2 days ago
    - #41 "EnvResolver interface design" [architecture] · Open · 3 days ago
    - #38 "Fix deployment log streaming" [deploy] [done] · Closed · 5 days ago
  - Empty state variant (if no issues): icon + "No issues yet" + "Create your first issue" button

**Style:** Use the same table style as the CI/CD page (pipeline runs list). Same Badge component for labels, StatusBadge for Open/Closed.

---

### Fix 2: Add Realtime and Events to the DevOps section

**What Realtime is:** A WebSocket channel manager — shows active channels, connected clients, message throughput.

**What Events is:** A Server-Sent Events (SSE) live stream viewer — shows real-time platform events as they flow through the event bus.

**Where to add them:** In DevOps section, after Forge:

```
DevOps
  Git Repos
  CI/CD
  Forge            ← from Fix 1
  Monitoring
  Realtime         ← ADD THIS (icon: radio or wifi, route: /realtime)
  Events           ← ADD THIS (icon: zap or activity, route: /events)
```

**Realtime page to add (new screen):**

- Page header: "Realtime" with stat chips: "3 channels" · "12 clients connected" · "847 msgs/min"
- Channel list cards (3 example cards):
  - Channel: `presence:lobby` · 5 clients · 42 msgs/min · Status: Active
  - Channel: `updates:project-1` · 3 clients · 18 msgs/min · Status: Active  
  - Channel: `notifications:global` · 4 clients · 7 msgs/min · Status: Idle
- Each card has: channel name (monospace), client count badge, throughput badge, status badge, "View" button
- Empty state: "No active channels. Clients connect to channels via the Realtime WebSocket API."

**Events page to add (new screen):**

- Page header: "Events" with a "Live" green pulsing indicator badge and a "Clear" button
- A terminal-like scrolling log (dark background, monospace font, like the Build Logs component):
  ```
  2026-06-29 14:23:01  auth.user.login          user_id=u_123  method=email
  2026-06-29 14:23:02  deploy.build.started     site_id=s_456  trigger=push
  2026-06-29 14:23:04  storage.file.uploaded    bucket=avatars  size=24KB
  2026-06-29 14:23:07  functions.exec.completed fn_id=fn_789   duration=42ms
  2026-06-29 14:23:08  realtime.channel.joined  channel=presence:lobby  client=c_001
  ```
- Filter bar above the log: Event type dropdown (All / auth / deploy / storage / functions / realtime), Search input
- Auto-scroll toggle (checkbox: "Auto-scroll to latest")

---

### Fix 3: Fix route naming — /deploy → /sites

**What changed:** The "Sites" feature was initially built as "/deploy" but the correct route is "/sites". This is a naming fix in the prototype only.

**Where to change it:**
- Sidebar nav item: change label from "Deploy" or "Sites (Deploy)" → just "Sites", route `/sites`
- Any breadcrumbs that say "/deploy" → change to "/sites"
- Any "Create site" links that point to "/deploy/new" → change to "/sites/new"
- Any page title that says "Deployments" for the list view → change to "Sites"
- The detail page title is already "Site Detail" — keep that

**Do NOT change:** The deployment history within a site, the "Deployments" tab on the site detail page, or any references to "deployment" as an action (deploying is still deploying — only the top-level route/page name changes from "Deploy" to "Sites").

---

## Constraints (do not change these)

- Dark theme — keep all colors, do not introduce light mode on these new pages
- Same sidebar layout and section order as the rest of the prototype
- Same component style: Button, Badge, StatusBadge, Card, Input, Tabs from the existing BigBase component library
- Same spacing and typography (Inter 400/500/600, 14px body, 12px monospace)
- Same accent color: indigo (#4F46E5)
- The "monthly accent" theme system (January through December palettes) — these new pages should use `var(--accent-*)` tokens, not hardcoded colors

---

## Deliverable

Three new screens added to the prototype:
1. Forge page (Issues tab shown by default, Board and Wiki tabs present)
2. Realtime page (channel list cards)
3. Events page (live log stream)

Plus the /deploy → /sites rename applied to sidebar, breadcrumbs, and page titles.

The prototype sidebar DevOps section should now read:
Git Repos · CI/CD · Forge · Monitoring · Realtime · Events
