---
bug_id: BUG-140
status: fixed
severity: high
scope: messaging
title: "Cross-Tenant Message Leak — messages table has no org_id column"
---

## Summary

The `messages` table has no `org_id` column. The `handleList` endpoint returns **every message ever sent by any org** — including email bodies, SMS texts, push notifications, and Telegram messages with recipient addresses and potentially OTPs.

## Affected Files

- `components/messaging/messaging.go:108-121` — table schema (missing `org_id`)
- `components/messaging/messaging.go:127-137` — `send()` INSERT (no `org_id`)
- `components/messaging/handlers.go:110-141` — `handleList()` SELECT (no `org_id` filter)

## Root Cause

The `messages` table was created without tenant isolation:

```sql
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    channel TEXT NOT NULL,
    to_addr TEXT NOT NULL,
    subject TEXT,
    body TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'sent',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
)
```

The `send()` function inserts without `org_id`, and `handleList()` queries without filtering by caller's org. Any authenticated user can read all messages from all organizations.

## Fix Approach

1. Add `org_id INTEGER NOT NULL DEFAULT 0` column to `messages` table
2. Update `send()` to extract `org_id` from context via `auth.OrgIDFromContext()` and include in INSERT
3. Update `handleList()` to filter `WHERE org_id = ?` using caller's org_id from context
4. Add `OrgID` field to `Message` struct
5. Add cross-tenant isolation test

## Verify

- [x] `go test ./components/messaging/ -run TestCrossTenantMessageIsolation -v` passes
- [x] `go test ./components/messaging/ -v` — all tests pass (22/22)
- [x] `go vet ./components/messaging/` — clean
