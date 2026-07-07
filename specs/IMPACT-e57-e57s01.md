# Impact Assessment — e57s01: Kernel Interface Hardening

**Generated:** 2026-07-06
**Mode:** lightweight
**Risk Score:** 7/10

## Target

Two changes in the `kernel/` package:

1. **Logger hoisting:** Remove 18 duplicate `type Logger interface` definitions from components. All components use `kernel.Logger` directly.
2. **DBer hoisting:** Convert 3 standalone `type DBer interface` definitions (mcp, webhooks, backup) to `type DBer = kernel.DBer` type aliases (matching the 12 components already using this pattern).
3. **New kernel/scope.go:** Add `WithProjectID` / `ProjectIDFromContext` with typed unexported context key.

## Dependents (21 total)

### Logger duplicates (18 components)
| Component | File | 
|-----------|------|
| api | `components/api/api.go:59` |
| auth | `components/auth/auth.go:56` |
| admin | `components/admin/admin.go:13` |
| deploy | `components/deploy/deploy.go:26` |
| sites | `components/sites/sites.go:25` |
| storage | `components/storage/storage.go:32` |
| functions | `components/functions/functions.go:24` |
| forge | `components/forge/forge.go:19` |
| git | `components/git/git.go:21` |
| cici | `components/cici/cici.go:16` |
| github | `components/github/github.go:25` |
| messaging | `components/messaging/messaging.go:18` |
| monitoring | `components/monitoring/monitoring.go:23` |
| realtime | `components/realtime/realtime.go:29` |
| proxy | `components/proxy/proxy.go:53` |
| db | `components/db/db.go:33` |
| webhooks | `components/webhooks/webhooks.go:30` |
| mcp | `components/mcp/mcp.go:493` |

### DBer standalone definitions (3 components)
| Component | File | Note |
|-----------|------|------|
| mcp | `components/mcp/mcp.go:45` | Full interface, convert to alias |
| webhooks | `components/webhooks/webhooks.go:45` | Full interface, convert to alias |
| backup | `components/backup/backup.go:20` | Full interface, convert to alias |

### New code (0 dependents)
- `kernel/scope.go` — net-new file, no existing callers

## Risk Classification

| Factor | Score | Rationale |
|--------|-------|-----------|
| Fan-in | 4/4 | 21 files across 18 components affected |
| Fan-out | 0/3 | kernel has no external dependencies |
| Churn | 3/3 | 5 recent commits touching kernel/ |

**Total: 7/10** — HIGH risk mechanically, but LOW risk semantically (identical interfaces, compile-time safety net).

## Affected Stories

All changes are mechanical — no story scope changes. The task is purely refactoring:

- **e57s01 (this story):** Contains the hoisting + new scope.go
- **No other stories affected:** Components are not changing behavior, only import paths

## Test Coverage

| Test | Covers |
|------|--------|
| `kernel/kernel_test.go` | Kernel start/stop, component registration |
| `kernel/eventbus_test.go` | Event publishing/handling |
| Component tests (18 suites) | Every component has tests exercising Logger/DBer through their respective test harnesses |

No test gaps — the compiler validates interface compatibility at build time.

## Recommended Action

**Proceed with implementation.** Risk score = 7 (at threshold, not exceeding). The compiler guarantees safety — if any component's Logger/DBer usage diverges from `kernel.Logger`/`kernel.DBer`, it fails at build time, not runtime. No grill-me session required.

**Caution:** The 18-file Logger deletion is mechanical but high-volume. Do one component at a time, verify `go build ./...` after each, to avoid debugging 18 compilation errors simultaneously.
