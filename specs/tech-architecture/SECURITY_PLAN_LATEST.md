# BigBase — Security Plan & Threat Model

**type:** security-plan
**context:** infra
**status:** active
**last-updated:** 2026-11-06

## 1. System Threat Model

BigBase is a single-binary Backend-as-a-Service (BaaS) running on host systems (VPS) with embedded SQLite and PostgreSQL support.

### Trust Boundaries
- **External Client to Proxy:** Public HTTPS traffic terminated via automated ACME TLS.
- **Proxy to Internal Components:** Event-bus-mediated request dispatching, authentication token verification, and route-policy enforcement.
- **Tenant Isolation:** Multi-tenant database queries scoped by `org_id` and `project_id`. All queries MUST be parameterized (no dynamic SQL concatenation).
- **Secret & Key Hierarchy:** AES-256-GCM envelope encryption for sensitive variables. Root encryption key provided via `BIGBASE_ROOT_ENCRYPTION_KEY` environment variable. Plaintext keys or secrets are never persisted or logged.

### Threat Categories & Mitigations (STRIDE)

| Threat | Category | Mitigation |
|---|---|---|
| **Spoofing** | Identity spoofing | HS256 JWT tokens with HMAC validation; bcrypt password hashing; DB-backed OTP and API key verification. |
| **Tampering** | Data tampering | Parameterized SQL queries; AES-256-GCM authenticated ciphertext; strict JSON schema validation. |
| **Repudiation** | Action denial | Immutable audit log events emitted on auth mutations and secret access; structured JSON logs with tracing. |
| **Information Disclosure** | Data leakage | Redaction of secrets and credentials from build, runtime, and LLM logs; generic error messages to API clients; `httpOnly`, `Secure`, `SameSite=Strict` cookies. |
| **Denial of Service** | Resource exhaustion | Global and per-route rate limiting; execution timeouts on all HTTP and subprocess operations; connection pooling. |
| **Elevation of Privilege** | Authorization bypass | Role-based permission checks; organizational tenant isolation; MCP tool parameter validation with authenticated principal binding. |

## 2. Hard Security Gates

- **Static Analysis:** `gosec` scan on all Go code (`npm run preflight:secscan`).
- **Secret Detection:** `gitleaks detect` run before commit and in CI pipeline.
- **CodeQL Scanning:** Semantic security query suite in CI (`.github/workflows/codeql.yml`).
- **Dependency Hygiene:** Dependabot version updates with automated audit.

## 3. Active Epic Threat Model

See epic-specific threat models under `specs/security/epics/` (e.g. [`specs/security/epics/e89/THREAT_MODEL.md`](../security/epics/e89/THREAT_MODEL.md)).
