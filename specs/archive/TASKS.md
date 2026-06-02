# Task Breakdown

Derived from `specs/RELEASE-PLAN.md`. Each task is a vertical slice:
independently implementable, testable, and deployable.

## Phase 1: UI-First Foundation

### Epic 017 — Enhanced Admin UI ✅ COMPLETE

| Task ID | Slice | Description | Depends On | Verify | Status |
|---------|-------|-------------|------------|--------|--------|
| `017-A` | 017-A | Realtime inspector page at `#/realtime` | None | `cd ui && npm run build` | ✅ |
| `017-B` | 017-B | Function logs viewer + `GET /api/functions/:id/logs` | None | `go test ./components/functions/ -run TestLogs -v` | ✅ |
| `017-C` | 017-C | Storage browser + thumbnail endpoint | None | `go test ./components/storage/ -run TestThumbnail -v` | ✅ |
| `017-D` | 017-D | Deploy detail + CICI pipeline viewer + log stream | None | `go test ./components/deploy/ -run TestLogStream -v` | ✅ |
| `017-E` | 017-E | Dashboard overhaul (dark mode, charts, sidebar, toasts) | 017-A | `cd ui && npm run build` | ✅ |

### Epic 021 — Testing & Quality Hardening

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `021-A` | 021-A | E2E test suite (Playwright) | None | `npx playwright test` |
| `021-B` | 021-B | API contract tests | 018-D | `go test ./tests/contract/...` |
| `021-C` | 021-C | Performance benchmarks | None | `go test -bench=. ./...` |
| `021-D` | 021-D | Coverage gates (80% threshold) | 021-C | CI coverage check passes |
| `021-E` | 021-E | Race detection + fuzz targets | 021-D | `go test -race ./...` |

## Phase 2: Infrastructure

### Epic 018 — Multi-DB Support

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `018-A` | 018-A | Extract `kernel/dber.go` shared interface | None | `go build ./...` |
| `018-B` | 018-B | PostgreSQL driver in `components/db/postgres.go` | 018-A | `go test ./components/db/ -run TestPostgres` |
| `018-C` | 018-C | Config-based driver selection (`--db-driver`, `--db-dsn`) | 018-B | `go run . serve --db-driver postgres --db-dsn "..."` |
| `018-D` | 018-D | Dual-driver CI test matrix (GitHub Actions) | 018-C | CI passes both SQLite and PG |
| `018-E` | 018-E | Versioned migration system | 018-B | `go test ./components/db/ -run TestMigration` |

## Phase 3: Security

### Epic 019 — Security Hardening

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `019-A` | 019-A | Rate limiting middleware | None | `go test ./components/auth/ -run TestRateLimit` |
| `019-B` | 019-B | Email verification flow | 018-E | `go test ./components/auth/ -run TestEmailVerify` |
| `019-C` | 019-C | Password reset flow | 019-B | `go test ./components/auth/ -run TestPasswordReset` |
| `019-D` | 019-D | Refresh token rotation | 018-E | `go test ./components/auth/ -run TestRefreshToken` |
| `019-E` | 019-E | Security headers middleware | None | `curl -I localhost:9999 \| grep -E "Content-Security|HSTS"` |

## Phase 4: Platform Operations

### Epic 020 — Platform Operations

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `020-A` | 020-A | Backup/restore CLI + API | None | `go run . backup && go run . restore` |
| `020-B` | 020-B | DB migration tooling CLI | 018-E | `go test ./components/db/ -run TestMigrationTool` |
| `020-C` | 020-C | Environment variable management | None | `go test ./components/api/ -run TestEnvVars` |
| `020-D` | 020-D | Custom domains for Sites | 018-C | `go test ./components/sites/ -run TestCustomDomain` |
| `020-E` | 020-E | Outbound webhook system | 019-E | `go test ./components/webhooks/ -run TestWebhookDelivery` |

## Phase 5: Developer Experience

### Epic 022 — Developer Experience

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `022-A` | 022-A | Onboarding checklist UI | 017-E | `cd ui && npm run build` |
| `022-B` | 022-B | 1-click scaffolding API | 018-E | `go test ./components/api/ -run TestScaffold` |
| `022-C` | 022-C | Event Bus visualizer | 017-A | `curl -N /api/monitoring/events` |
| `022-D` | 022-D | Sample apps with deploy buttons | 022-B | `go test ./components/deploy/ -run TestSampleDeploy` |
| `022-E` | 022-E | Interactive tutorial overlay | 022-D | Visual verification |

## Phase 6: Multi-Tenancy

### Epic 023 — Multi-Tenancy & Organizations

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `023-A` | 023-A | Organization CRUD + settings | 018-E | `go test ./components/auth/ -run TestOrganization` |
| `023-B` | 023-B | Team membership + invitations | 023-A | `go test ./components/auth/ -run TestMembership` |
| `023-C` | 023-C | Resource isolation (org-scoped) | 023-B | `go test ./components/api/ -run TestOrgIsolation` |
| `023-D` | 023-D | API key management | 023-C | `go test ./components/auth/ -run TestAPIKeys` |
| `023-E` | 023-E | Usage tracking per org | 023-D | `go test ./components/monitoring/ -run TestOrgUsage` |
