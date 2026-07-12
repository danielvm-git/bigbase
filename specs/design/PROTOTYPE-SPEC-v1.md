# BigBase Console — Prototype Spec v1

**Purpose:** Build `BigBase Console.html` from scratch. This is the sole authoritative prototype for the current codebase.  
**Source of truth:** `ui/src/` — every screen described here is derived directly from reading the actual source files.  
**Date:** 2026-06-30  

---

## Design System

- **Theme:** Dark only. `--bg-default` ≈ `#0f1117` (near-black), `--bg-surface` ≈ `#181c27` (cards/sidebar), `--bg-subtle` ≈ `#1e2335`
- **Accent:** Indigo. `--brand-500` = `#4F46E5`. Used for active nav items, primary buttons, progress fills, links.
- **Typography:** Inter — 400/500/600 weights. 14px body, 12px utility/muted text. `Fira Code` (monospace) for code, SQL, logs, env keys.
- **Components:** Button (primary/secondary/ghost/danger/link), Card, Badge (success/info/warning/error/neutral/accent), Input, Tabs, Modal, PageHeader, Breadcrumb, EmptyState.
- **Sidebar width:** 220px fixed. Content area: remainder.

---

## Global Layout (all pages except Login)

### Sidebar

```
[B] BigBase                     ← logo + wordmark

Overview
  ● Dashboard

Build
  ○ Sites
  ○ Functions

Data
  ○ Data Studio
  ○ SQL Editor
  ○ Storage

Auth
  ○ Users

Engage
  ○ Messaging

DevOps
  ○ Git Repos
  ○ CI/CD
  ○ Monitoring
  ○ Forge
  ○ Realtime
  ○ Events           ← exists as route, shown here as nav item

─────────────────
APPEARANCE

[🌙 Dark mode]       ← secondary sm button, full width, icon + label
                       label toggles: "Dark mode" / "Light mode"
Accent [● Indigo ▾]  ← ThemePicker: trigger button + popover menu (role=menu)
                       13 accents: Indigo (default) + 12 monthly themes
                       (january teal, february orange, march purple, april green,
                        may lavender, june rainbow, july peach, august silver,
                        september yellow, october pink, november blue, december red)
                       each menu item: color dot + label + ✓ on active

[⚙ Settings]         ← nav link with settings icon

[D]  daniel@gmail.com ← avatar circle (1 letter, accent bg) + email
[Logout]              ← secondary sm button, full width
```

Active nav item: indigo text + indigo left border accent + slightly lighter bg.

### AppFooter (appears below page content, inside the content area, on every page)

```
[B] © 2026 BigBase · MIT License          [Help]  v2.76.15  ·  Built with BigPowers by danielvm-git  ·  GitHub  ·  Changelog
```

- Left: "B" logomark (small, 20px) + muted text "© 2026 BigBase · MIT License"
- Right: `[Help]` secondary-sm button | version string (monospace muted, sourced from the repo `VERSION` file — currently v2.76.15) | separator `·` | "Built with BigPowers by danielvm-git" (BigPowers and danielvm-git are inline links, accent color) | "GitHub" link | "Changelog" link
- Thin top border, minimal padding (8px vertical), spans full content width

---

## Screen 1 — Login (`/login`)

Full-page centered layout, no sidebar.

```
         [B]
      BigBase
  Sign in to continue

  ┌─────────────────┐
  │ Email           │
  └─────────────────┘
  ┌─────────────────┐
  │ Password        │
  └─────────────────┘
               Forgot password?
  [     Sign In     ]   ← primary button, full width

        ─── or ───
  [G] Sign in with Google   ← Google colors, full width link-style button

  Don't have an account?  Register
```

- Card: centered, ~380px wide, padding 32px, rounded, elevated
- "B" logo circle (40px, indigo bg) + "BigBase" h1 + subtitle p in card header
- Email field shows inline error "Email is required" / "Enter a valid email address"
- Password field shows inline error "Password is required" / "Password must be at least 6 characters"
- "Forgot password?" is a ghost/link button aligned right, opens reset view
- Register mode: title "Create your account", button label "Register", toggle says "Already have an account? Sign In"
- Password reset view (separate card state): title "Reset password", subtitle, email input, "Send reset link" primary button; success state shows muted message "If an account exists for {email}, you will receive an email shortly." + "Back to sign in" link

---

## Screen 2 — Dashboard (`/`)

### PageHeader row
```
Welcome back, danielvm          [+ Create site]  ← primary sm button
Here's what's running on your BigBase instance.
```

### GET STARTED card (OnboardingChecklist)

Full-width card:
```
Get started                                         2 / 5
○  Deploy your first site
✓  Create your account
○  Add a serverless function
○  Invite a teammate
○  Explore the SQL editor
```
- Card header: "Get started" (left, medium weight) + "X / Y" counter (right, muted)
- Each step: `○` (incomplete) or `✓` (done) as plain text glyphs, not icons, followed by step label
- No dismiss link

### SystemStatusPanel card

