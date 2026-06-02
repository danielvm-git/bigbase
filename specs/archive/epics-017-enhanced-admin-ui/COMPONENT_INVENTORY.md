# BigBase Console - Component Inventory

**Auto-generated from Prototype**: June 1, 2026  
**Total Components**: 24 (primitives) + 8 (screens)  

## Primitive Components

### Layout
- **Sidebar** - Left navigation (240px desktop, 64px mobile), logo, nav sections, user menu
- **Header** - Top bar with page title, action buttons, theme toggle
- **Card** - Surface panel with border, shadow, and padding
- **Container** - Content wrapper with max-width constraint
- **Grid** - Responsive multi-column layout for metrics and lists

### Navigation
- **NavLink** - Sidebar navigation item with icon and label
- **TabNav** - Horizontal tab bar for page sections
- **Breadcrumb** - Path navigation (not yet used, planned)

### Forms
- **Input** - Text field with label, placeholder, and validation states
- **Textarea** - Multi-line text input
- **Select** - Dropdown selector (framework, role, etc.)
- **Checkbox** - Boolean input
- **SearchInput** - Input with search icon and auto-clear
- **InputGroup** - Labeled input with helper text

### Display
- **Badge** - Status label (success, warning, error, neutral)
- **StatusBadge** - Animated status with spinner for building state
- **Avatar** - User initial circle
- **Tag** - Small label (language, visibility)
- **Pill** - Rounded label with icon
- **ProgressBar** - Linear progress indicator
- **Spinner** - Loading animation (3 sizes)
- **Skeleton** - Loading placeholder card

### Actions
- **Button** - Interactive button (primary, secondary, danger, ghost, link)
- **IconButton** - Button with only icon
- **DropdownMenu** - Popup menu with actions (not fully implemented)
- **Modal** - Overlay dialog (not yet implemented)

### Notifications
- **Toast** - Auto-dismiss notification (info, success, error)
- **PreviewBanner** - Sticky banner (preview mode warning)
- **Alert** - Inline alert message

### Tables
- **Table** - Data table with sorting/filtering capability
- **TableHead** - Header row
- **TableCell** - Data cell with alignment
- **Row** - Data row with hover state

### Other
- **Icon** - 30+ SVG icons (Lucide-derived)
- **CodeBlock** - Monospace code display with syntax highlighting
- **StreamLog** - Terminal-style build log viewer

---

## Screen Components (Pages)

### 1. **Dashboard**
**Route**: `#/`  
**Props**: None (uses global data)  

```
Layout:
├─ Header (title: "Dashboard")
├─ PreviewBanner
├─ SystemStatusPanel
│  ├─ Status badge ("All systems operational")
│  ├─ MetricCard (CPU: 23%)
│  ├─ MetricCard (Memory: 512 MB)
│  ├─ MetricCard (Components: 16/16)
│  └─ ActivityFeed
│     └─ ActivityRow[] (5 items)
└─ SiteCardsGrid
   └─ SiteCard[] (3 sites)
```

**Features**:
- Real-time health status
- Quick links to all major sections
- Recent deployment history
- System metrics at a glance

---

### 2. **Sites**
**Route**: `#/sites`  
**Props**: None  

```
Layout:
├─ Header (title: "Sites", button: "Create Site")
├─ PreviewBanner
├─ SitesList (or SiteCard Grid)
│  └─ SiteListRow[] (3 sites)
│     ├─ Status badge
│     ├─ Framework tag
│     └─ Actions menu
└─ [When Create clicked]
   └─ CreateSiteWizard
      ├─ Step 1: SourceStep (GitHub repo select)
      ├─ Step 2: ConfigureStep (settings)
      └─ Step 3: DeployStep (build log)
```

**Features**:
- List all deployed sites
- Quick access to deploy/redeploy
- Framework and status badges
- Site detail view with tabs
- Create site wizard (3-step)

---

### 3. **Site Detail**
**Route**: `#/sites/:id`  
**Props**: `{ siteId: string }`  

```
Layout:
├─ Header (title: site.name, button: "Redeploy")
├─ PreviewBanner
├─ TabNav (Overview | Deployments | Domains | Logs | Settings)
└─ TabContent
   ├─ OverviewTab
   │  ├─ Config details
   │  └─ Recent deployment card
   ├─ DeploymentsTab
   │  └─ DeploymentsList
   │     └─ DeploymentRow[] (3+ rows)
   ├─ DomainsTab (empty, placeholder)
   ├─ LogsTab (empty, placeholder)
   └─ SettingsTab (empty, placeholder)
```

