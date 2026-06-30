# Update Brief — e63: Usage Dashboard

**Prototype file:** `BigBase Console.html`  
**Project:** BigBase Prototype (`ec1480a1`)  
**Effort:** Small — new global page + sidebar entry  
**Epic:** e63-usage-dashboard

---

## What to change

### 1. Sidebar — add Usage under Overview (Global Zone)

```
BEFORE:
Overview
  ○ Dashboard

AFTER:
Overview
  ○ Dashboard
  ○ Usage
```

### 2. New screen — Usage (`/usage`)

```
Usage
Organization resource usage — current billing period

┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐
│ DATABASE   │  │ STORAGE    │  │ USERS      │  │ ACTIVE     │
│ 48 MB      │  │ 124 MB     │  │ 2          │  │ DEPLOYS    │
│            │  │            │  │            │  │ 4          │
└────────────┘  └────────────┘  └────────────┘  └────────────┘

By project
─────────────────────────────────────────────────────────────────
┌────────────────────┬──────────┬──────────┬────────────────────┐
│ Project            │ DB size  │ Storage  │ Active deployments │
├────────────────────┼──────────┼──────────┼────────────────────┤
│ my-db              │ 34 MB    │ 48 MB    │ 3                  │
│ blog-db            │ 14 MB    │ 76 MB    │ 1                  │
│ (unassigned)       │ —        │ —        │ 0                  │
└────────────────────┴──────────┴──────────┴────────────────────┘

Storage by bucket
─────────────────────────────────────────────────────────────────
avatars    [████████████░░░░░░░░]  48 MB  / 200 MB
documents  [███░░░░░░░░░░░░░░░░░]  12 MB  / 200 MB
backups    [██░░░░░░░░░░░░░░░░░░]   8 MB  / 200 MB
```

- 4 stat cards at top (same pattern as existing stat rows)
- "By project" table: shows per-project breakdown. "(unassigned)" row covers sites not linked to any project.
- "Storage by bucket" section: horizontal progress bars (same `bb-progress` style), bucket name + used/total

---

## Mock data to add (`bb/data.js`)

```js
usage: {
  dbSizeMb: 48,
  storageMb: 124,
  totalUsers: 2,
  activeDeploys: 4,
  byProject: [
    { name: 'my-db', dbMb: 34, storageMb: 48, activeDeploys: 3 },
    { name: 'blog-db', dbMb: 14, storageMb: 76, activeDeploys: 1 },
  ],
  buckets: [
    { name: 'avatars', usedMb: 48, totalMb: 200 },
    { name: 'documents', usedMb: 12, totalMb: 200 },
    { name: 'backups', usedMb: 8, totalMb: 200 },
  ]
}
```