Full-width card:
```
All systems operational                              [HEALTHY]
Live health across 17 components · uptime 1d 4h

  COMPONENTS          CPU                  MEMORY
  17 / 17             0.1%                 3 MB
  all running         [████░░░░░░]         process heap
                      (indigo fill)        [██░░░░░░░░]
                                           (green fill)
                      ACTIVITY
  ─────────────────────────────────────────────────
  🚀 Deploy a1b2c3d4   37m ago    [COMPLETED]
  🚀 Deploy e5f6a7b8   2h ago     [FAILED]
  🚀 Deploy c9d0e1f2   6h ago     [COMPLETED]
```
- Banner: "All systems operational" bold left + HEALTHY badge (success, top-right)
- Subtitle: "Live health across N components · uptime Xd Yh"
- 3 metric tiles below banner (equal width):
  - **COMPONENTS**: large number "17 / 17", sub-label "all running", activity icon top-right of tile
  - **CPU**: large number "0.1%", horizontal progress bar (indigo fill, red if >80%), icon top-right
  - **MEMORY**: large number "3 MB", sub-label "process heap", horizontal progress bar (green fill), icon top-right
- ACTIVITY section below tiles in same card:
  - Section label "ACTIVITY" (12px, uppercase, muted)
  - List of recent deploy events: rocket icon + "Deploy {short-id}" + timestamp right-aligned + status badge right-aligned (COMPLETED/FAILED/RUNNING in all-caps)

### Stat cards row (4 equal cards)

```
[↗] 4        [↗] 0         [↗] 0         [↗] 0
Sites        Functions     Git Repos     Users
```
- Each card: stat icon (rocket/box/git-branch/users) top-left + chevron "›" top-right
- Large number + label below
- Cards are clickable links (navigate to the section)

### Bottom row (2 columns)

**Recent deployments** card (left):
```
Recent deployments                    View all →
─────────────────────────────────────────────
[READY]  a1b2c3d4     main · static   abc1234   37m ago
[FAILED] e5f6a7b8     main · node     def5678   2h ago
[READY]  c9d0e1f2     main · static             6h ago
```
- "View all →" is a link-style button aligned right in card header
- Each row: status badge left | repo id (monospace) | branch · app_type | commit sha (7 chars, monospace, muted) | time ago (muted, right-aligned)

**Jump back in** card (right):
```
Jump back in
─────────────────────────────────────────────
🚀 Deploy a site from GitHub           ›
⬜ Write a SQL query                    ›
📦 Add a serverless function            ›
👥 Invite a teammate                    ›
```
- Each row: icon + label + chevron. Full-width clickable rows.

---

## Screen 3 — Sites List (`/deploy`)

### PageHeader
```
Sites                    [+ Create site]  ← primary sm button
Deploy and host web apps straight from Git.
```
Subtitle is muted paragraph below header.

### Toolbar
```
[all] [production] [preview]      🔍 Search sites
```
- Segmented control (3 pills): all / production / preview
- Search input with search icon prefix, right of segmented control

### Site grid (when sites exist)

Cards in a responsive grid (2–3 cols):
```
┌──────────────────────────┐  [🗑]  ← trash delete button, top-right, ghost
│ my-app                        │
│ github/user/my-app            │
│ [RUNNING]    main             │
│ Last deploy: 37m ago          │
│ my-app.bigbase.local     ↗    │
└──────────────────────────┘
```
- SiteCard: name, full_name from GitHub, status badge, branch, last deploy time, URL link
- Delete button (ghost, danger) overlaid top-right of each card

### BuildCachePanel (below site grid)

Small card at bottom of page:
```
Build Cache
Cache size: 124 MB  ·  Hit rate: 78%  ·  Last hit: 2h ago    [Active]    [Clear cache]
```

### Empty state (no sites)
```
🚀
Create your first site
Connect a Git repository and BigBase builds, deploys, and serves it...
[+ Create site]
```

---

## Screen 4 — Create Site Wizard (`/deploy/new`)

3-step wizard. Breadcrumb: `Sites > Create site`. Header: "Create a new site" + [✕ Cancel] button.

**WizardSteps progress bar:**
```
[●]──────────[○]──────────[○]
Source      Configure    Deploy
```
Active step: filled indigo dot. Future: empty circle.

### Step 1 — Source

```
Where's your code?
Pick a source. We detect the stack and build it for you.

┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ [GH icon]        │  │ [box icon]        │  │ [rocket]          │
│ Connect Git      │  │ Existing BigBase  │  │ Start from       │
│                  │  │ repo              │  │ template         │
│ Deploy from      │  │ Use a repository  │  │ Astro, Next.js   │
│ GitHub. Redeploys│  │ already on this   │  │ & more.          │
│ on push.         │  │ instance.         │  │ [Coming soon]    │
└──────────────────┘  └──────────────────┘  └──────────────────┘
Selected: indigo border + indigo bg tint

When "Connect Git" selected:
┌──────────────────────────────────────┐
│ [GH]                                 │
│ Connect your GitHub account          │
│ Choose which repos BigBase deploys   │
│ from your GitHub account.            │
│ [Authorize GitHub]  ← primary sm    │
└──────────────────────────────────────┘

Or (when connected):
Connected as danielvm-git
🔍 Search repositories…
┌──────────────────────────────────────┐
│ ⌥ danielvm-git/my-site              │
│ ⌥ danielvm-git/blog     [Selected]  │
└──────────────────────────────────────┘
[Manage repository access on GitHub]  ← link
```

Footer: `[← Cancel]` | `[Continue →]` (primary, disabled until selection)