**Features**:
- Multi-tab interface
- Deployment history with durations
- Redeploy button with toast on trigger
- Build log access
- Configuration display

---

### 4. **SQL Editor**
**Route**: `#/sql`  
**Props**: None  

```
Layout:
├─ Header (title: "SQL Editor")
├─ PreviewBanner
└─ EditorContainer
   ├─ Sidebar
   │  └─ TableBrowser
   │     └─ Table[] (collections)
   └─ MainPanel
      ├─ CodeEditor (dark theme, Fira Code)
      ├─ Toolbar (Run button)
      └─ ResultsPanel
         ├─ Metadata (rows, time)
         └─ ResultsTable
            └─ TableRow[] (result set)
```

**Features**:
- Dark code editor
- Table browser sidebar
- Query execution with timing
- Result pagination
- Sample query provided
- Column headers from schema

---

### 5. **Storage**
**Route**: `#/storage`  
**Props**: None  

```
Layout:
├─ Header (title: "Storage", button: "Upload")
├─ PreviewBanner
└─ StorageContainer
   ├─ BucketNav
   │  └─ BucketButton[] (4 buckets)
   └─ FilesPanel
      ├─ Toolbar (Upload button)
      └─ FilesList
         └─ FileRow[] (6 files)
            ├─ File icon
            ├─ Name & type
            ├─ Size & modified
            └─ Actions menu
```

**Features**:
- Bucket selector
- File list with metadata
- Upload button (UI only)
- File action menu
- File type icons
- Sort/filter (planned)

---

### 6. **Users**
**Route**: `#/users`  
**Props**: None  

```
Layout:
├─ Header (title: "Users", button: "Invite")
├─ PreviewBanner
└─ UsersContainer
   ├─ SearchBar
   └─ UsersList
      └─ UserRow[] (4 users)
         ├─ Avatar
         ├─ Email
         ├─ Role badge
         ├─ Verified status
         └─ Created date
```

**Features**:
- User list with roles
- Verification badges
- Avatar initials
- Search functionality (UI)
- Invite button (modal planned)
- Sort by role or date

---

### 7. **Git Repos**
**Route**: `#/repos`  
**Props**: None  

```
Layout:
├─ Header (title: "Git Repos")
├─ PreviewBanner
└─ ReposContainer
   └─ ReposList
      └─ RepoRow[] (5 repos)
         ├─ Repo name
         ├─ Description
         ├─ Language tag
         ├─ Visibility badge
         ├─ Updated time
         └─ "Create site" link
```

**Features**:
- Connected repos from GitHub
- Language badges with colors
- Visibility indicators (public/private)
- Cross-link to "Create site"
- Last updated timestamp

---

### 8. **CI/CD Pipelines**
**Route**: `#/pipelines`  
**Props**: None  

```
Layout:
├─ Header (title: "CI/CD")
├─ PreviewBanner
└─ PipelinesContainer
   ├─ MetricCards (success rate)
   └─ PipelineRunsList
      └─ PipelineRow[] (5 runs)
         ├─ Repo + branch
         ├─ Commit hash
         ├─ Status badge
         ├─ Trigger type
         ├─ Actor/user
         └─ Duration
```

**Features**:
- Pipeline run history
- Success rate metrics
- Status tracking (ready/building/failed)
- Trigger type display (push/manual/schedule)
- Duration and actor info

---

### 9. **Monitoring**
**Route**: `#/monitoring`  
**Props**: None  

```
Layout:
├─ Header (title: "Monitoring")
├─ PreviewBanner
├─ SystemStatusPanel (full width)
│  ├─ Metrics display
│  ├─ Component table (16 components)
│  └─ Activity feed (5 items)
└─ (More monitoring tabs planned)
```

**Features**:
- Full system status panel
- Component table with versions and status
- Real-time metrics
- Activity feed
- Expandable details (planned)

---

### 10. **Placeholder** (Temporary)
**Route**: `#/functions`, `#/data-studio`, etc.  

```
Layout:
├─ Header (title: screen name)
├─ PreviewBanner
└─ EmptyState
   ├─ Icon
   ├─ Message
   └─ Actions
```

