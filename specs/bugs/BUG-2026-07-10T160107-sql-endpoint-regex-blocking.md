---
id: BUG-2026-07-10T160107
date: 2026-07-10
severity: medium
priority: medium
scope: api
status: fixed
title: SQL endpoint regex compiled per request — overhead (partial state documented)
source: registry backfill (BUG-2026-07-28T000005)
---

# SQL endpoint regex blocking — per-request compilation (PARTIAL)

The blocking/safety regexes for the SQL API endpoint were compiled with
`regexp.Compile`/`MustCompile` inside the hot path (per request), adding avoidable
allocation and CPU on every SQL API call.

The 2026-07-28 certification sweep recorded this as PARTIAL: pre-compiled package
patterns exist in `components/api/api.go` and the 83-test suite passes, but full
hot-path audit was not completed.

See `specs/bugs/registry.yaml` (BUG-2026-07-10T160107) for the authoritative fix
record. This stub was created by the 2026-07-28 registry integrity backfill so the
partial-fix state has a traceable artifact on disk.
