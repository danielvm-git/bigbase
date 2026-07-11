# BigBase Codebase Overview & Architecture
**Repository:** https://github.com/danielvm-git/bigbase  
**Version:** 2.10.0  
**Language:** Go 1.22+ (backend) + React 19 (frontend)  
**Architecture:** Entity-Component-Construct (ECC) pattern  
**Date:** 2026-07-11

---

## Executive Summary

BigBase is a **single-binary, modular BaaS platform** using the ECC (Entity-Component-Construct) architectural pattern. The system consists of:

- **Kernel**: Central lifecycle manager, event bus, database connection pool
- **19 Components**: Pluggable modules (Auth, DB, API, Storage, Git, Deploy, etc.)
- **React Admin Console**: Full-featured dashboard UI at `/admin/`
- **Marketing Site**: Landing page and documentation at `/`

**Key Philosophy:** Components communicate **exclusively via event bus**, never through direct imports. This ensures maximum decoupling and modularity.

---

## 1. Repository Structure

### Root-Level Organization

```
bigbase/
├── components/          # 19 pluggable backend modules
├── kernel/              # Core lifecycle & event bus
├── ui/                  # React admin console (Vite + TS)
├── config/              # Configuration system (profiles, defaults)
├── specs/               # Specifications, plans, design docs
├── tests/               # Integration test suite
├── scripts/             # Build and deployment scripts
├── infra/               # Terraform IaC (AWS)
├── packages/            # npm dependencies (managed by package.json)
├── main.go              # Application entrypoint (23.1 KB)
├── go.mod/go.sum        # Go dependency management
├── package.json         # Frontend dependencies
├── CLAUDE.md            # Claude Code project instructions
├── CONVENTIONS.md       # Coding standards & git workflow
├── README.md            # Public documentation
└── CHANGELOG.md         # Version history (33.1 KB)
```

### Key Configuration Files

| File | Purpose | Size |
|------|---------|------|
| `main.go` | Application entry point, CLI parsing, component initialization | 23.1 KB |
| `go.mod` | Go module dependencies | 2.9 KB |
| `package.json` | Frontend (npm) dependencies | 284 B |
| `.releaserc` | Semantic versioning config | 497 B |
| `CLAUDE.md` | Development instructions for Claude Code | 10.1 KB |
| `CONVENTIONS.md` | Git workflow & coding standards | 3.6 KB |

---

## 2. Backend Architecture: 19 Components

Each component is a self-contained module with:
- **Init(ctx, config)** → Initialize with config
- **Start(ctx)** → Activate after kernel startup
- **Stop(ctx)** → Graceful shutdown
- **Event hooks** → Subscribe/emit via kernel event bus

### Component Breakdown by Domain

#### Core Infrastructure (4 components)
| Component | Purpose | LoC | Role |
|-----------|---------|-----|------|
| **kernel** | Lifecycle manager, event bus, discovery | ~1,500 | Central hub |
| **config** | Configuration merging (Flags > Env > JSON) | ~200 | Config system |
| **proxy** | HTTP routing, header manipulation, CORS | 2,812 | Request routing |
| **admin** | Admin console server, asset embedding | 160 | UI server |

#### User & Access (3 components)
| Component | Purpose | LoC | Role |
|-----------|---------|-----|------|
| **auth** | Email/password, OAuth, JWT sessions | 7,740 | Auth system |
| **github** | GitHub OAuth, repo metadata | 915 | GitHub integration |
| **api** | REST API, request routing, middleware | 2,994 | API gateway |

#### Data & Storage (3 components)
| Component | Purpose | LoC | Role |
|-----------|---------|-----|------|
| **db** | SQLite/PostgreSQL connection pool, queries | 1,092 | Database layer |
| **storage** | File storage, S3 compatibility | 837 | File handling |
| **backup** | Database backup scheduling | 555 | Data protection |

#### Deployment & Git (5 components)
| Component | Purpose | LoC | Role |
|-----------|---------|-----|------|
| **git** | Git repository hosting, ref tracking | 430 | Git server |
| **deploy** | Site deployment, build triggers | 4,047 | Deployment engine |
| **sites** | Deployed site management | 1,406 | Site inventory |
| **cici** | CI/CD workflows, run execution | 763 | Pipeline runner |
| **webhooks** | Webhook delivery & retry logic | 548 | Event delivery |

#### Advanced Features (4 components)
| Component | Purpose | LoC | Role |
|-----------|---------|-----|------|
| **functions** | Serverless JS execution, triggers | 1,948 | Compute layer |
| **messaging** | Email/SMS outbound, history | 984 | Messaging service |
| **realtime** | WebSocket subscriptions, broadcasts | 687 | Live updates |
| **monitoring** | Metrics, health checks, dashboards | 2,614 | Observability |

