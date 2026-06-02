# UPDATE DESIGN SYSTEM

You are redesigning the BigBase Design System. The BigBase Console prototype (separate file) will reference this updated system.

## Current Context
- React + TypeScript admin UI, Appwrite-inspired, indigo brand (#4F46E5)
- Codebase: 16 pages, 12 built components, 32KB token system
- Release plan: EPIC 1 (Console UI) ready to start immediately (8-week timeline)

## What to Do

### 1. Expand Secondary Screens
Add complete screen designs for:
- **Functions list**: cards with name, runtime, trigger, status, created date, "Create function" CTA
- **Functions detail**: code editor, trigger config, env vars, logs/execution history
- **Data Studio expanded**: table browser with schema explorer, column operations (Add/Edit/Delete)
- **Login refined**: centered card, email/password, "Continue with Google", password reset, field validation
- **Messaging list**: templates with type/date/status, "Create template" CTA
- **Messaging detail**: template editor (subject, body, variables, preview)
- **Settings**: Account (email, password, 2FA stubs) | Workspace (name, members, plan) | Billing (stub)

All should follow existing design system patterns (same components, tokens, spacing, copy tone).

### 2. Add 12 Month-Based Themes (WCAG 2.1 AA Required)

Create theme selector with dropdown in sidebar footer. Each theme replaces accent color (`--brand-500`, `--brand-600`, `--brand-700`, `--border-accent`, `--bg-accent`).

**Month themes** (must pass ≥4.5:1 contrast on white and light backgrounds):

- January: Teal - rgb(13, 148, 136)
- February: Orange - rgb(234, 88, 12)
- March: Purple - rgb(124, 58, 237)
- April: Green - rgb(22, 163, 74)
- May: Lavender - rgb(167, 139, 250)
- June: Rainbow - linear-gradient(to right, rgb(239, 68, 68), rgb(245, 158, 11), rgb(16, 185, 129), rgb(59, 130, 246), rgb(139, 92, 246))
- July: Peach - rgb(253, 186, 116)
- August: Silver/Grey - rgb(156, 163, 175)
- September: Yellow - rgb(234, 179, 8)
- October: Pink - rgb(236, 72, 153)
- November: Blue - rgb(37, 99, 235)
- December: Red - rgb(220, 38, 38)

**Implementation**:
- Add theme selector dropdown (sidebar footer, next to dark mode toggle)
- Persist to localStorage (`bigbase-theme`)
- Apply via CSS custom property: `--brand-500: [rgb value]`
- Derive `--brand-600` and `--brand-700` shades from base color
- Rainbow theme: use gradient on button fills with text-shadow for readability
- Test all colors: ≥4.5:1 contrast with both light and dark surfaces

### 3. Update Design Tokens Documentation

- Extend color tokens table with month themes
- Add WCAG 2.1 AA compliance notes and contrast ratios
- Document theme switching mechanism (CSS variable cascade)
- Show before/after for theme selector UI

### 4. Complete Component States

Ensure all 30+ components have explicit documentation for:
- Default state
- Hover
- Focus (with focus ring)
- Active
- Disabled
- Loading
- Error
- Empty state (where applicable)

Include exact spacing, color tokens, and transitions.

### 5. Update Information Architecture

Sidebar structure (already proposed, confirm in updated design):
```
OVERVIEW → Dashboard
BUILD → Sites, Functions
DATA → Data Studio, SQL Editor, Storage
AUTH → Users
DEVOPS → Git Repos, CI/CD, Monitoring
```

Document cross-links (e.g., Sites → Git Repos for source selection, Functions → Monitoring for logs).

### 6. Responsive Design Specs

Document exact breakpoints and layout changes:
- Desktop (1440px): full sidebar
- Tablet (1024px): adjusted padding
- Mobile (768px): sidebar collapse to icon rail, single-column layouts
- Tiny (375px): mobile-optimized spacing

Include grid collapse rules, table horizontal scroll, modal vs. full-page decisions.

### 7. Copy & Tone Refinement

Apply consistent microcopy across all screens:
- Buttons: verb-first, specific ("Create site", "Authorize GitHub", not "Submit", "OK")
- Empty states: inviting title + benefit explanation + CTA
- Errors: actionable, no stack traces ("Build failed: exit code 1. Check logs.")
- Status labels: "Ready", "Building", "Failed", "Pending" (Title Case, no decorations)
- Section labels: UPPERCASE (e.g., "OVERVIEW", "BUILD")

### 8. Accessibility Verification

- Focus rings visible on all interactive elements (indigo, 3px @ 18% opacity)
- Contrast ≥4.5:1 on all text/background pairs (all 12 month themes included)
- Keyboard navigation in wizards (Tab to fields, Enter to submit)
- Status never by color alone (always include icon + text label)
- Touch targets ≥44px (mobile)
- Aria labels on icon-only buttons

### 9. Export Specs for Engineering

Create a "Specs" section documenting:
- Exact spacing for each component (padding, margin, gap in `--space-*` tokens)
- CSS variable names (match codebase naming)
- Font families, weights, sizes
- Shadow system (`--shadow-xs` through `--shadow-xl`)
- Border radius tokens
- Animation durations and easing functions
- Dark mode token remapping

### 10. React Component Source Completeness

Ensure all components in Design System's source/ folder:
- Have TypeScript props interfaces
- Support theme context (accept color/brand overrides)
- Export from index.ts
- Include stories/examples for each state

---

## Success Criteria

✅ All secondary screens designed (Functions, Data Studio, Messaging, Settings, Login refined)  
✅ 12 month themes implemented with WCAG 2.1 AA compliance verified  
✅ Design System feels unified (same components, spacing, tone across all screens)  
✅ Responsive behavior documented for 3+ breakpoints  
✅ Component states complete (12+ states per component)  
✅ Copy polished throughout (verb-first buttons, inviting empty states, actionable errors)  
✅ Accessibility specs explicit (focus rings, contrast, keyboard nav)  
✅ Engineering handoff specs clear and exportable  

---

## Deliverables

- Updated Design System project with all screens, components, tokens
- Comprehensive specs document (spacing, colors, fonts, animations)
- React component stubs with TypeScript interfaces
- Theme switching documentation with localStorage examples
- Accessibility verification report (all 12 themes pass WCAG AA)

---

**Target**: Production-ready for engineering handoff. The Console prototype will reference this system and inherit all refinements.