### Step 2 — Configure

```
Configure your site
Review settings, then deploy. URL preview: my-site.bigbase.local

┌─────────────────────────────────────┐
│ Site name *        [my-site       ] │
│ Production branch  [main          ] │
│ Root directory     [./            ] │
└─────────────────────────────────────┘
Source: danielvm-git/my-site
Branch: main
Root: ./
[Stack detected after build] · Node, Go, Python, or static
```

Footer: `[← Back]` | `[Deploy]` (primary)

### Step 3 — Deploy (split layout)

```
Left card:
  ⟳ (spinner while deploying)
  Deploying my-site…
  my-site.bigbase.local · main
  [building]

  → (when done):
  [Open app]
  [View site]  [All sites]

Right panel:
  Build output                    Live ←  pulsing dot
  ─────────────────────────────────────
  → Deployment started (branch: main)
  → Status: building
  → Cloning repository (branch: main)
  → Clone complete
  → Detected app type: static
  → Serving static files
  → Deployed at http://localhost:10001
  (dark terminal bg, monospace, scrollable)
```

---

## Screen 5 — Site Detail (`/deploy/:siteId`)

### Header

```
my-site                               [RUNNING]
← All sites  (link at very bottom of page)
```

### StatusTimeline (between header and tabs)

```
[●]─────────[●]─────────[●]─────────[●]─────────[●]─────────[○]
pending    building   deploying   running    draining    stopped

Health: ✓ Passed (42ms avg, 3 probes)  ← indigo text, small, below timeline
```
- Past steps: indigo dot + indigo line
- Current "running" step: larger indigo dot with animated live pulse + "Live ●" label
- Current "building": spinning animation dot
- Current "deploying"/"draining": yellow pulsing dot
- Failed step: red dot labeled "failed"
- Health line shown when health check data is present

### Tabs

```
[Deployments]  [Logs]  [Domains]  [Previews]  [Env Vars]  [Deploy Keys]  [Cache]  [Manifest]
```

8 tabs. `Previews` added by e65 (preview environments); `Deploy Keys` mirrors
production's SiteDeployKeysTab (name, key prefix, last used, created, copy/revoke).

#### Deployments tab

Table: columns `Status | Branch | Commit | Started | URL | Actions`

```
[RUNNING]  main  abc1234  Jun 30 14:23  http://...  [Rollback] ← danger sm
[STOPPED]  main  def5678  Jun 29 11:01  http://...  [Rollback]
[FAILED]   main  ghi9012  Jun 28 08:44  —           [Rollback]
```

Rollback button (secondary sm, red/danger border, only on non-current deployments).

**Rollback confirmation modal:**
```
┌─────────────────────────────────────────┐
│ Rollback Deployment?                  ✕ │
│                                         │
│ This will stop the current live         │
│ deployment and restore this one.        │
│ The current deployment will be marked   │
│ as rolled_back.                         │
│                                         │
│ Commit: abc1234  ·  Jun 29, 11:01       │
│                                         │
│         [Cancel]  [Confirm Rollback]    │
│                       ↑ red/danger      │
└─────────────────────────────────────────┘
```

#### Logs tab

Terminal-style log viewer (dark bg, monospace, scrollable, full height):
```
[14:23:01]  Step 1/3: Cloning repository
[14:23:02]  Cloned danielvm-git/my-site @ abc1234
[14:23:05]  Step 2/3: Building (npm run build)
[14:23:12]  > my-site@1.0.0 build
[14:23:15]  Build complete: 42 files
[14:23:16]  Step 3/3: Deploying
[14:23:17]  Serving on :10001
```

#### Domains tab (SiteDomainsTab)

```
Custom Domains                          [+ Add domain]

mysite.com                              [VERIFIED]    [Remove]
www.mysite.com                          [PENDING]     [Remove]

Default URL
my-site.bigbase.local                   [ACTIVE]

[Verify domains]  ← secondary sm
```

#### Env Vars tab (SiteEnvVarsTab)

```
Environment Variables                   [+ Add variable]

KEY                   VALUE              Actions
DATABASE_URL          ••••••••  [👁]    [✏] [🗑]
API_SECRET            ••••••••  [👁]    [✏] [🗑]
NODE_ENV              ••••••••  [👁]    [✏] [🗑]
```
- Values masked as "••••••••" with show/hide eye toggle per row
- Empty state: "No environment variables set. Add variables that will be injected at build and runtime."

#### Cache tab (SiteCacheTab / BuildCachePanel)

```
Build Cache                             [Active]
Cache size: 124 MB  ·  Last hit: 2h ago  ·  Hit rate: 78%     [Clear cache]

Cache entries
─────────────────────────────────────────────────────────────────
Key                       Size     Age      Last Used
node_modules/.cache       98 MB    3 days   2h ago
.next/cache               26 MB    3 days   2h ago
─────────────────────────────────────────────────────────────────
Note: Clearing cache forces a full rebuild on next deploy.
```

#### Manifest tab (SiteManifest)

```
App Manifest (bigbase.yaml)

version: 1
framework: node
build:
  command: "npm run build"
start:
  command: "npm run start"
  port: 3000

[Copy]  ← icon button top-right of code block

[Edit Manifest]  ← secondary sm → switches to textarea editor + Save/Cancel
```