#### Integrations (1 component)
| Component | Purpose | LoC | Role |
|-----------|---------|-----|------|
| **mcp** | Model Context Protocol (MCP) server | 1,290 | LLM integration |

### Component Communication Pattern

```
Event Bus (Kernel)
    ↑                    ↑
    |                    |
[Component A]        [Component B]
 Subscribe to B       Subscribe to A
 Emit: event.type     Emit: other.type
```

**Zero direct imports** between components. Example:
- ❌ `import "github.com/danielvm/bigbase/components/auth"`
- ✅ `kernel.On("auth.user.created", handler)`

---

## 3. Frontend Architecture: React Admin Console

### Directory Structure

```
ui/
├── src/
│   ├── pages/           # 34 page components (routes)
│   ├── components/      # 114+ reusable UI components
│   ├── context/         # Theme, Auth, Toast state
│   ├── hooks/           # Custom React hooks
│   ├── lib/             # Utilities, API clients, helpers
│   ├── tokens/          # Design tokens (colors, spacing, etc.)
│   ├── styles/          # CSS, animations, global styles
│   ├── types/           # TypeScript type definitions
│   ├── mocks/           # Mock data for dev
│   ├── App.tsx          # Router setup
│   ├── Layout.tsx       # Sidebar shell, navigation
│   └── index.css        # Global stylesheet (46 KB)
├── dist/                # Build output (Vite)
├── package.json         # Dependencies: React 19, Vite, TS
├── vite.config.ts       # Vite build config
├── tsconfig.app.json    # TypeScript config
└── embed.go             # Go file: embeds React build into binary
```

### Page Routes (34 screens)

| Screen | Route | File | Domain |
|--------|-------|------|--------|
| Login | `/login` | LoginPage.tsx | Auth |
| Dashboard | `/` | DashboardPage.tsx | Overview |
| **Sites** | `/deploy` | DeployPage.tsx | Build |
| Site Detail | `/deploy/:siteId` | SiteDetailPage.tsx | Build |
| Create Site | `/deploy/new` | CreateSitePage.tsx | Build |
| **Functions** | `/functions` | FunctionsPage.tsx | Build |
| Function Detail | `/functions/:id` | FunctionDetailPage.tsx | Build |
| **Data Studio** | `/data` | DataStudioPage.tsx | Data |
| **SQL Editor** | `/sql` | SqlEditorPage.tsx | Data |
| **Storage** | `/storage` | StoragePage.tsx | Data |
| **Users** | `/users` | UsersPage.tsx | Auth |
| **Messaging** | `/messaging` | MessagingPage.tsx | Engage |
| Messaging Detail | `/messaging/:id` | MessagingDetailPage.tsx | Engage |
| **Git Repos** | `/repos` | GitReposPage.tsx | DevOps |
| **CI/CD** | `/cici` | CiciPage.tsx | DevOps |
| **Monitoring** | `/monitoring` | MonitoringPage.tsx | DevOps |
| **Forge** | `/forge` | ForgePage.tsx | DevOps |
| **Realtime** | `/realtime` | RealtimePage.tsx | DevOps |
| **Events** | `/events` | EventsPage.tsx | DevOps |
| **Settings** | `/settings` | SettingsPage.tsx | Footer |
| 404 | `*` | NotFoundPage.tsx | Error handling |

### Component Library (114+ components)

**Primitives:**
- Button, Badge, Input, Card, Tabs, Modal
- Spinner, Toast, Tooltip, Icon, Avatar
- Empty State, Page Header, Breadcrumb

**Layout:**
- AppShell, Sidebar, SidebarSection, AppFooter
- Layout (main router wrapper)

**Domain-Specific:**
- DeployKeysList, CopyButton, TutorialOverlay
- DashboardMetrics, ComponentHealthGrid
- EnvVarEditor, FunctionLogsPanel, RequestChart
- SitesSkeleton, StatusBadge, ChoiceCard
- Tabs, WizardSteps, ThemePicker

**Context Providers:**
- ThemeProvider (light/dark + 12 accent colors)
- ToastProvider
- TutorialProvider

### Design System

**Theme System:**
- Light/Dark mode toggle
- 12 Accent colors (default→december)
- Stored in localStorage: `bigbase-theme`, `bigbase-accent`

**Token Types:**
- Colors (neutral scale 0–900, brand, status: success/warning/error)
- Spacing (0, 1, 2, 3, 4, 5, 6, 8, 10, 12, 16, 20, 24, 32)
- Radius (xs, s, m, l, full)
- Shadow (xs, s, m, l, xl)
- Typography (Inter font system)

---

## 4. Key Architectural Patterns

### Pattern 1: ECC (Entity-Component-Construct)

