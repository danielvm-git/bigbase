# BigBase Prototype Enhancement Prompt for Claude Design

**Date**: June 1, 2026  
**Status**: Analysis complete - Ready for design enhancement  
**Both Design Files Analyzed**: 
- Console Prototype (XuXY9-y0hJ4jHEp20TARoA)
- Design System (dBcuYCxVNNHJfYNvsweekQ)

---

## Executive Summary

You have two complementary prototypes:

1. **BigBase Console.html** - 8-screen interactive prototype focusing on user journeys (Sites, SQL, Storage, Users, Repos, CI/CD, Monitoring, Dashboard)
2. **BigBase Design System** - Comprehensive design system with components, tokens, and Appwrite-inspired patterns

**Analysis shows**: The Console prototype is feature-rich but the Design System prototype is the more foundational, Appwrite-aligned design. There are gaps between them that should be unified, plus missing screens based on the codebase and release plan.

---

## What You Have (Current State)

### Console Prototype Coverage
- ✅ 8 screens: Dashboard, Sites, SQL Editor, Storage, Users, Git Repos, CI/CD, Monitoring
- ✅ Create Site wizard (3-step)
- ✅ System status panel
- ✅ Dark/light theme toggle
- ✅ Responsive design
- ✅ Version footer

### Design System Coverage
- ✅ Appwrite-inspired information architecture
- ✅ Complete design tokens (colors, typography, spacing, shadows)
- ✅ 30+ component definitions with states
- ✅ Rich component preview pages
- ✅ React source components (TypeScript)
- ✅ 12+ preview cards for design tokens
- ✅ Focus on Sites journey (hero flow)

### Codebase Reality (Your Implementation)
```
ui/src/
├─ components/    (12 built: Button, Input, Card, Badge, Tabs, 
│                 ChoiceCard, SiteCard, PageHeader, etc.)
├─ pages/         (16 pages: Deploy, Sites, Create, Functions, 
│                 Monitoring, CI/CD, Users, etc.)
├─ Layout.tsx     (Sidebar navigation)
└─ index.css      (32KB token system, Appwrite-inspired)
```

**Status**: Pages exist but many are placeholder/basic implementations. Design system tokens are in place but not all components are fully styled.

---

## Gaps Identified

### Gap 1: Information Architecture Mismatch
**Issue**: Console prototype uses flat sidebar; Design System proposes grouped sidebar.

**Console Sidebar**: Flat list (Dashboard, Sites, SQL, Storage, Users, Repos, CI/CD, Monitoring)

**Design System Sidebar** (Recommended):
```
BigBase
├─ OVERVIEW → Dashboard
├─ BUILD → Sites, Functions
├─ DATA → Data Studio, SQL, Storage
├─ AUTH → Users
├─ DEVOPS → Git Repos, CI/CD, Monitoring
```

**Your Codebase**: Currently has sections but not fully aligned to either pattern.

**ACTION**: Design System's IA is more Appwrite-aligned and recommended. Console should adopt it.

---

### Gap 2: Screen Coverage

**Console Prototype Includes** (8):
- Dashboard ✅
- Sites (list + create + detail) ✅
- SQL Editor ✅
- Storage ✅
- Users ✅
- Git Repos ✅
- CI/CD Pipelines ✅
- Monitoring ✅

**Design System Focuses On** (4):
- Dashboard
- Sites (hero journey)
- Login
- Data Studio (secondary)

**Codebase Has Stubbed Out** (16):
- CiciPage (CI/CD)
- CreateSitePage ✅
- DashboardPage ✅
- DataStudioPage (basic)
- DeployPage (being refactored to Sites)
- ForgePage (build system)
- FunctionsPage (basic)
- GitReposPage ✅
- LoginPage ✅
- MessagingPage (missing from both designs)
- MonitoringPage ✅
- SiteDetailPage ✅
- SqlEditorPage ✅
- StoragePage ✅
- UsersPage ✅
- NotFoundPage

**ACTION**: Console prototype has good breadth. Design System should expand secondary screens (Functions list, Data Studio, Login refinement, Messaging).

---

### Gap 3: Component Consistency

**Console Components** (24 primitives):
- Button, Input, Card, Badge, Avatar, Toast, etc.
- Mostly styled, some incomplete

**Design System Components** (30+):
- More detailed states documented
- Component preview pages
- React source provided

**Codebase Components** (12 built):
- Badge, Button, Card, ChoiceCard, EmptyState, Input, PageHeader, PreviewBanner, SiteCard, SitesSkeleton, Tabs, WizardSteps

**ACTION**: Unify component specs. Design System's preview pages + React source should inform Console styling.

