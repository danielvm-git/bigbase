# BigBase System Design

**Version**: 1.0  
**Status**: Prototype Complete  
**Date**: June 1, 2026  

## Overview

BigBase is an open-source hosting platform and database that provides authentication, storage, and serverless capabilities for modern applications. The system design encompasses a comprehensive console UI with full operational visibility.

## Architecture Components

### Core System (16 Components)

1. **Database (db)** - Primary data store, no dependencies
2. **API** - REST/GraphQL gateway (depends: db)
3. **Storage** - S3-compatible object storage (depends: db)
4. **Git Integration** - GitHub/GitLab connectivity (depends: db)
5. **Messaging** - Email/SMS delivery system (depends: db)
6. **Deploy** - Deployment orchestration (depends: db, git)
7. **Monitoring** - System observability & alerting (depends: db)
8. **Admin** - Internal management tools (no dependencies)
9. **GitHub Sync** - GitHub API bridge (depends: db)
10. **CI/CD** - Pipeline execution engine (depends: db)
11. **Functions** - Serverless function runtime (depends: db)
12. **Auth** - Authentication & authorization (depends: db)
13. **Forge** - Build system & compilation (depends: db)
14. **Realtime** - WebSocket/event streaming (depends: auth, api)
15. **Proxy** - Edge request routing (no dependencies)
16. **Sites** - Web hosting layer (depends: db)

### System Status Metrics

- **Components**: 16 total, all operational
- **CPU Usage**: ~23%
- **Memory**: 512 MB (25% utilization)
- **Requests**: 1.2k/sec
- **Uptime**: 14d 6h+

## Console Features

### 1. Dashboard
- **Health Banner**: All systems operational status
- **System Status Panel**: Component status, resource utilization, activity feed
  - Real-time metrics: CPU, memory, request rate
  - Component health grid (16/16 running)
  - Activity log with deployment, build, migration events
- **Site Cards**: Recent deployments with status badges
- **Quick Actions**: Create site, view deployments, manage settings

### 2. Sites Management
- **List View**: All deployed sites with:
  - Framework (Astro, Next.js, Vite, Static HTML, SvelteKit)
  - Current status (ready, building, failed)
  - Git repo and branch
  - Live URL
  - Last deployment
- **Create Site Wizard**:
  - Step 1: Source (GitHub repo selection with OAuth flow)
  - Step 2: Configure (framework, build settings, environment variables)
  - Step 3: Deploy (with live build log streaming)
- **Site Details**:
  - Overview tab: Deployment history, configuration
  - Deployments tab: Full deploy log with status and duration
  - Domains tab: Custom domain management
  - Logs tab: Build and runtime logs
  - Settings tab: Site configuration and redeploy

### 3. SQL Editor
- Dark-themed query editor (Fira Code monospace)
- Sidebar: Table browser
- Query execution with:
  - Row count and execution time
  - Result table with column headers
  - Query sample: SELECT id, email, role, verified FROM users WHERE verified = true

### 4. Storage (Object Storage)
- **Buckets**: public-assets, avatars, uploads, backups
- **File Browser**:
  - File list with name, size, type, modified date
  - Bucket navigation
  - Upload action button
  - File types: images, CSV exports, PDF documents, SQL dumps

### 5. Users
- **User Management**:
  - List all team members
  - Email, role (owner/admin/member), verification status
  - Creation date
  - Search functionality
  - Invite new users button
- **Avatars**: User initials in colored circles
- **Role Badges**: Verification status indicator

### 6. Git Repos
- **Repository Browser**:
  - Repo name, description, last updated
  - Language badge (Astro, TypeScript, Go, JavaScript, Svelte)
  - Visibility badge (public/private)
  - Link to "Create site" from repo
- **Supported Repos**: Connected via GitHub OAuth

### 7. CI/CD Pipelines
- **Pipeline Dashboard**:
  - Success rate tiles
  - Running/queued/completed pipelines
- **Pipeline Runs**:
  - Repo, branch, commit
  - Status (ready, building, failed)
  - Trigger type (push, manual, schedule)
  - Actor (username or 'system')
  - Duration and timestamp

### 8. Monitoring
- **System Panel** (enhanced):
  - Component health (16/16, version, status, dependencies, hooks)
  - CPU and memory graphs
  - Request rate
  - Uptime counter
- **Activity Feed**:
  - Deploy completed
  - Build running
  - Auth DB migration
  - Push notifications
  - Nightly cron jobs

