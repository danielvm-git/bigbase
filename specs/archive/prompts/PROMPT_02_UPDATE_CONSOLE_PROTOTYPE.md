# UPDATE CONSOLE PROTOTYPE

You are updating the BigBase Console prototype. The Design System (separate file) has been fully updated with new screens, 12 month themes, complete components, and refined copy. Your task: integrate all Design System updates into the Console and unify both designs.

## Current Context
- Console: 8 screens, Sites wizard, dark mode, responsive
- Design System: Now has 10+ screens, 12 month themes, complete component specs
- Codebase: 16 pages ready for implementation
- Timeline: 8-week EPIC 1 implementation starts after handoff

## What to Do

### 1. Adopt Updated Design System's Information Architecture

Replace Console's sidebar with Design System's grouped structure:
```
OVERVIEW
  → Dashboard

BUILD
  → Sites
  → Functions

DATA
  → Data Studio
  → SQL Editor
  → Storage

AUTH
  → Users

DEVOPS
  → Git Repos
  → CI/CD
  → Monitoring
```

Update all navigation links and cross-references. Test wizard flow transitions.

### 2. Add Missing Screens from Updated Design System

Integrate all new screens designed in the updated Design System:
- **Functions list**: same card pattern as Sites (name, runtime, trigger, status, created, "Create" CTA)
- **Functions detail**: code editor, trigger config, env vars, logs
- **Data Studio expanded**: table browser, schema explorer, column operations
- **Login refined**: centered card, email/password, Google button, validation, reset link
- **Messaging list**: templates, type/date/status, "Create" CTA
- **Messaging detail**: template editor
- **Settings**: Account | Workspace | Billing tabs

All screens should use exact styling from Design System (spacing, colors, fonts, states).

### 3. Integrate 12 Month-Based Themes

Add theme selector dropdown to sidebar footer (next to dark mode toggle).

**Themes** (from updated Design System):
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

**Implementation**:
- Dropdown selector with 12 options
- Persist selection to localStorage
- Apply theme by updating CSS custom properties (`--brand-500`, `--brand-600`, `--brand-700`)
- All themes tested for WCAG 2.1 AA compliance (≥4.5:1 contrast)
- Works in both light and dark modes

### 4. Unify Component Styling

Make sure all 24+ components in Console exactly match Design System specs:
- Button: primary/secondary/danger/ghost/link, all sizes, all states
- Input: text/email/password/select/textarea, with validation states
- Card: surface/elevated, with header/title variants
- Badge: all semantic colors (success/warning/error/info)
- Tabs: active/inactive states
- Table: sortable headers, hover rows, actions column
- ChoiceCard: selected state with ring and check
- SiteCard: thumbnail, status badge, meta (URL, branch, time)
- WizardSteps: step rail with done/active/upcoming
- Toast: success/error/info auto-dismiss
- Skeleton: card/text/circle loading states
- EmptyState: icon + title + body + CTA
- PreviewBanner: mock-backend notice
- PageHeader: title + subtitle + actions

Use exact spacing, colors, shadows, and motion from Design System.

### 5. Polish All Journeys

Refine user flows with smooth transitions and state management:

**Journey A - First-Time Sites Deploy** (already exists, enhance):
- Wizard steps animate smoothly between Source → Configure → Deploy
- "Continue" button only enabled when required fields filled
- Deploy progress shows streaming log (building → ready)
- Toast notification on success
- "View site" CTA appears only when ready

**Journey B - Returning User / Sites List**:
- Cards show status at a glance (badge + color wash)
- Filter/search bar above list
- Hover lift on cards, quick-action menu (⋮)
- Click card → Site detail with Redeploy button

**Journey C - Login & Session**:
- Centered card (email + password + "Continue with Google")
- Field validation with inline errors (red border + message)
- Password reset link
- Session timeout handling

**Journey D - Dashboard as Hub**:
- Health banner + stat cards (uptime, deployment count)
- Cross-linked cards to Sites, Functions, Repos, Users
- Recent activity feed
- "Get started" CTAs when no data