No manifest state:
```
No bigbase.yaml found.
Auto-detection active (Framework: node).
[Create bigbase.yaml]  ← primary sm
```

### Bottom of page

```
← All sites   ← link back to /deploy
```

---

## Screen 6 — Data Studio (`/data`)

### PageHeader
```
Data Studio           [Query this]  ← secondary sm, only visible in Schema mode
```

### Layout: 2-column

```
┌──────────────┬────────────────────────────────────────────┐
│ Collections  │                                            │
│ ──────────── │   Select a collection to browse.          │
│ users        │   (muted, when nothing selected)          │
│ posts        │                                            │
│ products     │   ← when selected: ──────────────         │
│ orders       │   [Data]  [Schema]  toggle buttons        │
│ sessions     │                                            │
│              │   Data mode:                              │
│              │   Filter: [field=value...]  Sort: [-field] │
│              │   ┌──────┬──────┬────────────────────┐    │
│              │   │  id  │email │    created_at       │    │
│              │   ├──────┼──────┼────────────────────┤    │
│              │   │  1   │a@b.c │ 2026-01-01 ...     │    │
│              │   │  2   │c@d.e │ 2026-01-02 ...     │    │
│              │   └──────┴──────┴────────────────────┘    │
│              │                                            │
│              │   Schema mode:                             │
│              │   [+ Add column]  [Query this]             │
│              │   Name      Type     Actions               │
│              │   id        integer  [Edit] [Delete]       │
│              │   email     text     [Edit] [Delete]       │
└──────────────┴────────────────────────────────────────────┘
```
- Collections sidebar: "Collections" label + list of collection name buttons; active one highlighted
- Data mode: filter input + sort input above table, full data table below
- Schema mode: disclaimer note "Schema preview only — column changes not persisted until DDL API ships." above table
- "No collections yet." empty state in sidebar

**Add/Edit column modal:**
```
┌───────────────────────┐
│ Add column          ✕ │
│                       │
│ [Column name       ]  │
│ [text ▾           ]  │ ← type select: text/integer/number/boolean
│                       │
│ [Save]  [Cancel]      │
└───────────────────────┘
```

---

## Screen 7 — SQL Editor (`/sql`)

### PageHeader
```
SQL Editor
```

### Layout: vertical

```
┌─────────────────────────────────────────────────────────────┐
│  SELECT name FROM sqlite_master WHERE type = 'table'...     │
│  (8 rows, monospace textarea, dark bg, no border visible,   │
│   code font, line-height 1.5, 8 rows tall)                 │
└─────────────────────────────────────────────────────────────┘
[Run (⌘⏎)]  ← primary button below textarea

── Error (when query fails): ─────────────────────────────────
  no such table: xyz  ← red error text

── Results (when query succeeds): ────────────────────────────
  3 rows returned  ← muted meta
  ┌─────────────┬──────────────┐
  │ name        │ type         │
  ├─────────────┼──────────────┤
  │ users       │ table        │
  │ deployments │ table        │
  │ sessions    │ table        │
  └─────────────┴──────────────┘
```

---

## Screen 8 — Storage (`/storage`)

### PageHeader
```
Storage          [List]  [Grid]  [Refresh]  ← List/Grid toggle buttons
```

### Upload card
```
┌─────────────────────────────────────────────┐
│  [Choose file]  avatar.png             [Upload] │
└─────────────────────────────────────────────┘
```

### File list — List mode (default)
```
┌─────────────┬────────┬───────────────┬──────────────────┬────────────────────┐
│ Name        │ Size   │ Type          │ Uploaded         │ Actions            │
├─────────────┼────────┼───────────────┼──────────────────┼────────────────────┤
│ avatar.png  │ 24 KB  │ image/png     │ Jun 30, 14:23    │ [Download][Delete] │
│ report.pdf  │ 1.2 MB │ application/  │ Jun 29, 09:15    │ [Download][Delete] │
│             │        │ pdf           │                  │                    │
└─────────────┴────────┴───────────────┴──────────────────┴────────────────────┘
```
- Image filenames are clickable (accent color) → opens image preview modal
- Download is a native anchor link, Delete is danger sm button

### File list — Grid mode
Cards in auto-fill grid (160px min):
```
┌────────────┐  ┌────────────┐
│ [thumbnail]│  │ [📄 icon  ]│
│            │  │            │
│ avatar.png │  │ report.pdf │
│ 24 KB      │  │ 1.2 MB     │
└────────────┘  └────────────┘
```

### Image preview modal (on image click)
Full-screen overlay, centered card, image at max 90vw/70vh, Close button in card header.

### Empty state
```
No files uploaded.  ← muted paragraph
```

---

## Screen 9 — Functions List (`/functions`)

### PageHeader
```
Functions                [Refresh]  [Create function]
```

