---
bug_id: BUG-133
status: fixed
severity: critical
scope: storage
title: "IDOR on File Storage — list/download/delete any org's files"
created: 2026-07-24
github_issue: 133
security_impact: CRITICAL
---

# BUG-133: IDOR on File Storage

## Problem

**Actual:** Authenticated users could list, download, and delete every file in `storage_files` regardless of organization. The table had no `org_id`; handlers only required auth.

**Expected:** Each org only sees and mutates its own files. Cross-tenant list/get/delete must fail closed (404 / empty list).

**Security impact: CRITICAL** — exploit path is any authenticated JWT/API key against storage endpoints.

## Root Cause Analysis

### Reproduce
Upload as org A; as org B call list/download/delete on A's file id — succeeded without tenant filter.

### Isolate
`components/storage` schema + unscoped INSERT/SELECT/DELETE. Auth wraps routes but was unused for scoping.

### Hypothesize / Verify
Missing `org_id` column and unscoped SQL — same class as BUG-140 / BUG-143 / sites IDOR. Confirmed by code inspection and isolation tests.

**Risk level:** High (disclosure + destructive cross-tenant delete).

## Fix Approach

1. Add `org_id` to `storage_files` (CREATE + ALTER).
2. Require org context on upload/list/download/delete/thumbnail via `kernel.OrgIDFromContext`.
3. Auth middleware dual-writes `kernel.WithOrgID` so production JWT/API-key context reaches storage/monitoring.
4. Cross-tenant isolation tests.

## Resolution

**Fixed:** 2026-07-24  
**PR:** #150 (`2736ef581`) — storage org scoping  
**Follow-up:** auth middleware bridges org into `kernel.WithOrgID` so handlers that read kernel context work behind `auth.Middleware` (storage/monitoring).