---

### Gap 4: Theme Support

**Current**: Light and dark mode toggle only.

**Requested**: 12-month theme options (January=Teal through December=Red) with specific RGB colors all WCAG 2.1 AA compliant.

**ACTION**: Add theme picker dropdown to Console showing 12 branded month colors + ensure accessibility.

---

### Gap 5: Wizard & Journey Flows

**Console**: Sites wizard works (3-step: Source → Configure → Deploy)

**Design System**: 
- Documents Journey A in detail (first-time deploy)
- Suggests Journey B (returning user)
- Suggests Journey C (login)
- Suggests Journey D (dashboard hub)

**Codebase**: CreateSitePage and SiteDetailPage exist but could use Journey refinement.

**ACTION**: Ensure all journey transitions are smooth, animated, and prevent dead ends.

---

### Gap 6: Screens Missing From Both Designs But In Codebase

- **Forge Page** (build system / container orchestration) — not designed
- **Messaging Page** (email/SMS) — not designed
- **Settings Page** (workspace/account) — only footer mentioned
- **Org/Team Management** — not in scope of current designs

**ACTION**: These are lower priority (per Release Plan EPIC 10: Collaboration). Document as "coming soon" or placeholder screens.

---

### Gap 7: Copy & Microcopy

**Console**: Generic/placeholder labels

**Design System**: Detailed microcopy guide
- "Where's your code?" → GitHub connection
- "Your site is live" → deployment success
- Verb-first buttons ("Create site", not "New site")

**Codebase**: Using placeholder copy, not polished

**ACTION**: Apply Design System's voice & tone guide to all screens.

---

## What Should Change (Prompt for Claude Design)

### Priority 1: Unify & Enhance Both Prototypes

1. **Adopt the Design System's Information Architecture** into the Console prototype
   - Regroup sidebar: BUILD (Sites, Functions), DATA (Data Studio, SQL, Storage), DEVOPS (Repos, CI/CD, Monitoring)
   - Test navigation cross-links (e.g., Sites → Git Repos for source selection)

2. **Expand Console to Show All Major Journeys**
   - Journey A: First-time Sites deploy (already there, polish)
   - Journey B: Returning user / Sites list with filters/search
   - Journey C: Login screen (Design System has this, integrate into Console)
   - Journey D: Dashboard as hub with cross-links

3. **Reconcile Component Styling**
   - Use Design System's component preview pages as the source of truth
   - Apply those exact colors, spacing, states to Console screens
   - Ensure all 24 console components match Design System specs

4. **Add 12 Month-Based Theme Colors**
   - Add theme picker dropdown (next to dark mode toggle) with 12 options:
     - January: Teal (rgb(13, 148, 136))
     - February: Orange (rgb(234, 88, 12))
     - March: Purple (rgb(124, 58, 237))
     - April: Green (rgb(22, 163, 74))
     - May: Lavender (rgb(167, 139, 250))
     - June: Rainbow (gradient)
     - July: Peach (rgb(253, 186, 116))
     - August: Silver/Grey (rgb(156, 163, 175))
     - September: Yellow (rgb(234, 179, 8))
     - October: Pink (rgb(236, 72, 153))
     - November: Blue (rgb(37, 99, 235))
     - December: Red (rgb(220, 38, 38))
   - **WCAG 2.1 AA Compliance Required**: All colors must maintain ≥4.5:1 contrast with text
   - Apply theme to accent color (`--bg-accent`, `--fg-accent`) throughout
   - Persist theme selection (localStorage)

5. **Polish Secondary Screens** (expand Console to match Codebase scope)
   - **Functions page**: List view with status, runtime, trigger type, creation date, "Create function" CTA
   - **Data Studio**: Table browser with schema introspection, column operations
   - **Login**: Centered card with email + password fields, error states, "Continue with Google" button
   - **Messaging**: Message templates list, create/edit template form
   - **Settings**: Account + workspace settings (placeholder acceptable for now)

6. **Apply Microcopy Guide** from Design System
   - Rewrite all button labels to be verb-first and specific
   - Update empty states with inviting titles + payoff descriptions
   - Refine error messages (no stack traces, actionable)
   - Use consistent status labels (Ready / Building / Failed / Pending)

7. **Add Missing UX Details**
   - Smooth wizard step transitions (animate progress)
   - Toast notifications for key actions (deploy started, site created, etc.)
   - Skeleton loaders on list pages while "fetching"
   - Breadcrumb navigation on detail pages
   - Quick-action context menus (⋮ button on cards/rows)

### Priority 2: System Design Refinement