### Create/Edit inline form (when creating):
```
┌─────────────────────────────────────────────────────────────┐
│ New Function                                                │
│ [Name *          ] [JavaScript ▾] [HTTP ▾]                 │
│ [Env JSON ({"KEY":"val"})        ]  [Timeout 30]           │
│ ┌───────────────────────────────────────────────────────┐  │
│ │ // your function code                                 │  │
│ │ export default async function(req, ctx) {             │  │
│ │   return { status: 200, body: 'hello' }               │  │
│ │ }                                                     │  │
│ └───────────────────────────────────────────────────────┘  │
│ [Create]  [Cancel]                                          │
└─────────────────────────────────────────────────────────────┘
```
- Name input, Runtime select (JavaScript only), Trigger select (HTTP/Schedule/Event)
- Schedule input appears when trigger = "Schedule"
- Env JSON input, Timeout number input
- Source code textarea (monospace, 8 rows)

### Function card grid
Cards in a responsive grid:
```
┌──────────────────────────┐  ┌──────────────────────────┐
│ sendWelcomeEmail         │  │ processWebhook           │
│ javascript · http trigger│  │ javascript · http trigger│
│ Created Jan 1, 2026      │  │ Created Jan 5, 2026      │
│ [Open] [Run] [Logs]      │  │ [Open] [Run] [Logs]      │
│ [Edit] [Delete]          │  │ [Edit] [Delete]          │
└──────────────────────────┘  └──────────────────────────┘
```
- Open = primary sm → navigates to detail
- Run = secondary sm → triggers run inline
- Logs = ghost sm → navigates to detail logs tab
- Edit = ghost sm → expands edit form
- Delete = danger sm → confirms delete

### Run result panel (below grid after Run click)
```
Run Result — sendWelcomeEmail
[console log line 1]
[console log line 2]
{"status": 200, "body": "hello"}
```
Dark bg code block, max 400px height, scrollable.

### Empty state
```
No functions yet.  ← muted
```

---

## Screen 10 — Function Detail (`/functions/:id`)

### Breadcrumb + Header
```
Functions > sendWelcomeEmail
sendWelcomeEmail                [Back]
```

### Tabs
```
[Code]  [Triggers]  [Variables]  [Logs]
```

#### Code tab
```
┌──────────────────────────────────────────────────────────┐
│ export default async function(req, ctx) {                │
│   const { email } = req.body                             │
│   await ctx.email.send({ to: email, ... })               │
│   return { status: 200 }                                 │
│ }                                                        │
│ (16-row monospace textarea, code-textarea style)         │
└──────────────────────────────────────────────────────────┘
[Save code]  ← sm button
```

#### Triggers tab
```
Trigger: http
Runtime: javascript  ·  Timeout: 30s
```
When trigger = schedule: shows Schedule: `*/5 * * * *`

#### Variables tab
```
┌────────────────────────────────────────────────┐
│ {                                              │
│   "SMTP_HOST": "smtp.example.com",             │
│   "SMTP_PORT": "587"                           │
│ }                                              │
│ (8-row monospace textarea, editable JSON)      │
└────────────────────────────────────────────────┘
[Save variables]  ← sm button
```
Inline error if JSON is invalid.

#### Logs tab
FunctionLogsPanel: scrollable log table with timestamp, level badge (info/warning/error), message. "View in Monitoring →" link at bottom.

---

## Screen 11 — Users (`/users`)

### PageHeader
```
Users                    [Refresh]
```

### Table
```
┌──────┬─────────────────────┬──────────────────────┬─────────┐
│ ID   │ Email               │ Created              │ Actions │
├──────┼─────────────────────┼──────────────────────┼─────────┤
│ 1    │ daniel@gmail.com    │ Jan 1, 2026, 12:00   │[Delete] │
│ 2    │ team@company.com    │ Jan 5, 2026, 09:00   │[Delete] │
└──────┴─────────────────────┴──────────────────────┴─────────┘
```
Delete = danger sm.

### Empty state
```
No users found.  ← muted
```

---

## Screen 12 — Messaging (`/messaging`)

### PageHeader
```
Messaging                [Open sample template]  [Refresh]
```

### Page tabs
```
[Templates]  [Send test]  [History]
```

#### Templates tab

Grid of template rows (each row is a link → detail page):
```
┌──────────────────────────────────────────────────────────────┐
│ Welcome Email               [email]  [active]  Jun 30, 2026  │
│ Password Reset              [email]  [active]  Jun 29, 2026  │
│ Order Confirmation          [email]  [draft]   Jun 28, 2026  │
│ SMS Verification            [sms]    [active]  Jun 27, 2026  │
└──────────────────────────────────────────────────────────────┘
```
- template-row: full-width row with: name | type badge (neutral) | status badge (success=active, neutral=draft) | date

#### Send test tab

Channel sub-tabs: `[EMAIL]  [SMS]  [PUSH]`

Email form:
```
[To                  ]
[Subject             ]
[Body                ]
 (4-row textarea)
[Send Email]  ← primary
```

SMS form:
```
[To (phone)          ]
[Message             ]
 (3-row textarea)
[Send SMS]
```

Push form:
```
[Device Token        ]
[Title               ]
[Body                ]
[Send Push]
```

#### History tab

Section heading "Outbound history"
```
┌──────────┬─────────────────────┬─────────────────┬──────────┬───────────────────┐
│ Channel  │ To                  │ Subject         │ Status   │ Sent              │
├──────────┼─────────────────────┼─────────────────┼──────────┼───────────────────┤
│ [email]  │ user@example.com    │ Welcome!        │ sent     │ Jun 30, 14:23     │
│ [sms]    │ +15551234567        │ —               │ sent     │ Jun 29, 09:15     │
└──────────┴─────────────────────┴─────────────────┴──────────┴───────────────────┘
```
Channel column: colored badge chip (email=blue, sms=green, push=purple)

