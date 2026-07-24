---
bug_id: BUG-133
status: fixed
severity: critical
scope: storage
title: "IDOR on File Storage — list/download/delete any org's files"
github_issue: 133
---

# BUG-133: IDOR on File Storage

## Summary

`storage_files` has no `org_id` column; upload/list/download/delete handlers require only authentication. Any authenticated user can access every org's files.

## Root Cause

Missing tenant column and unscoped SQL in `components/storage/storage.go`:

- `INSERT` omits `org_id`
- `SELECT ... FROM storage_files` has no `WHERE org_id = ?`
- Download/delete/thumbnail lookup by `id` only

## Fix

1. Add `org_id INTEGER NOT NULL DEFAULT 0` via CREATE + ALTER migration
2. Require `kernel.OrgIDFromContext` on all handlers
3. Set `org_id` on upload; scope list/get/delete/thumbnail by caller org
4. Cross-tenant isolation tests

## Verify

- `go test ./components/storage/ -run TestStorageOrgIsolation -v`
- `go test ./components/storage/ -v`
- `go vet ./components/storage/...`
