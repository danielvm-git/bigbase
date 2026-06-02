# BigBase Specs

All planning and architecture documents for the BigBase project.

## How to Use

| If you want to... | Read this first |
|-------------------|-----------------|
| Understand the full architecture | `CONTEXT.md` |
| Know what's being built now | `RELEASE-PLAN.md` |
| See task-level breakdown | `TASKS.md` |
| Know what's in/out of scope | `SCOPE.md` |
| Understand domain terms | `UBIQUITOUS_LANGUAGE.md` |
| Trace stories to code | `TRACEABILITY.md` |
| Find refactoring opportunities | `REFACTOR.md` |
| Check session state | `STATE.md` |
| See architecture decisions | `adr/` |
| Find bug investigations | `bugs/` |
| See past component specs | `001-cli-bigbase-breathes.md` through `015-deploy-github-journey.md` |

---

## Planning Documents (v2.0)

| Document | Description |
|----------|-------------|
| `CONTEXT.md` | Domain model, architecture, component catalog, event flow, deployment topology |
| `UBIQUITOUS_LANGUAGE.md` | DDD-style glossary of all domain terms |
| `SCOPE.md` | v2.0 scope: what's in, what's out, future candidates |
| `RELEASE-PLAN.md` | 7 epics (017–023) with vertical slices and verify commands — **UI-first order** |
| `TASKS.md` | Independently grabbable tasks derived from RELEASE-PLAN, with dependency tiers |
| `STATE.md` | Session state tracker — updated by build agents during work |
| `TRACEABILITY.md` | Story-to-code mapping for all 14 original slices |
| `REFACTOR.md` | 6 known refactoring opportunities with severity, scope, and proposed fixes |
| `IMPACT.md` | Blast radius analysis for Epic 018 (Multi-DB) |
| `epics/017-enhanced-admin-ui/` | Design system, 8 screen specs, component inventory, and React migration guide for Epic 017 |

---

## Architecture Decisions

| Document | Decision |
|----------|----------|
| `adr/001-sqlite-json-blob-api.md` | SQLite + JSON blob for auto-CRUD API |
| `adr/002-jwt-bcrypt-auth.md` | JWT + bcrypt for auth component |
| `adr/003-github-app-sites.md` | GitHub App for Sites deployment |

---

## Original Slice Specs (v1.0 — All Implemented)

| Document | Slice | Status |
|----------|-------|--------|
| `001-cli-bigbase-breathes.md` | CLI + ECC Kernel | ✅ done |
| `002-proxy-landing.md` | Proxy + Landing Page | ✅ done |
| `003-db-auto-api.md` | DB + Auto REST API | ✅ done |
| `004-auth.md` | Auth (email/password + JWT) | ✅ done |
| `005-admin-ui.md` | Admin UI (React SPA) | ✅ done |
| `006-storage.md` | Storage (file upload/download) | ✅ done |
| `006-appwrite-look-and-feel.md` | Appwrite Design Token Port | ✅ done |
| `007-git.md` | Git Repo Management | ✅ done |
| `007-commercial-landing-page.md` | Commercial Landing Page | ✅ done |
| `008-forge.md` | Forge (Issues, Kanban, Wiki) | ✅ done |
| `009-cici.md` | CI/CD Pipeline Engine | ✅ done |
| `010-functions.md` | Functions (JS Runtime) | ✅ done |
| `011-realtime.md` | Realtime (WebSocket) | ✅ done |
| `012-messaging.md` | Messaging (Email, SMS, Push) | ✅ done |
| `013-deploy.md` | Deploy (App Runner) | ✅ done |
| `014-monitoring.md` | Monitoring (Metrics, Logs, Alerts) | ✅ done |
| `015-deploy-github-journey.md` | Sites: Deploy from GitHub | ✅ done |

---

## Supporting Documents

| Document | Description |
|----------|-------------|
| `GOOGLE-OAUTH.md` | Google OAuth social login plan |
| `PLAN.md` | Slice 11 (Realtime) implementation plan |
| `DIAGNOSIS.md` | SPA redirect diagnosis (resolved) |
| `DEPLOY.md` | Production deployment to Contabo VPS |
| `bugs/BUG-LOG.md` | Bug tracking log |
| `README.md` | This file |