---

## Screen 13 — Messaging Detail (`/messaging/:id`)

### Breadcrumb + Header
```
Messaging > Welcome Email
Welcome Email                    [Back]  [Send test]
```
Subtitle: "Preview only — edits are not saved until the template API ships." (muted)

### Tabs
```
[Editor]  [Preview]
```

#### Editor tab
```
[Subject: Welcome to {{app_name}}!       ]
[Body:                                   ]
[ Hi {{user_name}},                      ]
[ Welcome to {{app_name}}! You can       ]
[ access your account at...              ]
[ (8-row textarea)                       ]
[Variables: user_name, app_name, code    ]
Type: email  ·  Status: active
```

#### Preview tab — 2 columns
```
Left card: Rendered preview
Subject: Welcome to BigBase!

Hi Alex,
Welcome to BigBase! You can
access your account at...

Right card: Variables
{{user_name}}
{{app_name}}
{{reset_link}}
{{code}}
```

---

## Screen 14 — Git Repos (`/repos`)

### PageHeader
```
Git Repos                [Refresh]  [New Repo]
```

### New repo form (inline card, when New Repo clicked):
```
┌─────────────────────────────────────────────────────────────┐
│ [Name *          ] [Description          ] [□ Private] [Create] │
└─────────────────────────────────────────────────────────────┘
```
Horizontal form row.

### Table
```
┌──────────────┬────────┬──────────────────┬────────────────────┬─────────┐
│ Name         │ Branch │ Description      │ Created            │ Actions │
├──────────────┼────────┼──────────────────┼────────────────────┼─────────┤
│ my-app       │ main   │ My web app       │ Jun 30, 14:23      │[Delete] │
│ api-service  │ main   │ —                │ Jun 29, 09:15      │[Delete] │
└──────────────┴────────┴──────────────────┴────────────────────┴─────────┘
```
Name column: `<code>` styled (monospace). Delete = danger sm.

### Empty state
```
No repos yet.  ← muted
```

---

## Screen 15 — CI/CD (`/cici`)

### Empty state (no repo selected yet)
```
CI/CD
Select a repo to view workflows.
[Select repo... ▾]  ← select dropdown
```

### PageHeader (when repo selected)
```
CI/CD              [my-app ▾]  [New Workflow]
                   ↑ repo selector
```

### New Workflow form (inline card)
```
┌─────────────────────────────────────────────────────────────┐
│ New Workflow                                                 │
│ [Workflow name *            ]                               │
│ [YAML config:               ]                               │
│  name: build                                               │
│  on: push                                                  │
│  jobs:                                                     │
│    build:                                                  │
│      steps:                                                │
│        - run: npm install                                  │
│  (10-row code textarea)                                    │
│ [Save]                                                     │
└─────────────────────────────────────────────────────────────┘
```

### Tabs
```
[Workflows]  [Runs]
```

#### Workflows tab
```
Workflows
┌────────────────────────┬─────────┐
│ Name                   │ Actions │
├────────────────────────┼─────────┤
│ build                  │ [Run]   │
│ deploy                 │ [Run]   │
└────────────────────────┴─────────┘
```
Name in monospace code style.

#### Runs tab
```
Runs
┌──────────┬────────┬───────────┬──────────────────┬──────────────────┬──────┐
│ ID       │ Event  │ Status    │ Started          │ Finished         │ Logs │
├──────────┼────────┼───────────┼──────────────────┼──────────────────┼──────┤
│ a1b2c3d4 │ manual │ [success] │ Jun 30, 14:23    │ Jun 30, 14:25    │[Logs]│
│ e5f6a7b8 │ push   │ [failed]  │ Jun 29, 09:15    │ Jun 29, 09:16    │[Logs]│
└──────────┴────────┴───────────┴──────────────────┴──────────────────┴──────┘
```
- ID: monospace 8-char truncated
- Status: Badge with statusBadgeVariant (success=green, failed=red, running=yellow)
- Logs button → expands log panel below table

**Expanded log panel (below table):**
```
Logs — a1b2c3d4
[build]
npm install
added 142 packages in 8s
npm run build
Build complete
[deploy]
Deploying to production...
Done.
```
Dark bg code block.

---

## Screen 16 — Monitoring (`/monitoring`)

### PageHeader
```
Monitoring
```

### Tabs
```
[Overview]  [Host]  [Logs]  [Alerts]
```

#### Overview tab

**System section:**
```
System
┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐
│ CPU        │  │ HEAP MB    │  │ GOROUTINES │  │ UPTIME     │
│ 1.2%       │  │ 3.8        │  │ 28         │  │ 1d 4h      │
└────────────┘  └────────────┘  └────────────┘  └────────────┘
```
4 equal stat cards, large numbers.

