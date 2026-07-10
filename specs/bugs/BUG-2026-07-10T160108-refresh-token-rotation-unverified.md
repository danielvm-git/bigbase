---
bug_id: BUG-2026-07-10T160108
status: fixed
severity: medium
scope: auth
title: Refresh token rotation logic not fully verified
---

# BUG-2026-07-10T160108: Refresh token rotation verification

## Problem

**Security impact: MEDIUM** — CWE-613. Rotation + family invalidation existed but the tip-of-family token after replay was not asserted in tests.

## Fix

Extended `family_invalidation_on_replay` to assert the current refresh token (rt2) is also revoked after replaying an older family member. Implementation already correct.

## Verify

→ verify: `go test ./components/auth/ -run TestRefreshToken -count=1`