**Secondary Journeys**:
- Functions: Create → Editor → Deploy (mirror Sites pattern)
- Data Studio: Table browser → Select table → View/edit
- Messaging: List → Create/edit template → Preview

### 6. Apply Updated Copy & Microcopy

Use Design System's refined copy throughout:
- Buttons: "Create site", "Authorize GitHub", "Deploy", "Redeploy" (verb-first, specific)
- Empty states: "Create your first site" (invitation) + benefit + CTA
- Errors: "Build failed: exit code 1. Check logs." (actionable, no traces)
- Status: "Ready" / "Building" / "Failed" / "Pending" (Title Case)
- Section labels: "OVERVIEW", "BUILD", "DATA" (UPPERCASE)
- Friendly tone throughout (approachable, second person)

### 7. Enhance Responsive Design

Test and refine at all breakpoints (update styling as needed):

**Desktop (1440px)**: Full sidebar (240px), full content

**Tablet (1024px)**: Adjusted padding, sidebar still visible

**Mobile (768px)**: 
- Sidebar collapses to 64px icon rail (labels hidden)
- Single-column layouts
- Tables horizontal scroll
- Modals/forms full-width with safe area padding

**Tiny (375px)**:
- Extra tight spacing
- Touch targets ≥44px
- Form fields full-width
- Font sizes scaled down slightly

### 8. Add Polish Details

- Smooth CSS transitions (motion tokens from Design System: 150ms fast, 200ms medium, 300ms slow)
- Skeleton loaders on list pages while "fetching"
- Breadcrumb navigation on detail pages
- Quick-action context menus (⋮) on cards/rows
- Focus rings visible on all interactive elements
- Hover states on buttons and links (color change or lift)
- Loading spinners on buttons during async operations

### 9. Verify Accessibility

- Focus rings visible (indigo, 3px @ 18%)
- Contrast ≥4.5:1 all text/background (test with all 12 themes)
- Keyboard navigation in wizards (Tab, Enter, Arrow keys)
- Status never by color alone (icon + text label)
- Touch targets ≥44px
- Aria labels on icon-only buttons
- Form labels associated with inputs

### 10. Update System Design Documentation in Console

Embed or link to Design System's specs:
- Color tokens table (with month themes)
- Typography scale
- Spacing/radius/shadow tokens
- Component states matrix
- Motion/easing definitions
- Responsive breakpoint specs
- Accessibility guidelines

---

## Integration with Design System

The Console now references the Design System as its source of truth:
- Components: use exact Design System specs
- Colors: cascade from Design System tokens via CSS variables
- Spacing: follow Design System's 4px base rhythm
- Copy: follow Design System's tone guide
- Accessibility: implement Design System's standards
- Responsive: match Design System's breakpoint behavior

When Design System updates, Console inherits changes automatically (via token cascade).

---

## Success Criteria

✅ Console adopts Design System's information architecture (grouped sidebar)  
✅ All Design System screens integrated into Console (Functions, Data Studio, Messaging, Settings, Login, etc.)  
✅ 12 month themes working, theme selector visible, localStorage persistence  
✅ All components styled to match Design System specs exactly  
✅ All user journeys (A, B, C, D) smooth with proper state transitions  
✅ Copy polished throughout (verb-first buttons, inviting empty states, actionable errors)  
✅ Responsive design working at 1440px, 1024px, 768px, 375px  
✅ Accessibility verified (focus rings, contrast, keyboard nav, touch targets)  
✅ Design System is source of truth; all styling consistent  

---

## Deliverables

- Updated Console.html with all screens, new IA, 12 themes, polished journeys
- All screens styled to Design System specs
- Mock data for all screens (3+ sites in various states, functions, users, etc.)
- Responsive design tested and working
- Accessibility verified (WCAG 2.1 AA)
- Ready for engineering handoff

---

**Target**: Production-ready Console prototype that engineering can implement over 8 weeks (EPIC 1). All screens, journeys, themes, and copy finalized.
