---
id: BUG-2026-07-10T160005
date: 2026-07-10
severity: high
priority: high
scope: mcp
status: fixed
cwe: CWE-200
title: MCP error details leaked at 13+ sites — raw Go errors exposed to MCP clients
source: registry backfill (BUG-2026-07-28T000005)
---

# MCP error detail leakage (CWE-200)

Raw `fmt.Sprintf("Error: %v", err)` patterns in MCP tool responses leaked internal
Go error details (stack hints, SQL fragments, file paths) to MCP clients across
13+ call sites.

See `specs/bugs/registry.yaml` (BUG-2026-07-10T160005) for the authoritative fix
record. This stub was created by the 2026-07-28 registry integrity backfill so the
closed SECURITY bug has a traceable artifact on disk.