```
Entity = App Instance
Component = Auth | Deploy | Functions | etc.
Construct = Config + Lifecycle (Init → Start → Stop)
```

**Benefit:** Each component is independently testable and replaceable.

### Pattern 2: Event-Driven Communication

```go
// In component A
kernel.On("deploy.site.created", func(ctx, data) {
  // Handle event without importing component B
})

// In component B
kernel.Emit("deploy.site.created", siteData)
```

### Pattern 3: Layered HTTP Routing

```
HTTP Request
    ↓
Proxy Component (CORS, headers, routing)
    ↓
API Component (REST endpoint handler)
    ↓
DB/Auth/Deploy Components (business logic)
    ↓
HTTP Response
```

### Pattern 4: Configuration Merge

```
Precedence: CLI Flags > Environment Variables > Config File

Example:
go run . serve --port 9000              # ← Wins
PORT=8000 go run . serve                # ← Loses
# In defaults.jsonc: { "port": 7000 }   # ← Loses
```

---

## 5. Technology Stack

### Backend
| Layer | Technology | Version | Purpose |
|-------|-----------|---------|---------|
| Language | Go | 1.22+ | Type-safe, fast compilation |
| Database | SQLite (default) / PostgreSQL | Latest | Persistence layer |
| HTTP Server | net/http | stdlib | Request handling |
| Event Bus | Custom (kernel) | in-house | Async communication |
| Git | git CLI | 2.x | Repository hosting |
| Metrics | Prometheus-compatible | custom | Observability |

### Frontend
| Layer | Technology | Version | Purpose |
|-------|-----------|---------|---------|
| Framework | React | 19 | UI components |
| Build Tool | Vite | 5+ | Fast dev server, optimized builds |
| Language | TypeScript | 5+ | Type safety |
| Routing | React Router | 6+ | SPA navigation |
| Styling | CSS (vanilla) | — | No CSS-in-JS overhead |
| Icons | Lucide | 31 names | SVG icon set |
| Fonts | Inter, Fira Code | Google Fonts | Typography |

### DevOps & Deployment
| Tool | Purpose |
|------|---------|
| Terraform | AWS infrastructure (compute.tf, network.tf, main.tf) |
| Docker | Container deployment (implied by cloud-init.yaml) |
| GitHub Actions | CI/CD pipeline (in .github/workflows/) |
| New Relic | Application monitoring & logging |

---

## 6. Development Workflow

### Standard Commands

```bash
# Development
go run .                                  # Start backend + embedded UI
cd ui && npm run dev                     # Start React dev server (hot reload)

# Testing
go test ./...                            # Run all backend tests
npm test                                 # Run frontend tests

# Building
go build -o bigbase .                    # Build single binary (includes embedded UI)
cd ui && npm run build                   # Build React for embedding

# Linting
golangci-lint run ./...                  # Lint Go code
npm run lint                             # Lint React code

# Coverage
go test -coverprofile=coverage.out ./... # Backend coverage
go tool cover -func=coverage.out         # Print summary
```

### Git Workflow (from CONVENTIONS.md)

**Branch naming:**
- `fix/BUG-123` — Bug fix for issue BUG-123
- `feat/e17s01` — Feature for epic e17, story s01
- `chore/cleanup` — Non-feature work

**Commit messages:**
```
feat(auth): add OAuth2 Google provider

Closes BUG-123
```

**PR Process:**
1. Create feature branch
2. Write tests first (TDD)
3. Implement feature
4. Open PR for review
5. Merge after CI passes + approval

---

## 7. Specification System

### Specs Organization

```
specs/
├── epics/                  # Feature epics (e17, e26, e45, etc.)
├── bugs/                   # Bug registry with DAST findings
├── plans/                  # Implementation plans & roadmaps
├── tech-architecture/      # Architecture ADRs & decisions
├── design/                 # Design system and prototypes
├── requirements/           # Feature requirements
├── state.yaml              # Current state snapshot
├── release-plan.yaml       # Version roadmap
└── PROTOTYPE-VS-CODEBASE.md # Latest design parity doc
```

### Epic Naming Scheme

```
e17  = Epic 17 (enhanced admin UI)
e17s01 = Story 1 of e17
e17s01b1 = Batch 1 of story 1
```

---

## 8. Current State (2026-07-11)

### Recent Changes (Last 2 Weeks)

| Date | Type | Summary | Component(s) |
|------|------|---------|-------------|
| 2026-07-11 | docs | Codebase overview document | — |
| 2026-07-09 | feat | Events nav item + EventsPage | UI + admin |
| 2026-07-08 | feat | Deploy Keys management UI | sites component |
| 2026-06-27 | fix | Tooltip component types | UI |
| 2026-06-24 | feat | Design system foundation (22 components) | UI |