1. **Document Design System Updates**
   - Add the 12 month themes to the color token documentation
   - Add WCAG compliance annotations
   - Document theme switching behavior (CSS variable override)

2. **Component Spec Completeness**
   - Ensure all 24 console components have full state documentation
   - Add loading states, error states, disabled states explicitly
   - Document responsive behavior for mobile

3. **Responsive Refinement** (desktop → tablet → mobile)
   - Sidebar collapse at 768px (icon rail)
   - Table horizontal scroll on mobile
   - Modal/bottom sheet forms on mobile (not full page)
   - Touch target sizes ≥44px

### Priority 3: Implementation Readiness

1. **Export Component Specs**
   - Create a "Specs" section showing exact spacing, colors, fonts for each component
   - Include CSS variable names that match the codebase

2. **React Component Stubs**
   - Ensure Design System's React components are complete and exportable
   - Provide TypeScript interfaces for props

3. **Dark Mode + Theme Switching**
   - Document CSS class/attribute approach (e.g., `[data-theme="dark"]`, `[data-theme="June"]`)
   - Show how to implement theme persistence in React

---

## Release Plan Alignment

### EPIC 1: Console UI Prototype Implementation (NOW)
**Status**: Designs should be production-ready after enhancements.

**Current**: Console prototype + Design System exist but fragmented.

**Needed**: Unified prototype with all journeys, consistent styling, month themes, full screen coverage.

### EPIC 2: Authentication & Authorization (Q3)
**Designs Missing**: OAuth flow details, magic link flow, session management screens. Current Login screen is minimal.

**Suggest**: Add login refinement, password reset, 2FA screens (can be placeholder for now).

### EPIC 3: Deployment Engine & CI/CD (Q3-Q4)
**Designs Have**: Sites wizard, deployment history, redeploy flow. Good coverage.

**Suggest**: Show build log streaming in detail, advanced build settings, domain management screen (linked from Site detail).

### EPIC 4: Database & Query Engine (Q4)
**Designs Have**: SQL Editor page, basic Data Studio.

**Suggest**: Expand Data Studio to show table browser, column operations (Add/Edit/Delete), schema explorer tree.

### EPIC 5: Object Storage (Q4)
**Designs Have**: Storage page with bucket list, file list.

**Suggest**: Show upload progress, file preview, bucket policy UI (can be modal/form).

### EPIC 6: Functions (Q4)
**Designs Have**: Functions page listed but minimal.

**Suggest**: Function editor, environment variables, logs/execution history, create/deploy flow.

### EPIC 7: Monitoring (Q1 2027)
**Designs Have**: Monitoring page with component status, metrics.

**Suggest**: Add metric graphs, alert rules, log filtering UI.

---

## Detailed Prompt for Claude Design

---

### **PROMPT TO USE IN CLAUDE.AI/DESIGN**:

---

**You have two BigBase prototypes that need to be unified and enhanced. Your mission: make them production-ready for engineering handoff.**

**What you're working with:**

1. **BigBase Console.html** (`https://api.anthropic.com/v1/design/h/XuXY9-y0hJ4jHEp20TARoA?open_file=BigBase+Console.html`)
   - 8 fully-designed screens (Dashboard, Sites, SQL, Storage, Users, Repos, CI/CD, Monitoring)
   - Mock data, dark/light mode, responsive
   - Missing: organized IA, secondary screens, polish

2. **BigBase Design System** (`https://api.anthropic.com/v1/design/h/dBcuYCxVNNHJfYNvsweekQ`)
   - Appwrite-inspired IA (BUILD / DATA / DEVOPS grouping)
   - 30+ components with states
   - Design tokens, microcopy guide, accessibility specs
   - Missing: secondary screens, theme variety, integration with Console

**Your codebase reality:**
- React + TypeScript UI with 16 pages, 12 built components, 32KB token system
- Pages exist but need consistent styling + copy refinement
- Ready for engineering to begin EPIC 1 (UI implementation)

---

**What to do:**

### Part A: Unify the Architecture

1. **Adopt Design System's Information Architecture** in the Console
   - Reorganize Console's sidebar to match Design System grouping:
     ```
     OVERVIEW → Dashboard
     BUILD → Sites, Functions
     DATA → Data Studio, SQL Editor, Storage
     AUTH → Users
     DEVOPS → Git Repos, CI/CD, Monitoring
     ```
   - Update navigation links and cross-references
   - Test wizard transitions (e.g., Sites → GitHub connection, Git Repos → Create site button)

