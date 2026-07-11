---
bug_id: BUG-2026-07-10T220000
status: fixed
severity: high
scope: ui, api
title: Deploy Keys — revoke and copy operations broken
---

# BUG-2026-07-10T220000: Deploy Keys copy & revoke don't work

## Problem

**Actual:** User can see Deploy Keys tab and keys listed in the table, but:
1. Clicking "Revoke" button does nothing or fails silently
2. When generating a new key, no way to copy the key value to clipboard
3. No error messages shown to user when operations fail

**Expected:** 
- Clicking "Revoke" opens a confirmation modal, confirms, then deletes the key
- When generating a key, a "Copy" button appears next to the key value and successfully copies it
- Failures show clear error messages

**Reproduce:**
1. Open a site's Deploy Keys tab
2. Try to revoke an existing key → click "Revoke" button
3. Try to generate a new key → look for copy button in the modal

## Root Cause Analysis

Code review shows:
- DeployKeyRow has Revoke button that calls `onRevoke(k.key_id)`
- This triggers RevokeConfirmModal to open
- Modal's Revoke button calls `handleRevoke()` which POSTs to `/api/sites/{id}/deploy-keys/{keyID}`
- Backend handler `handleRevokeSiteKey` is implemented and registered at DELETE route

Possible issues:
1. **Authorization** — user context not properly passed to the handler
2. **Silent API failure** — the API call fails but no error is shown to user
3. **Network/CORS** — the DELETE request is being blocked
4. **Modal not opening** — clicking Revoke doesn't open the confirmation modal at all

## TDD Fix Plan

(To be filled in after RCA)

## Acceptance Criteria

- [ ] Revoke button opens confirmation modal
- [ ] Revoking a key successfully deletes it from the table
- [ ] Copy button appears in generate key modal and copies the key
- [ ] Clear error messages on API failures
- [ ] Manual verification in browser
