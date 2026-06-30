## BUG-2026-06-30T223500 — EventsPage missing from sidebar navigation

**Severity:** medium  
**Priority:** medium  
**Scope:** ui,navigation  
**Date:** 2026-06-30  
**Status:** fixed

---

## Summary

`EventsPage` was routed at `/events` in `App.tsx` but had no entry in the `devOpsNav` array in `Layout.tsx`. Users could only reach the Events page by manually typing the URL — the sidebar provided no nav item.

---

## Root Cause

When `EventsPage` was originally added, the route was wired in `App.tsx` but the corresponding nav item was not added to `devOpsNav` in `Layout.tsx`. There is no CI check enforcing that every routed page has a sidebar entry, so the omission went undetected.

---

## Files Changed

- `ui/src/Layout.tsx` — added `{ to: '/events', label: 'Events', icon: 'zap' }` to `devOpsNav`
- `ui/src/components/Icon.tsx` — added `'zap'` to `IconName` union type and zap SVG path to `paths` Record

---

## Resolution

**Fixed:** 2026-06-30  
**Root cause confirmed:** `devOpsNav` array not updated when `EventsPage` route was added  
**Fix applied:**
1. Added Events nav item to `devOpsNav` in `Layout.tsx:66`
2. Registered `'zap'` icon in `Icon.tsx` `IconName` union (line 41) and paths Record (line 253)

**Hardening added:** The `IconName` union type in `Icon.tsx` now acts as a compile-time type guard — any nav item referencing an unregistered icon name causes `tsc` to fail immediately at `NavItem.icon: IconName`. This prevents a class of silent nav-item misconfigurations at build time rather than at runtime.

**Evidence:** 78 test files, 538 tests pass; `tsc --noEmit` clean; ESLint clean on modified files.

**Commits:**
- `4fc4c753` — `fix(ui): add Events nav item to devOpsNav sidebar`
- `08f98ecf` — `fix(ui): register zap icon in Icon.tsx for Events nav item`
