# BigBase v2.0 — Scope Definition

## In Scope

The following 7 epics define the v2.0 release. Each is a standalone vertical
slice that can be delivered independently, though some share dependencies.

### Epic 017: Multi-DB Support (PostgreSQL)
- Generalize DBer interface to a shared kernel-level abstraction
- Add PostgreSQL driver via `lib/pq`
- Config-based driver selection (`--db-driver`, `--db-dsn`)
- Dual-driver CI test matrix
- Versioned migration system replacing ad-hoc CREATE TABLE

### Epic 018: Security Hardening
- Rate limiting middleware (token bucket per IP + per user)
- Email verification flow
- Password reset (forgot + reset)
- Refresh token rotation for JWT
- Security headers middleware (CSP, HSTS, X-Frame-Options)

### Epic 019: Enhanced Admin UI
- Realtime inspector page
- Function logs viewer
- Storage browser with preview
- Deploy & CICI pipeline detail viewer
- Dashboard overhaul with live charts and quick actions

### Epic 020: Platform Operations
- Backup/restore CLI and API
- DB migration tooling CLI
- Environment variable management API and UI
- Custom domains for Sites
- Outbound webhook system

### Epic 021: Testing & Quality Hardening
- E2E test suite (Playwright)
- API contract tests
- Performance benchmarks
- Coverage gates (80% minimum)
- Race condition hardening and fuzz testing

### Epic 022: Developer Experience
- Onboarding checklist UI (dashboard home)
- 1-click scaffolding (DB schemas, repos, functions)
- Event Bus visualizer canvas
- Sample applications with deploy buttons
- Interactive tutorial

### Epic 023: Multi-Tenancy & Organizations
- Organization CRUD and settings
- Team membership and invitations
- Resource isolation (org-scoped DB, storage, functions)
- API key management per org

## Out of Scope (v2.0)

- **SDK/client library generation** — REST API docs only
- **Redis integration** — SQLite and PostgreSQL remain the only data stores
- **Container/runtime isolation** — deploys continue as host processes
- **Enterprise SSO** (SAML, LDAP) — Google OAuth remains the only SSO
- **Rate limiting with Redis** — in-memory + SQLite bucket store for v2.0
- **PostgreSQL read replicas** — single PostgreSQL connection for v2.0
- **Billing/invoicing integration** — usage tracking is technical only, no payment gateway
- **CGo dependencies** — pure Go only (`modernc.org/sqlite`, `lib/pq` with CGo disabled)

## Future Candidates (v3.0+)

- Client SDK generation (JS, Go, Python)
- Redis for rate limiting, pub/sub, caching
- Container-based deployment isolation
- Enterprise SSO (SAML, LDAP, OIDC)
- PostgreSQL read replicas and sharding
- Billing and usage-based metering