**Requests section (below System):**
```
Requests
┌────────────────────┐  ┌────────────────────┐
│ 548 TOTAL          │  │ 73.0ms AVG LATENCY │
└────────────────────┘  └────────────────────┘

┌──────────────────────────┬───────┬─────────────┬────────────────────────────────┐
│ ENDPOINT                 │ COUNT │ AVG LATENCY │ STATUS CODES                   │
├──────────────────────────┼───────┼─────────────┼────────────────────────────────┤
│ /api/auth/login          │ 83    │ 102.2ms     │ [200: 83]                      │
│ /api/auth/me             │ 37    │ 0.1ms       │ [200: 3] [401: 34]             │
│ /api/auth/oauth/google   │ 17    │ 0.1ms       │ [404: 17]                      │
│ /api/collections/jogos   │ 192   │ 6.2ms       │ [200: 190] [201: 1] [401: 1]   │
└──────────────────────────┴───────┴─────────────┴────────────────────────────────┘
```
- ENDPOINT column: left-aligned, monospace path
- STATUS CODES: multiple inline chip badges per row
  - 2xx: green chip (success)
  - 3xx: neutral chip
  - 4xx: yellow/orange chip (warning)
  - 5xx: red chip (error)
  - Format: `{code}: {count}` inside chip

#### Host tab

**CPU section:**
```
CPU
  [Donut gauge: 90px, arc fill blue #3b82f6, "34.2%" in center]
  CPU % — last 30 samples
  [Sparkline chart: 240px wide × 48px tall, blue line, last 30 values]
```

**Memory section:**
```
Memory
  [Donut gauge: 90px, arc fill green #22c55e, "3.8 MB" in center]
  Heap MB — last 30 samples
  [Sparkline: green line]
```

**Network section:**
```
Network
┌─────────────────────────┐  ┌─────────────────────────┐
│ Bytes In (cumulative)   │  │ Bytes Out (cumulative)  │
│ 1.2 GB                  │  │ 847 MB                  │
│ [small green sparkline] │  │ [small red sparkline]   │
└─────────────────────────┘  └─────────────────────────┘
```

**Disk section:**
```
Disk
Disk usage   [████████████████████░░░░░░░░░░░░░]  45.2 GB / 200 GB
             ↑ full-width horizontal bar, indigo fill
```

#### Logs tab

```
[🔍 Filter logs...        ]  ← search input

┌─────────────┬─────────┬────────────────────────────────────────────┐
│ Timestamp   │ Level   │ Message                                    │
├─────────────┼─────────┼────────────────────────────────────────────┤
│ 14:23:01    │ [info]  │ deploy started for site s_456              │
│ 14:23:04    │ [warn]  │ slow response on /api/data (1.2s)          │
│ 14:22:55    │ [error] │ auth token expired for user u_789          │
└─────────────┴─────────┴────────────────────────────────────────────┘
```
- Level shown as badge variant: info (neutral/blue), warning (amber), error (red)

#### Alerts tab

```
Alerts                                                [+ Create alert]

┌────────────────────────────────────────────────────────────────────┐
│ High CPU          cpu_percent > 80         [Active ●]              │
│ Memory pressure   memory_mb > 512          [Inactive ○]            │
└────────────────────────────────────────────────────────────────────┘

Each alert card:
  Name (bold) · metric operator threshold · toggle switch right-aligned
```

**Create alert form (below list, when button clicked):**
```
┌────────────────────────────────────────────────────┐
│ [Alert name               ]                        │
│ [cpu_percent ▾]  [gt ▾]  [80        ]             │
│                  ↑operator  ↑threshold              │
│ [Create]  [Cancel]                                 │
└────────────────────────────────────────────────────┘
```
Metric select: cpu_percent / memory_mb / request_count / error_rate
Operator select: gt / lt / eq

---

## Screen 17 — Forge (`/forge`)

### PageHeader
```
Forge
[Select repo... ▾]  ← dropdown at top, required to show issues/board
```

### Tabs (when repo selected)
```
[Issues]  [Board]  [Wiki]
```
Note: Wiki tab exists as a label but shows "Coming soon" or empty content.

#### Issues tab

```
Issues                             [+ New Issue]

[Status ▾]  [Label ▾]  [🔍 Search issues...]

┌─────────────────────────────────────┬──────────────────────┬────────┬──────────────────┐
│ Title                               │ Labels               │ Status │ Created          │
├─────────────────────────────────────┼──────────────────────┼────────┼──────────────────┤
│ Fix login redirect loop             │ [bug] [auth]         │ [Open] │ Jun 29, 2026     │
│ Add dark mode to settings           │ [enhancement]        │ [Open] │ Jun 28, 2026     │
│ Update README                       │ [docs]               │[Closed]│ Jun 27, 2026     │
└─────────────────────────────────────┴──────────────────────┴────────┴──────────────────┘
```
- Label badges: colored chips (bug=red, enhancement=blue, docs=gray, auth=purple, etc.)
- Status: Open (success/green badge), Closed (neutral badge)
- Clicking a row → opens issue detail below or in-place

**Issue detail (when row clicked):**
```
← Back to issues

Fix login redirect loop                          [Open ▾]
Labels: [bug] [auth]

## Description
After login, user is redirected back to /login instead of /.
Steps to reproduce: 1. Navigate to /login...

─── Comments ─────────────────────────────────────────────
[D]  daniel@gmail.com · Jun 29, 2026
  Confirmed. Happens when token is missing from cookie.

[D]  daniel@gmail.com · Jun 30, 2026
  Fixed in commit abc1234. Will deploy in next release.

─── Add comment ───────────────────────────────────────────
[                                                          ]
[                 (3-row textarea)                         ]
[Add comment]  ← primary sm
```

