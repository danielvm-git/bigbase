---
id: BUG-2026-07-10T160001
date: 2026-07-10
severity: high
priority: critical
scope: auth
status: fixed
cwe: CWE-601
title: OAuth redirect URI uses attacker-controlled Host when publicURL unset
source: registry backfill (BUG-2026-07-28T000005)
---

# OAuth Host header poisoning (CWE-601)

OAuth redirect URI was built from `r.Host` when `publicURL` was unset, allowing an
attacker-controlled `Host` header to steer the OAuth callback to an attacker origin.

See `specs/bugs/registry.yaml` (BUG-2026-07-10T160001) for the authoritative fix
record, files changed, and regression guard. This stub was created by the
2026-07-28 registry integrity backfill so the closed SECURITY bug has a
traceable artifact on disk.