2. **Expand Console with Secondary Screens**
   - Add **Login** (from Design System, refine + polish)
   - Add **Functions list** (mirror Sites list pattern: cards with status, create CTA, detail nav)
   - Add **Data Studio** (table browser, schema explorer, column operations)
   - **Messaging** list (email template list, create button, detail view)
   - **Settings** screen (basic: dark toggle, theme picker, account info — placeholder for now)

3. **Ensure Component Consistency**
   - Make sure Console's Button, Input, Card, Badge, etc. exactly match Design System's component preview pages
   - Use Design System's color tokens, spacing, shadows, motion
   - Test all component states: default, hover, focus, active, disabled, loading, error

---

### Part B: Add 12 Month-Based Themes with WCAG 2.1 AA Compliance

Create a **Theme Selector** (dropdown next to dark-mode toggle) with 12 theme options. Each theme swaps the accent color (`--bg-accent`, `--fg-accent`, `--border-accent`, `--brand-*` tokens).

**Month themes** (exact RGB provided; all must pass WCAG 2.1 AA for text contrast):

| Month | Name | RGB | Target Contrast |
|-------|------|-----|-----------------|
| January | Teal | rgb(13, 148, 136) | ≥4.5:1 on white, ≥7:1 on gray-100 |
| February | Orange | rgb(234, 88, 12) | ≥4.5:1 on white, ≥7:1 on gray-100 |
| March | Purple | rgb(124, 58, 237) | ≥4.5:1 on white, ≥7:1 on gray-100 |
| April | Green | rgb(22, 163, 74) | ≥4.5:1 on white, ≥7:1 on gray-100 |
| May | Lavender | rgb(167, 139, 250) | ≥4.5:1 on white, ≥7:1 on gray-100 |
| June | Rainbow | linear-gradient(to right, rgb(239, 68, 68), rgb(245, 158, 11), rgb(16, 185, 129), rgb(59, 130, 246), rgb(139, 92, 246)) | Adjust text shadow for readability |
| July | Peach | rgb(253, 186, 116) | ≥4.5:1 on white, ≥7:1 on gray-100 |
| August | Silver/Grey | rgb(156, 163, 175) | ≥4.5:1 on white, ≥7:1 on gray-100 |
| September | Yellow | rgb(234, 179, 8) | ≥4.5:1 on white, ≥7:1 on gray-100 |
| October | Pink | rgb(236, 72, 153) | ≥4.5:1 on white, ≥7:1 on gray-100 |
| November | Blue | rgb(37, 99, 235) | ≥4.5:1 on white, ≥7:1 on gray-100 |
| December | Red | rgb(220, 38, 38) | ≥4.5:1 on white, ≥7:1 on gray-100 |

**Implementation Notes:**
- Add theme selector to the sidebar footer (next to dark-mode toggle)
- Persist theme choice to localStorage (key: `bigbase-theme`)
- Apply theme by setting CSS custom property: `--brand-500: [rgb value]`, `--brand-600: [derived shade]`, `--brand-700: [derived shade]`
- For **Rainbow** theme, apply gradient to accent areas (buttons, status badges, links) with text shadow for readability
- All other accent-colored elements (focus rings, active states, links) automatically update via token cascade
- Test all colors with WCAG 2.1 AA contrast checker; flag any that don't meet 4.5:1 minimum

**Dark Mode + Theme**: Themes should work in both light and dark mode. In dark mode, adjust semantic tokens (`--fg-on-accent`, `--border-accent`) to maintain contrast.

---

### Part C: Polish Journeys & Copy

1. **Journey A**: First-time Sites deploy (already exists, refine)
   - Smooth wizard step transitions (CSS animations)
   - Clear progress indicator
   - "Continue" button only enabled when required fields filled
   - Toast notification on deploy success

2. **Journey B**: Returning user / Sites list
   - Cards show status at a glance (badge + subtle color wash)
   - Filter/search bar above list
   - Hover lift on cards, quick-action menu (⋮)
   - Click card → Site detail with Redeploy button

3. **Journey C**: Login & session
   - Centered login card (email + password + "Continue with Google" option)
   - Inline field validation (red border + error message)
   - Password reset link
   - Session timeout graceful degradation

4. **Journey D**: Dashboard as hub
   - Health banner + stat cards (uptime, deployment count, etc.)
   - **Cross-linked cards** that navigate to Sites, Functions, Repos, Users
   - Recent activity feed
   - "Get started" shortcuts (empty state when no data)

5. **Apply Microcopy Guide** (from Design System):
   - Button labels: "Create site", "Authorize GitHub", "Deploy", "Redeploy", not "Submit"
   - Empty state: "Create your first site" (invitation) + body explaining benefit + CTA
   - Errors: actionable, no stack traces ("Build failed: exit code 1. Check logs.")
   - Status badges: Ready / Building / Failed / Pending (single word, Title Case)

