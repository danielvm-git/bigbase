# Threat Model: e86 — Monitoring Logs (Streaming, Pagination, Org Scope, Auto-Ingestion)

**Date**: 2026-08-11
**Epic**: e86 — Monitoring Logs
**Nature**: Production feature + security fix. e86s03 closes an active cross-tenant data leak (BUG-143 class). e86s02 adds a new streaming surface. e86s04 adds an internal ingestion path.
**Risk Level**: HIGH (s03 org-scoping is a security boundary change; s02 adds an SSE surface)

## 1. Attack Surface

| Story | Surface | Auth Gate | New/Existing | Risk |
|-------|---------|-----------|--------------|------|
| e86s03 | `GET /api/monitoring/logs` (org filter added) | Session cookie + org context | Modified | HIGH |
| e86s03 | `POST /api/monitoring/logs` (org_id required) | Session cookie + org context | Modified | HIGH |
| e86s03 | `GET /api/monitoring/logs/{id}` (org scoped) | Session cookie + org context | Modified | HIGH |
| e86s03 | `monitoring_logs.org_id` migration | — (schema) | Modified | MEDIUM |
| e86s02 | `GET /api/monitoring/logs/stream` (SSE) | Session cookie (same-origin EventSource) | New | MEDIUM |
| e86s04 | deploy event → log ingest (internal bus) | Internal event bus (trusted) | New | MEDIUM |

## 2. Trust Boundaries

```
[User A browser] ──HTTP──> [Proxy/Auth middleware] ──> [monitoring handlers]
                                                            │
                              kernel.OrgIDFromContext (BUG-143) │
                                                            ▼
                                                     [monitoring_logs]
                                                            ▲
[Deploy component] ──event bus (trusted internal)──> [ingest handler]

Boundaries:
- User ↔ API: untrusted; org identity comes ONLY from kernel.OrgIDFromContext (never from request body).
- Deploy ↔ monitoring: trusted internal bus; org_id must be enriched by deploy at emit time (not client-supplied).
- SSE stream: subscriber filters broadcast by its own org; a client must never receive another org's rows.
```

## 3. Vulnerabilities & Mitigations

| # | Category | Finding | Mitigation | Verify |
|---|----------|---------|-----------|--------|
| 1 | CWE-639 Auth Bypass | `handleLogSearch`/`handleLogByID` without `org_id` filter → any org reads all | `WHERE org_id = ?` from context; NULL-org rows never returned | e86s03t3 |
| 2 | CWE-862 Missing Auth | `handleLogCreate` without org → un-scoped row | 401 `org_id required` when context missing (BUG-143) | e86s03t2 |
| 3 | CWE-639 Cross-Org byID | `GET /logs/{id}` leaks another org's row | `WHERE id = ? AND org_id = ?` | e86s03t3, e86s03t4 |
| 4 | CWE-200 Exposure | Pre-migration rows (NULL org) exposed | NULL-org rows treated platform-internal; tenant queries always filter `org_id = ?` (NULL never matches) | e86s03t4 |
| 5 | CWE-200 SSE Leak | Log stream broadcasts all orgs to one subscriber | Subscriber carries org; broadcast filtered (or subscribe-with-org gating) | e86s02t5 (post-s03) |
| 6 | CWE-269 Spoofed Org in Ingestion | Ingest row org_id taken from event data that an attacker could influence | org_id enriched by deploy at emit from internal context — never from deployment request body; ingest warns when org_id absent | e86s04t2, e86s04t3 |
| 7 | CWE-285 Route Collision | `/api/monitoring/logs/stream` swallowed by `/api/monitoring/logs/` catch-all → 404 or wrong handler | Register exact `GET /api/monitoring/logs/stream` pattern (Go 1.22+ ServeMux most-specific wins) | e86s02t2 |
| 8 | CWE-89 SQL Injection | `?q=` / `?cursor=` interpolated into SQL | Parameterized `?` binds only (repo SQL-safety doctrine) — cursor/query are bound values, never string-concatenated | e86s01t1 |

## 4. Risk Assessment

| Dimension | Rating | Rationale |
|-----------|--------|-----------|
| Production Risk | HIGH before s03 | Active cross-tenant log leak (any auth user reads all orgs' logs) |
| Production Risk | LOW after s03 | Isolation matches alerts (BUG-143 pattern, proven) |
| Ingestion Risk | MEDIUM | Internal bus is trusted; the failure mode is missing org enrichment (row becomes platform-internal, not a leak) |
| Regression Risk | MEDIUM | Migration on existing installs; evidence.go reader must stay green |

## 5. Security Gate

- e86s03 tasks carry `security: high` → verify steps MUST include "no new security findings in affected paths"
- e86s04 tasks carry `security: medium` → same gate
- e86s02/e86s01 `security: low` — parameterized queries + same-origin cookie auth (no new boundaries)

## 6. CWE Mapping

1-4 → CWE-639/862/200 (org isolation) · 5 → CWE-200 · 6 → CWE-269 · 7 → CWE-285 · 8 → CWE-89
