# Story e39s03: Terminal-style live log viewer

**type:** feat
**context:** infra
**BCPS:** 2

## Context

Replace the static `<pre>`-based `BuildLogs` component in `SiteDetailPage` with a terminal-style live log viewer that connects to the WebSocket streaming endpoint built in e39s01. The viewer reuses the existing `StreamLog` component (already styled dark with line numbers, auto-scroll, cursor animation) and enhances it with a toolbar (search/filter, copy, timestamp toggle) and ANSI color support.

## Module Zoom-Out (BuildLogs)

- **Purpose:** Display build log output for a deployment
- **Callers:** `SiteDetailPage.tsx` (via `useBuildLogs` hook and `<BuildLogs>` component)
- **Contracts:** Props `{ lines: string[], loading?: boolean, error?: string | null }` — enhanced with `isStreaming`

## Steps

### 1. Add WebSocket streaming support to `useBuildLogs` hook
**→ verify:** `cd ui && npx tsc --noEmit`

Connect to the existing `/api/deploy/:id/logs/stream` WebSocket endpoint (built in e39s01, `components/deploy/log_stream.go`). The `logHub` sends each line as a WebSocket text frame. On connect, it replays historical lines first, then sends live lines.

Implementation:
- When `deploymentId` is provided, open a WebSocket to `/api/deploy/:id/logs/stream`
- On message, append the line to `lines` state (accumulating, not replacing)
- Expose `isStreaming: boolean` — true while WebSocket is connected and deployment is in-progress
- On WebSocket close (normal closure = build finished), set `isStreaming: false`
- On WebSocket error, fall back to existing polling behavior
- Keep existing `lines`, `loading`, `error`, `status` return shape unchanged
- Auto-detect WebSocket availability (if `WebSocket` constructor exists)

### 2. Enhance `StreamLog` component
**→ verify:** `cd ui && npm run build`

Add features to the existing `StreamLog` component (`ui/src/components/StreamLog.tsx`):

**Toolbar** (rendered above the log area):
- Search/filter `<input>` — filters displayed lines client-side, case-insensitive
- "Copy all" `<Button>` — copies all log lines to clipboard via `navigator.clipboard.writeText()`
- Timestamp toggle — prepends `[HH:MM:SS]` to each line (client-side generated)

**ANSI color support:**
- Parse ANSI escape sequences (colors, bold, dim) used by npm/yarn/build tools
- Implement a minimal regex-based ANSI-to-HTML converter (no external deps):
  - `\x1b[31m` → `<span style="color: #ff6b6b">`, `\x1b[0m` → `</span>`
  - Handle 16 basic foreground colors + bold/dim
  - Unrecognized sequences: strip silently
- Apply via `dangerouslySetInnerHTML` on each line's `<code>` element

**New props** (additive, backward-compatible):
```typescript
interface StreamLogProps {
  logs: string[]
  isStreaming?: boolean
  className?: string
  autoScroll?: boolean
  // NEW
  showToolbar?: boolean
  showTimestamps?: boolean
  searchQuery?: string
  onSearchChange?: (q: string) => void
}
```

**Loading state** while streaming:
- Show animated cursor `▊` (already exists in StreamLog as `.stream-log-cursor`)
- Keep auto-scroll behavior (already exists)

### 3. Create `TerminalLogViewer` and replace in `SiteDetailPage`
**→ verify:** `cd ui && npm run build && go test ./components/deploy/ -run TestLogsAPI -v`

New component `TerminalLogViewer` (`ui/src/components/TerminalLogViewer.tsx`):
```typescript
interface TerminalLogViewerProps {
  deploymentId: string
}
```
- Internally uses `useBuildLogs(deploymentId)` for data fetching
- Wraps `<StreamLog>` with toolbar state (search, timestamps)
- Handles all states: loading (spinner/skeleton), empty (no logs message), error (error card), streaming (LIVE badge + cursor), complete (no badge)
- Exported from `ui/src/components/index.ts`

Replace in `SiteDetailPage.tsx`:
- Remove `BuildLogs` import and usage in the `logs` tab
- Replace with `<TerminalLogViewer deploymentId={latestFromDeployments?.id || site?.latest_deployment?.id || ''} />`
- Remove `useBuildLogs` direct call from SiteDetailPage (TerminalLogViewer handles it internally)
- Remove unused variables: `lines`, `logsLoading`, `logsError`

### 4. Deprecate old `BuildLogs` component
**→ verify:** `cd ui && npm run build`

- Remove `BuildLogs` from `ui/src/components/index.ts` export barrel
- Keep `BuildLogs.tsx` file on disk (git history preservation, no broken imports from other branches)
- Run `rg "BuildLogs" ui/src/` to verify no remaining imports
- Full `npm run build` to confirm zero broken references

## Verification Script (Manual)

1. Start the server: `go run . serve --port 9999`
2. Navigate to Admin UI > Deploy > select a site
3. Click "Build Logs" tab
4. Verify: dark terminal-style log viewer with line numbers
5. Trigger a new deployment
6. Verify: logs stream in real-time with cursor animation and LIVE badge
7. Verify: search/filter works (type in search box, only matching lines shown)
8. Verify: "Copy all" copies full log to clipboard
9. Verify: when deployment completes, LIVE badge disappears
10. Verify: with no WebSocket (e.g., older browser), logs still load via polling

## Out of scope

- Line-wrapping toggle (always pre-wrap)
- Download logs as file
- Log level filtering (info/warn/error)
- ANSI background colors (only foreground + bold/dim)
- Scroll-to-bottom button (auto-scroll is sufficient)

## Risks

- **ANSI parsing edge cases**: Some build tools use complex escape sequences. Mitigation: strip gracefully for unknown sequences, add a `showRaw` toggle if needed in future.
- **WebSocket reconnect**: If the connection drops mid-build, lines could be lost. Mitigation: fall back to polling on error, which replays full log.
- **Memory**: Long builds produce thousands of lines. Mitigation: server caps at 500 lines via `maxDeployLogLines`, client naturally bounded by DOM.
