# Threat Model — e68 Native Database Connection String Env Var

**Epic:** e68 — Deploy — Native Database Connection String Env Var  
**Story:** e68s01 — Inject DB Connection String into App Environment  
**Date:** 2026-07-07  
**Risk Level:** Medium

## Surface Area

| Surface | Description |
|---------|-------------|
| Deploy runtime env injection | `startApp` passes `DB_PATH` (SQLite) or `DATABASE_URL` (PostgreSQL) to child processes |
| Main config wiring | `main.go` forwards active `db-driver` and DSN into `deploy.Options` |
| Deployed app processes | Node/SvelteKit SSR, Go, Python apps receive native DB credentials in their environment |

## Vulnerability Categories

| Category | Applicability | Notes |
|----------|---------------|-------|
| Secrets exposure (CWE-200) | **Primary** | Connection strings (especially Postgres URLs with credentials) are injected into app env |
| Privilege escalation | Low | Apps already run as same OS user as BigBase; DB_PATH grants file-level access to SQLite |
| Command injection | None | DSN is not passed through a shell; set as `KEY=value` env pair |
| Path traversal | Low | SQLite path is server-controlled, not user-supplied at injection time |

## Risk Assessment

**Medium** — Exposure is intentional: deployed apps need direct DB access for SSR. The trust boundary is the site owner’s own application code, not arbitrary tenants on a shared host.

### Attack Scenarios

1. **Malicious app reads env** — A deployed app logs `DATABASE_URL` or reads `DB_PATH`. *Expected behavior* for owner-operated apps; mitigated by site-scoped deploy isolation.
2. **SQLite file tampering** — App with `DB_PATH` can read/write the platform DB file. *Acceptable* when the app is trusted; same trust model as REST API access.
3. **Credential leak via logs** — Deploy logs must not echo injected env values. *Mitigation:* never log DSN values; only log that injection occurred.

## Mitigations

1. Inject only server-resolved DSN (from `BIGBASE_DB_DSN` / `--db-dsn`), never from user-supplied site env vars.
2. Use absolute paths for SQLite `DB_PATH` so apps resolve the file correctly without path confusion.
3. Do not log connection string values in deploy or supervisor logs.
4. Site env var system remains separate; `DATABASE_URL` from site settings can override manifest vars but native injection uses platform DSN (document precedence: site env vars appended after native — existing order preserved; site owner can override if needed).

## Verdict

**Proceed** — Feature is required for SvelteKit SSR. Risks are bounded by existing deploy trust model. No additional auth gate needed beyond current site ownership controls.
