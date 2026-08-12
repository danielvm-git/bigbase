# ctxo MCP serves stale index; search_symbols param mismatch

**Source:** Operational diagnosis + validate-fix (2026-08-11)
**Severity:** MEDIUM
**Scope:** infra (ctxo MCP)

## Description

Three ctxo MCP defects:
1. **Stale index** — `.ctxo/index/` was built 12+ hours behind the code (no refresh automation; servers don't hot-reload). MCP queries returned pre-fix symbol data.
2. **Tool contract mismatch** — `search_symbols` requires `pattern` (the harness-visible contract suggested `query`), so calls with `query` failed `-32602`.
3. **verify-index contradiction** — the gate demands a committed index, but `.gitignore` excluded `.ctxo/*`; and the gate's temp comparison build runs with DEFAULT config (indexes only 21 npm-workspace TS files), so a correctly-configured index (Go root, per BUG-2026-07-10T160209) can never match.

## Root cause

- No automation kept the derived index fresh; servers load once at spawn.
- The `pattern`/`query` mismatch is a docs-vs-schema drift in how the tool was invoked.
- verify-index is upstream-broken: (a) it does not carry the project `config.yaml` into its temp comparison build, and (b) every index JSON embeds a per-file `lastModified` timestamp, so two clean rebuilds differ across all 554 files — a committed baseline is impossible.

## Fix applied

1. Rebuilt the index (546 files / 1878 symbols) + synced the SQLite cache; killed 12h-stale server instances.
2. Audited all 14 ctxo tool schemas via `tools/list`; confirmed `search_symbols` → `pattern`; verified the full tool surface responds with 0 errors.
3. `.githooks/pre-commit` now runs `ctxo index` when source files change (freshness automation; the real fix). Committed-index approach was tried and reverted (timestamp churn — see 846ecfb0a). verify-index documented as upstream-broken in `.ctxo/config.yaml`.

## Status
fixed

## Source
validate-fix-2026-08-11

## Discovered
2026-08-11