Screens not yet implemented:
- Functions
- Data Studio
- Analytics
- Settings (full page)

---

## Theme System

### Color Variables
```
Light Mode:
--neutral-0 to --neutral-900      (25 shades)
--brand-50 to --brand-700          (indigo)
--success, --warning, --error, --info

Dark Mode:
All colors remapped with suffix `[data-theme="dark"]`
```

### Responsive Breakpoints
```
Desktop:  >= 1024px (full sidebar)
Tablet:    768px-1023px (full sidebar, adjusted padding)
Mobile:    < 768px (icon-only sidebar, single column)
```

---

## State Management

### Context Providers
- **ToastProvider**: Push notifications globally
- **ThemeProvider**: Dark/light mode toggle (localStorage)
- **DataProvider**: Mock data + future API integration

### Hook Usage
```javascript
const toast = useToast()       // Push toast notifications
const [theme, setTheme] = useTheme()  // Theme toggle
const [screen, setScreen] = useState('dashboard')  // Navigation
```

---

## Icons (Lucide-derived)

**Navigation**: layout-dashboard, database, terminal, folder, users, git-branch, settings, log-out  
**Actions**: plus, send, check, refresh-cw, external-link, copy  
**Status**: check-circle, alert-triangle, info, clock  
**UI**: chevron-right, chevron-left, chevron-down, more-horizontal, search, mail  
**Features**: rocket, code, box, git-pull-request, activity, zap, hard-drive, bell  

---

## Form Fields & Validation

### Signup/Login
- Email input (required, valid email format)

### Create Site Wizard
- Repo selector (required, searchable)
- Site name (required, alphanumeric + hyphens)
- Branch selector (required, dropdown)
- Framework selector (required, 5 options)
- Build command (text field, optional override)
- Environment variables (key-value pairs, optional)

### Storage Upload
- File input (drag-drop support planned)

### User Invite
- Email input (required, valid email)
- Role selector (required, 3 roles)

---

## Performance Targets

- **Initial Load**: < 2s (with CDN)
- **Page Navigation**: < 300ms (hash routing)
- **Form Submission**: < 500ms (mock API)
- **List Rendering**: < 100ms (100 items)
- **Theme Toggle**: < 50ms
- **Search/Filter**: < 200ms

---

## Accessibility (WCAG 2.1 AA)

- [ ] All buttons have accessible labels
- [ ] Form inputs have associated labels
- [ ] Color contrast ratio ≥ 4.5:1
- [ ] Focus ring visible on keyboard navigation
- [ ] Icon buttons have aria-label
- [ ] Modals have focus trap
- [ ] Status messages announced with aria-live
- [ ] Spinners have role="status"

---

## Testing Coverage

### Unit Tests
- Component rendering (React Testing Library)
- State management and hooks
- Utility functions (timeAgo, statusVariant)
- Icon rendering

### Integration Tests
- Page navigation
- Form submission
- Toast notifications
- Theme toggle

### E2E Tests (Planned)
- Full user journeys
- Create site workflow
- Deployment triggering
- Dark mode persistence

---

## Browser Support

- Chrome/Edge: Latest 2 versions
- Firefox: Latest 2 versions
- Safari: Latest 2 versions
- Mobile: iOS Safari 14+, Chrome Android 14+

---

## Dependencies

### Runtime
- React 18.3.1 (CDN)
- React DOM 18.3.1 (CDN)
- Babel 7.29.0 (CDN)

### Assets
- Inter font (Google Fonts)
- Fira Code font (Google Fonts)
- 30+ Lucide icons (inline SVG)
- BigBase logo (inline SVG)

---

## Known Limitations (Prototype)

- No backend integration (mock data only)
- No actual file upload
- No real SQL query execution
- No real Git OAuth flow (UI only)
- No persistence between page reloads
- Limited error scenarios
- No animations (CSS ready, JS not added)

---

## Next Steps for Implementation

1. **Week 1-2**: Set up React/TypeScript project, establish build pipeline
2. **Week 3-4**: Implement core layout, navigation, component library
3. **Week 5-6**: Build all screens with static mock data
4. **Week 7**: API stub implementation and integration
5. **Week 8**: Testing, accessibility audit, performance optimization
6. **Week 9+**: Backend development and live data integration
