# e86s01 — Paginated log search API + UI (keyset cursor)

**type:** feat
**risk:** P1
**context:** domain
**BCPs:** 3

## Summary

Replace the hard `LIMIT 100` in `handleLogSearch` with keyset (cursor-based) pagination and add a "Load more" affordance in the Logs tab. Independent of e86s03 (builds on the same query layer, no schema change).

## Context

Monitoring component — logs API is HTTP-only (no cross-component callers; verified). `id` is `time.Now().UnixNano()` (monotonic) → valid keyset cursor. Response contract changes from `{"data": [...]}` to include `next_cursor`/`has_more`; the UI Logs tab (MonitoringPage.tsx ~L170-204) is the only consumer.

## Requirements

#### MODIFIED: Log queries support cursor pagination
**Before:** `GET /api/monitoring/logs` returns the latest 100 rows (`LIMIT 100`), no paging, response `{"data": [...]}`.
**After:** `GET /api/monitoring/logs?cursor=<id>&limit=<n>` (limit default 100, max 500, min 1) returns `{"data": [...], "next_cursor": "<id>"|null, "has_more": bool}`. Keyset: `WHERE id < ?` (id is monotonic UnixNano), `has_more` via limit+1 probe. `?q=` filter composes with cursor. UI renders "Load more" when `has_more`.

## Implementation Steps

1. `handleLogSearch` (monitoring.go:564) — parse `limit` (default 100, clamp 1-500) and `cursor`; when cursor set, add `AND id < ?`; fetch `limit+1` rows and drop the extra to derive `has_more`. → verify: `go test ./components/monitoring/... -run TestLogPagination -v`
2. Response shape — build `next_cursor` from the last returned row's id (null when `!has_more`); keep `?q=` LIKE filter composed with cursor. → verify: `go test ./components/monitoring/... -run 'TestLogPagination|TestLogSearch' -v`
3. Pagination tests (extend `monitoring_test.go`): insert 250 rows → page 1 = 100 + `has_more=true` + cursor; page 2 via cursor = 100; page 3 = 50 + `next_cursor=null`; cursor+`?q=` combined filter. → verify: `go test ./components/monitoring/... -run TestLogPagination -v`
4. UI (MonitoringPage.tsx Logs tab) — track `nextCursor`/`hasMore` state; `fetchLogs(q, cursor?)` replaces on load/search, appends on cursor; reset cursor on new search. → verify: `cd ui && npx vitest run src/pages/MonitoringPage.test.tsx`
5. UI — "Load more" `<Button>` below the table when `hasMore` (disabled while fetching; hidden when `!hasMore`). → verify: `cd ui && npx vitest run src/pages/MonitoringPage.test.tsx && cd ui && npm run build`

## Verification Script (Step-by-Step)

1. Insert >100 logs (POST loop).
2. Open Monitoring → Logs tab → first 100 rows render, "Load more" visible.
3. Click "Load more" → next page appends; button persists while more pages exist.
4. Search a term → results filtered; "Load more" pages stay filtered.
5. Last page → button disappears.

## Out of scope

- Org scoping (e86s03) — pagination works on the un-scoped query until s03 lands, then composes.
- UI virtualization / infinite scroll — button-based paging only.

## Risks

- `id` collision (two POSTs same nanosecond) could skip a row at a page boundary — mitigate by `ORDER BY created_at DESC, id DESC` tiebreak in the cursor comparison.
- `has_more` probe cost — one extra row fetch per page, negligible at LIMIT ≤500.

## Acceptance Criteria

- [ ] 250-row dataset pages 100/100/50 with correct `has_more`/`next_cursor`
- [ ] `?q=` composes with cursor
- [ ] Logs tab appends pages via "Load more"