**New issue form (when + New Issue clicked):**
```
Title: [                             ]
Description: [                       ]
             [  (4-row textarea)     ]
Labels: [bug ×] [+ add label]
[Create issue]  [Cancel]
```

#### Board tab

4-column kanban:
```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  OPEN        │  │  IN PROGRESS │  │  REVIEW      │  │  CLOSED      │
│              │  │              │  │              │  │              │
│ ┌──────────┐ │  │ ┌──────────┐ │  │              │  │ ┌──────────┐ │
│ │Fix login │ │  │ │Add dark  │ │  │              │  │ │Update    │ │
│ │redirect  │ │  │ │mode      │ │  │              │  │ │README    │ │
│ │[bug][auth│ │  │ │[enhance] │ │  │              │  │ │[docs]    │ │
│ └──────────┘ │  │ └──────────┘ │  │              │  │ └──────────┘ │
└──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘
```
- Each column: column label (uppercase, muted) + issue cards stacked
- Issue card: title (truncated) + label badge chips
- Clicking a card → issue detail view (same as Issues tab detail)

---

## Screen 18 — Realtime (`/realtime`)

### PageHeader
```
Realtime
5 active connections  ·  3 rooms   ← success badge + neutral badge in header area
```

### Connection list
```
┌──────────────┬───────────────────────────────────────┬──────────────────┐
│ User ID      │ Rooms                                 │ Connected        │
├──────────────┼───────────────────────────────────────┼──────────────────┤
│ u_abc123     │ [presence:lobby] [notifications:global]│ 5m ago           │
│ u_def456     │ [presence:lobby]                      │ 12m ago          │
│ u_ghi789     │ [game:room-42]                        │ 1h ago           │
└──────────────┴───────────────────────────────────────┴──────────────────┘
```
- User ID: monospace
- Rooms: small monospace badge chips per room
- Live-updating via polling

### Empty state
```
No active connections.
WebSocket connections will appear here when clients connect.
```

---

## Screen 19 — Events (`/events`)

### PageHeader
```
Event Bus
Live stream of internal event bus emissions.       [● Live]  [Clear]
```
- "● Live" pulsing green badge (success) when SSE connected; yellow "Connecting…" badge when connecting
- [Clear] is secondary sm button

### Filter bar (below header)
```
[Event type: All ▾]   [🔍 Filter events...]
```
Event type options: All / auth / deploy / storage / functions / realtime

### Terminal log block

Dark bg (`--bg-subtle`), monospace `Fira Code`, rounded corners, full width, ~500px tall, scrollable:
```
2026-06-30 14:23:08  realtime.channel.joined  channel=presence:lobby  client=c_001
2026-06-30 14:23:07  functions.exec.completed fn_id=fn_789             duration=42ms
2026-06-30 14:23:04  storage.file.uploaded    bucket=avatars           size=24KB
2026-06-30 14:23:02  deploy.build.started     site_id=s_456            trigger=push
2026-06-30 14:23:01  auth.user.login          user_id=u_123            method=email
```
Newest at top. Each row: `[timestamp]  [hook_name padded to ~28 chars]  [key=value pairs]`

**Auto-scroll checkbox below log:**
```
☑ Auto-scroll to latest
```

### Empty state (no events yet)
```
Waiting for events…
Events will appear here as they flow through the BigBase event bus.
```
Icon + title + muted subtitle, centered.

---

## Screen 20 — Settings (`/settings`)

### PageHeader
```
Settings
```

### Tabs
```
[Account]  [Workspace]  [Billing]
```

#### Account tab

Card "Account":
```
Email               daniel@gmail.com         (read-only, muted value)
Name                Daniel                   (read-only, muted value)
Two-factor auth     [Not configured]  ← info/neutral badge

──────────────────────────────────────────────────────────
Change password
[Current password              ]
[New password                  ]
[Save]  ← sm primary
```

#### Workspace tab

Card "Workspace":
```
Workspace name  [My BigBase                ]  [Save]
```

Card "Members (2)":
```
daniel@gmail.com          [admin]   ← accent/indigo badge
team@company.com          [member]  ← neutral badge
```

#### Billing tab

Card "Billing":
```
Current plan        [Free]  ← success badge
Renews              2026-07-30

──────────────────────────────────────────────────────────
Usage this month
Functions: 1,240     Storage: 48 MB     Sites: 3
```

---

## Constraints

- All screens in a single HTML file with JavaScript navigation between them
- Dark theme only — `--bg-default` near-black, `--fg-primary` near-white
- Indigo accent `#4F46E5` for all highlights, active states, primary buttons
- Interactive: sidebar navigation switches between screens, tabs switch panels, buttons/toggles work
- Same Inter font throughout (400/500/600), 14px body, 12px utility text, Fira Code for code/logs
- Badge colors: success=green, info=blue, warning=amber, error=red, neutral=gray, accent=indigo
- All forms show placeholder states (empty screens should show empty states, not hide the UI)

## Deliverable

A single `BigBase Console.html` file where:
- All 20 screens are navigable via the sidebar
- Every screen matches the exact UI structure from the source code
- Interactive tabs, modals, and form states work
- The file is self-contained — no external dependencies
