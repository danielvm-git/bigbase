# Log viewers not accessible (no role=log, not keyboard-scrollable)

**Source:** WCAG 2.2 AA audit (2026-08-11)
**Severity:** MODERATE
**WCAG:** 4.1.3 Status Messages / 2.1.1 Keyboard — Level A/AA

## Description

Streaming/static log regions lack `role="log"` + `aria-live` (newly streamed lines never announced) and are not keyboard-scrollable (scrollable containers without `tabIndex={0}`):

- `BuildLogs.tsx` (line 35) — `<pre>` log region
- `StreamLog.tsx` (line 108) — `.stream-log` container
- `FunctionLogsPanel.tsx` (line 66) — `<pre>` log output

## Recommended Fix
Add `role="log" aria-live="polite" aria-label="<name> log" tabIndex={0}` to each log container so streamed lines are announced and keyboard users can focus+scroll.

## Status
open

## Source
wcag-2.2-audit-2026-08-11

## Discovered
2026-08-11
