# Story-to-Code Traceability

Maps each spec story to its implementing code files and tests.

## Slice 1: CLI — "BigBase Breathes" (`001-cli-bigbase-breathes.md`)

| File | Tests |
|------|-------|
| `main.go` | — |
| `kernel/kernel.go` | `kernel/kernel_test.go` |
| `kernel/component.go` | (tested via kernel test) |
| `kernel/eventbus.go` | (tested via kernel test) |

## Slice 2: Proxy + Landing Page (`002-proxy-landing.md`)

| File | Tests |
|------|-------|
| `components/proxy/proxy.go` | `components/proxy/proxy_test.go` |

## Slice 3: DB + Auto API (`003-db-auto-api.md`)

| File | Tests |
|------|-------|
| `components/db/db.go` | `components/db/db_test.go` |
| `components/api/api.go` | `components/api/api_test.go` |

## Slice 4: Auth (`004-auth.md`)

| File | Tests |
|------|-------|
| `components/auth/auth.go` | `components/auth/auth_test.go` |
| `components/auth/jwt.go` | (tested via auth test) |

## Slice 5: Admin UI (`005-admin-ui.md`)

| File | Tests |
|------|-------|
| `components/admin/admin.go` | `components/admin/admin_test.go` |
| `ui/` (full directory) | — |

## Slice 6: Storage (`006-storage.md`)

| File | Tests |
|------|-------|
| `components/storage/storage.go` | `components/storage/storage_test.go` |

## Slice 7: Git (`007-git.md`)

| File | Tests |
|------|-------|
| `components/git/git.go` | `components/git/git_test.go` |

## Slice 8: Forge (`008-forge.md`)

| File | Tests |
|------|-------|
| `components/forge/forge.go` | `components/forge/forge_test.go` |

## Slice 9: CI/CD (`009-cici.md`)

| File | Tests |
|------|-------|
| `components/cici/cici.go` | `components/cici/cici_test.go` |
| `components/cici/workflows.go` | (tested via cici test) |
| `components/cici/runs.go` | (tested via cici test) |

## Slice 10: Functions (`010-functions.md`)

| File | Tests |
|------|-------|
| `components/functions/functions.go` | `components/functions/functions_test.go` |
| `components/functions/handlers.go` | (tested via functions test) |
| `components/functions/runtime.go` | (tested via functions test) |

## Slice 11: Realtime (`011-realtime.md`)

| File | Tests |
|------|-------|
| `components/realtime/realtime.go` | `components/realtime/realtime_test.go` |
| `components/realtime/hub.go` | (tested via realtime test) |
| `components/realtime/client.go` | (tested via realtime test) |

## Slice 12: Messaging (`012-messaging.md`)

| File | Tests |
|------|-------|
| `components/messaging/messaging.go` | `components/messaging/messaging_test.go` |
| `components/messaging/handlers.go` | (tested via messaging test) |

## Slice 13: Deploy (`013-deploy.md`)

| File | Tests |
|------|-------|
| `components/deploy/deploy.go` | `components/deploy/deploy_test.go` |

## Slice 14: Monitoring (`014-monitoring.md`)

| File | Tests |
|------|-------|
| `components/monitoring/monitoring.go` | `components/monitoring/monitoring_test.go` |

## Additional Components

| Component | Files | Tests |
|-----------|-------|-------|
| GitHub App | `components/github/github.go`, `components/github/app_auth.go` | `components/github/github_test.go` |
| Sites (Deploy from GitHub) | `components/sites/sites.go` | `components/sites/sites_test.go` |

## Test Package Summary

| Package | Test Files | Status |
|---------|-----------|--------|
| `kernel` | 1 | ✅ All passing |
| `components/proxy` | 1 | ✅ |
| `components/db` | 1 | ✅ |
| `components/api` | 1 | ✅ |
| `components/auth` | 1 | ✅ |
| `components/admin` | 1 | ✅ |
| `components/storage` | 1 | ✅ |
| `components/git` | 1 | ✅ |
| `components/forge` | 1 | ✅ |
| `components/cici` | 1 | ✅ |
| `components/functions` | 1 | ✅ |
| `components/realtime` | 1 | ✅ |
| `components/messaging` | 1 | ✅ |
| `components/deploy` | 1 | ✅ |
| `components/monitoring` | 1 | ✅ |
| `components/sites` | 1 | ✅ |
| `components/github` | 1 | ✅ |

**Total: 17 packages, 17 test files, 0 failures.**
