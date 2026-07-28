---
id: BUG-144
date: 2026-07-23
severity: high
priority: critical
scope: auth
status: fixed
cwe: CWE-942
title: Insecure postMessage targetOrigin during OAuth popup — JWT leak to any origin
source: registry backfill (BUG-2026-07-28T000005)
---

# postMessage JWT leak via OAuth popup (CWE-942)

The OAuth popup callback posted the JWT to a `postMessage` target without pinning
`targetOrigin`, so any origin that could open the popup window (or intercept the
message) could read the JWT.

See `specs/bugs/registry.yaml` (BUG-144) for the authoritative fix record,
files changed, and regression guards. This stub was created by the 2026-07-28
registry integrity backfill so the closed SECURITY bug has a traceable artifact
on disk.
