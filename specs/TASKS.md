# Task Breakdown

Derived from `specs/RELEASE-PLAN.md`. Each task is a vertical slice:
independently implementable, testable, and deployable.

## Phase 1: Foundation

### Epic 017 — Multi-DB Support

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `017-A` | 017-A | Extract `kernel/dber.go` shared interface | None | `go build ./...` |
| `017-B` | 017-B | PostgreSQL driver in `components/db/postgres.go` | 017-A | `go test ./components/db/ -run TestPostgres` |
| `017-C` | 017-C | Config-based driver selection (`--db-driver`, `--db-dsn`) | 017-B | `go run . serve --db-driver postgres --db-dsn "..."` |
| `017-D` | 017-D | Dual-driver CI test matrix (GitHub Actions) | 017-C | CI passes both SQLite and PG |
| `017-E` | 017-E | Versioned migration system | 017-B | `go test ./components/db/ -run TestMigration` |

### Epic 021 — Testing & Quality Hardening

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `021-A` | 021-A | E2E test suite (Playwright) | None | `npx playwright test` |
| `021-B` | 021-B | API contract tests | 017-D | `go test ./tests/contract/...` |
| `021-C` | 021-C | Performance benchmarks | None | `go test -bench=. ./...` |
| `021-D` | 021-D | Coverage gates (80% threshold) | 021-C | CI coverage check passes |
| `021-E` | 021-E | Race detection + fuzz targets | 021-D | `go test -race ./...` |

## Phase 2: Security

### Epic 018 — Security Hardening

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `018-A` | 018-A | Rate limiting middleware | None | `go test ./components/auth/ -run TestRateLimit` |
| `018-B` | 018-B | Email verification flow | 017-E | `go test ./components/auth/ -run TestEmailVerify` |
| `018-C` | 018-C | Password reset flow | 018-B | `go test ./components/auth/ -run TestPasswordReset` |
| `018-D` | 018-D | Refresh token rotation | 017-E | `go test ./components/auth/ -run TestRefreshToken` |
| `018-E` | 018-E | Security headers middleware | None | `curl -I localhost:9999 \| grep -E "Content-Security|HSTS"` |

## Phase 3: Admin UI

### Epic 019 — Enhanced Admin UI

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `019-A` | 019-A | Realtime inspector page | None | `cd ui && npm run build` |
| `019-B` | 019-B | Function logs viewer | None | `go test ./components/functions/ -run TestLogs` |
| `019-C` | 019-C | Storage browser + preview | None | `go test ./components/storage/ -run TestThumbnail` |
| `019-D` | 019-D | Deploy/CICI detail viewer | None | `go test ./components/deploy/ -run TestLogStream` |
| `019-E` | 019-E | Dashboard overhaul (charts, quick actions) | 019-A | `cd ui && npm run build` |

## Phase 4: Platform Operations

### Epic 020 — Platform Operations

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `020-A` | 020-A | Backup/restore CLI + API | None | `go run . backup && go run . restore` |
| `020-B` | 020-B | DB migration tooling CLI | 017-E | `go test ./components/db/ -run TestMigrationTool` |
| `020-C` | 020-C | Environment variable management | None | `go test ./components/api/ -run TestEnvVars` |
| `020-D` | 020-D | Custom domains for Sites | 017-C | `go test ./components/sites/ -run TestCustomDomain` |
| `020-E` | 020-E | Outbound webhook system | 018-E | `go test ./components/webhooks/ -run TestWebhookDelivery` |

## Phase 5: Developer Experience

### Epic 022 — Developer Experience

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `022-A` | 022-A | Onboarding checklist UI | 019-E | `cd ui && npm run build` |
| `022-B` | 022-B | 1-click scaffolding API | 017-E | `go test ./components/api/ -run TestScaffold` |
| `022-C` | 022-C | Event Bus visualizer | 019-A | `curl -N /api/monitoring/events` |
| `022-D` | 022-D | Sample apps with deploy buttons | 022-B | `go test ./components/deploy/ -run TestSampleDeploy` |
| `022-E` | 022-E | Interactive tutorial overlay | 022-D | Visual verification |

## Phase 6: Multi-Tenancy

### Epic 023 — Multi-Tenancy & Organizations

| Task ID | Slice | Description | Depends On | Verify |
|---------|-------|-------------|------------|--------|
| `023-A` | 023-A | Organization CRUD + settings | 017-E | `go test ./components/auth/ -run TestOrganization` |
| `023-B` | 023-B | Team membership + invitations | 023-A | `go test ./components/auth/ -run TestMembership` |
| `023-C` | 023-C | Resource isolation (org-scoped) | 023-B | `go test ./components/api/ -run TestOrgIsolation` |
| `023-D` | 023-D | API key management | 023-C | `go test ./components/auth/ -run TestAPIKeys` |
| `023-E` | 023-E | Usage tracking per org | 023-D | `go test ./components/monitoring/ -run TestOrgUsage` |

## Parallel Execution Groups

Tasks within the same dependency tier can be parallelized:

- **Tier 0** (no deps): 017-A, 021-A, 021-C, 018-A, 018-E, 019-A, 019-B, 019-C, 019-D, 020-A, 020-C
- **Tier 1** (deps on Tier 0): 017-B, 021-D
- **Tier 2** (deps on Tier 1): 017-C, 021-E, 019-E
- **Tier 3** (deps on Tier 2): 017-D, 021-B, 018-B, 018-C, 018-D, 022-A
- **Tier 4**: 017-E, 020-B, 020-E, 022-B
- **Tier 5**: 020-D, 022-C, 023-A
- **Tier 6**: 022-D, 023-B
- **Tier 7**: 022-E, 023-C
- **Tier 8**: 023-D
- **Tier 9**: 023-E
