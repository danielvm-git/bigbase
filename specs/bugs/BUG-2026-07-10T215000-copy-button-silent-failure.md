---
bug_id: BUG-2026-07-10T215000
status: fixed
severity: medium
scope: ui
title: CopyButton swallows clipboard errors silently, user sees nothing
---

# BUG-2026-07-10T215000: CopyButton fails silently when clipboard API errors

## Problem

**Actual:** User clicks the copy button in the "Generate Deploy Key" modal result. The button does nothing and no feedback is given — user doesn't know if the copy succeeded or failed.

**Expected:** Either the copy succeeds and shows "Copied!" feedback, or fails and shows an error message.

**Root cause:** `CopyButton.tsx` calls `navigator.clipboard.writeText()` without try-catch, so if the clipboard API fails (HTTPS required, user denied permission, browser restriction, etc.), the error is silently swallowed and the UI gives no feedback.

**Risk level:** Medium — users cannot copy generated deploy keys, defeating the feature's UX.

## TDD Fix Plan

### Cycle 1 — Add error handling to CopyButton

**RED:** Click copy button when clipboard API is unavailable/restricted → no feedback (silent failure).

**GREEN:** Wrap `navigator.clipboard.writeText()` in try-catch. On error, log to console and show `alert('Failed to copy to clipboard. Please try again.')`.

**verify:** `cd ui && npm run build`

**REFACTOR:** None — minimal error boundary.

## Resolution

**Fixed:** 2026-07-10
**Root cause confirmed:** No try-catch on clipboard API call.
**Fix applied:** Added try-catch with error logging and user alert in `CopyButton.tsx`.
**Evidence:** `npm run build` clean.
**Next:** Commit, push, deploy.