### Known Gaps

| Gap | Priority | Effort | Notes |
|-----|----------|--------|-------|
| Prototype missing Events screen | High | 🟡 Medium | Design prototype needs update |
| Forge/Realtime specs incomplete | Medium | 🟡 Medium | Specs exist but limited docs |
| Users CRUD not implemented | Low | 🟢 Low | List-only for now |
| Settings page was stub (now full) | Closed | ✅ | Implemented in e17s17 |

---

## 9. How to Navigate the Codebase

### To understand a feature:

1. **Start with Route**
   - Find page in `ui/src/pages/MyPage.tsx`
   - Note the API endpoint it calls

2. **Find API Endpoint**
   - Search in `components/api/` for the route handler
   - Look for `GET /api/feature/...` pattern

3. **Trace Business Logic**
   - Find the component that owns the feature (e.g., `components/deploy/`)
   - Read the core file: `components/deploy/engine.go`

4. **Check Data Model**
   - Look in `components/db/` for schema or queries
   - Search for table creation SQL

5. **Find Event Hooks**
   - Search for `kernel.On("feature.*")`
   - See what other components listen to

### Example: How Sites Deployment Works

```
User clicks "Create Site" (ui/src/pages/CreateSitePage.tsx)
  ↓
Calls POST /api/sites (ui/src/lib/sites.ts)
  ↓
Handler in components/api/routes_sites.go
  ↓
Creates site via components/sites/engine.go
  ↓
Emits "sites.created" event (kernel event bus)
  ↓
Listened by components/deploy/engine.go
  ↓
Triggers deploy via components/deploy/engine.go
  ↓
Emits "deploy.started" event
  ↓
Listened by components/monitoring/ and components/messaging/
  ↓
Sends notification, logs metrics
```

---

## 10. Key Files to Know

### Backend Entry Points

| File | Purpose | Read When |
|------|---------|-----------|
| `main.go` | CLI parsing, kernel setup, component init | Understanding startup flow |
| `kernel/kernel.go` | Event bus, lifecycle manager | Understanding component communication |
| `components/api/routes.go` | HTTP route handlers | Adding new endpoints |
| `components/proxy/proxy.go` | Request routing, middleware | Understanding request flow |

### Frontend Entry Points

| File | Purpose | Read When |
|------|---------|-----------|
| `ui/src/main.tsx` | React entry point, DOM mount | Understanding React bootstrap |
| `ui/src/App.tsx` | Route definitions | Understanding page routing |
| `ui/src/Layout.tsx` | Sidebar, navigation structure | Understanding shell layout |
| `ui/src/index.css` | Global styles, token CSS variables | Understanding design tokens |

### Configuration & Setup

| File | Purpose | Read When |
|------|---------|-----------|
| `CLAUDE.md` | Development instructions | First time setup |
| `CONVENTIONS.md` | Git & coding standards | Before committing |
| `config/defaults.jsonc` | Default app config | Setting up instances |
| `.env.example` | Environment variable template | Configuring secrets |

---

## 11. Deployment & Production

### Single-Binary Deployment

```bash
# Build
go build -o bigbase .  # Produces 50+ MB binary (includes embedded React)

# Run
./bigbase serve --port 8080 --db /data/bigbase.db

# Deployed site access
# Marketing:  http://localhost:8080
# Admin:      http://localhost:8080/admin/
# API:        http://localhost:8080/api/*
```

### Infrastructure (Terraform)

Configured for **AWS deployment**:
- `compute.tf` — EC2 instance, security groups
- `network.tf` — VPC, subnets, routing
- `main.tf` — Provider config, outputs

---

## 12. Related Documentation

| Document | Location | Purpose |
|----------|----------|---------|
| Tech Stack | specs/plans/TECH_STACK_LATEST.md | Architecture deep-dive |
| Design System | specs/design/ | UI component specs |
| Release Plan | specs/release-plan.yaml | Version roadmap |
| Bug Registry | specs/bugs/registry.yaml | Known issues & security findings |
| Epics | specs/epics/e*/SPEC.md | Feature specifications |

---

## Summary

**BigBase** is a **production-grade BaaS platform** combining:
- 📦 **19 pluggable Go components** (ECC architecture)
- ⚛️ **Full-featured React admin console** (34 screens, 114+ components)
- 🔌 **Event-driven communication** (zero direct component imports)
- 🚀 **Single-binary deployment** (everything in one ~50 MB binary)
- 📐 **Spec-driven development** (all features documented in specs/)

**For design work:** Focus on `ui/src/` (React code) and `specs/design/` (prototype files).

**For backend work:** Start with `main.go`, then drill into `components/<name>/` for your domain.

**For contributions:** Read CLAUDE.md first, then CONVENTIONS.md for standards.