---

### Part D: Secondary Screens (new or expanded)

**Functions page:**
- List of functions with cards (name, runtime, trigger type, status, created date)
- "Create function" CTA (links to function editor, not in-scope for prototype but show navigation)
- Detail view: function code editor (dark code box), trigger config, env vars, logs

**Data Studio:**
- Sidebar: table browser (collections list)
- Main panel: table view with columns, row count, sortable headers
- Actions: Add column (modal), Edit column, Delete (confirm)
- Schema explorer (expandable tree, show dependencies)

**Login (refined from Design System):**
- Centered card (max-width 400px)
- Email input + Password input
- Inline validation (email format, password length)
- "Continue with Google" button (secondary style)
- Forgot password link
- No-account path (sign up) to login link

**Messaging:**
- List of message templates (name, type: Email/SMS, created date, status)
- "Create template" CTA
- Detail view: template editor (subject, body, variables/placeholders, preview)

**Settings:**
- Tabs: Account | Workspace | Billing (placeholder)
- Account: email, password change, 2FA (stubs)
- Workspace: name, description, members (link to Users page), plan
- Dark mode toggle + Theme selector

---

### Part E: System Design Document Updates

1. **Extend Design Tokens table** to include month theme colors + WCAG compliance notes
2. **Add Theme Switching section** explaining how CSS variable cascade updates colors
3. **Document all 24+ components** with explicit states: default, hover, focus, active, disabled, loading, error, empty
4. **Responsive grid**: show exact breakpoints + layout changes (sidebar collapse, table scroll, modal vs. full-page)
5. **Accessibility checklist**: focus rings, contrast, keyboard nav, skip links, aria labels

---

### Part F: Hand-off to Engineering

1. **Export a specs document** with:
   - Exact spacing for each component (padding, margin, gap)
   - Color var names matching codebase (`--brand-500`, `--bg-accent`, etc.)
   - Font families, weights, sizes (Inter for UI, Fira Code for code)
   - Shadow and radius values
   - Animation durations and easing functions

2. **Ensure React component stubs** (in Design System's source/) are complete:
   - TypeScript props interfaces
   - State management (e.g., theme context)
   - Export list matching Console screen structure

3. **Document theme switching** in code:
   - How to read/write localStorage for theme
   - CSS class/attribute approach (e.g., apply `[data-theme="December"]` to root)
   - How tokens cascade to update colors globally

---

**Success Criteria:**

✅ Console + Design System are **unified** (same IA, components, tokens, copy)  
✅ **12 month themes** added, all WCAG 2.1 AA compliant  
✅ **All major journeys** present (Login, Sites create/list/detail, Dashboard, at least 2 secondary screens)  
✅ **Secondary screens** sketched (Functions, Data Studio, Messaging, Settings) — don't need to be fully detailed, but should follow the same design system  
✅ **Microcopy** polished throughout (verb-first buttons, inviting empty states, actionable errors)  
✅ **Responsive** layout tested at 1440px, 1024px, 768px, 375px  
✅ **Accessibility**: focus rings visible, contrast ≥4.5:1, keyboard navigation works in wizard  
✅ **Component specs** clear and exportable for engineering  

---

**Notes for Claude Design:**

- You have real codebase context now (16 pages, 12 components built, 32KB tokens already in place). Don't reinvent; refine and unify.
- The Design System's Appwrite-inspired IA is solid. Bring that philosophy into the Console's visual presentation.
- The month themes are a delightful addition (birthday month = brand color). Make sure they feel intentional, not gimmicky.
- The engineering team is ready for handoff after EPIC 1 (8-week timeline). Keep specs clean and exportable.
- Both prototypes should feel like **one product**, not two separate designs. Same voice, same components, same hierarchy.

---

**When done:**
- Provide updated Console.html with all changes
- Provide updated Design System with new screens + theme specs
- Export both as unified handoff bundles for engineering

---

---

## Summary for You

**In Clara Design, say the above prompt.** The design team will:

1. Unify both prototypes' information architecture
2. Add the 12 month-based themes (WCAG-compliant)
3. Fill in missing secondary screens (Functions, Data Studio, Messaging, Settings)
4. Polish all copy and microcopy
5. Create a handoff-ready specs doc for engineering

**Expected output:** 
- One unified, production-ready Console.html
- One unified, production-ready Design System
- Ready for EPIC 1 implementation (8 weeks)

This is the gap analysis and prompt. Use it to guide the design refinement! 🎯