## Design System

### Color Tokens
- **Brand**: Indigo (#4F46E5)
- **Accent**: Emerald/success (#10B981)
- **Warning**: Amber (#F59E0B)
- **Error**: Red (#EF4444)
- **Neutral Grayscale**: 25 levels (0-900)

### Typography
- **Sans**: Inter (400, 450, 500, 600, 700)
- **Mono**: Fira Code (400, 500, 600)
- **Scales**: xs (12px), s (14px), m (16px), l (20px), xl (24px), 2xl (32px), 3xl (40px)

### Component Library
- Buttons: primary, secondary, danger, ghost, link
- Cards: surface panels with borders and shadows
- Inputs: text fields with labels and focus states
- Badges: status indicators (success, warning, error, neutral)
- Avatars: user initials with backgrounds
- Toasts: notification system (info, success, error)
- Tables: sortable, with pagination support

### Responsive Design
- Desktop: Full sidebar (240px) + content area
- Tablet/Mobile: Icon-only sidebar (64px) + collapsed nav

### Dark Mode
- Full dark theme support with token remapping
- Toggle in sidebar footer
- Automatic theme persistence

## Deployment & Build Process

### Build Pipeline
1. **Detect**: Framework auto-detection (Astro, Next.js, Vite, SvelteKit, Static)
2. **Install**: npm install based on package.json
3. **Build**: Framework-specific build command execution
4. **Upload**: Assets uploaded to edge CDN
5. **Deploy**: Live URL provisioning with DNS routing

### Build Log Streaming
- Real-time console output
- Status indicators (pending, running, success, failed)
- Duration tracking
- Automatic progression notifications

### Environment Variables
- Per-site configuration
- Build-time and runtime variables
- Secure storage and management

## Security & Operations

### Authentication
- Email-based login (magic links)
- OAuth integration (GitHub)
- Session management
- Role-based access control (RBAC)

### Data Protection
- S3-compatible object storage with bucket policies
- Database encryption at rest
- SSL/TLS for all transport
- Audit logging for all operations

### Monitoring & Observability
- Real-time component health tracking
- CPU/memory resource monitoring
- Request rate and latency metrics
- Activity feed for audit trail
- Build and deployment logging

## API Endpoints (Planned)

```
GET  /api/v1/sites                 # List sites
POST /api/v1/sites                 # Create site
GET  /api/v1/sites/:id             # Get site details
POST /api/v1/sites/:id/deploy      # Trigger deployment

GET  /api/v1/deployments           # List deployments
GET  /api/v1/deployments/:id       # Get deployment details
GET  /api/v1/deployments/:id/logs  # Stream build logs

GET  /api/v1/repos                 # List connected repos
GET  /api/v1/users                 # List team members
POST /api/v1/users                 # Invite user

GET  /api/v1/storage/buckets       # List storage buckets
GET  /api/v1/storage/:bucket/files # List files
POST /api/v1/storage/:bucket/upload # Upload file

GET  /api/v1/pipelines             # List CI/CD runs
POST /api/v1/pipelines/trigger     # Manual pipeline run

POST /api/v1/sql/query             # Execute SQL
GET  /api/v1/monitoring/metrics    # System metrics
GET  /api/v1/monitoring/components # Component status
```

## Data Models

### Site
```
{
  id: string
  name: string
  framework: string
  repo: string
  branch: string
  root: string
  status: 'ready' | 'building' | 'failed'
  url: string
  commit: string
  updated: timestamp
  env: 'production' | 'preview'
  deployments: Deployment[]
}
```

### Deployment
```
{
  id: string
  status: 'ready' | 'building' | 'failed'
  commit: string
  commitMsg: string
  when: timestamp
  duration: string
  logs: string[]
}
```

### Function
```
{
  id: string
  name: string
  runtime: string
  trigger: 'HTTP' | 'Schedule'
  timeout: number
  status: 'active' | 'inactive'
  updated: timestamp
}
```

### User
```
{
  id: string
  email: string
  role: 'owner' | 'admin' | 'member'
  verified: boolean
  created: timestamp
}
```

## Version & License

- **Current Version**: v1.1.0
- **License**: MIT
- **Built with**: Go backend, React frontend, ECC architecture
- **Repository**: github.com/bigbase/console

## Next Steps

See RELEASE_PLAN.md for feature prioritization and implementation roadmap.
