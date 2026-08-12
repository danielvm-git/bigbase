# Traceability Matrix

Generated: 2026-08-12T00:24:37.616472+00:00

## Story Coverage

| Story | Title | Files | Tests | Status |
|---|---|---:|---:|---|
| e17s01 | e17s01 | 0 | 0 | Dark |
| e17s02 | e17s02 | 0 | 0 | Dark |
| e17s03 | e17s03 | 0 | 0 | Dark |
| e17s04 | e17s04 | 0 | 0 | Dark |
| e17s05 | e17s05 | 0 | 0 | Dark |
| e17s06 | e17s06 | 0 | 0 | Dark |
| e17s07 | e17s07 | 0 | 0 | Dark |
| e17s08 | e17s08 | 0 | 0 | Dark |
| e17s09 | e17s09 | 0 | 0 | Dark |
| e17s10 | e17s10 | 0 | 0 | Dark |
| e17s11 | e17s11 | 0 | 0 | Dark |
| e17s12 | e17s12 | 0 | 0 | Dark |
| e17s13 | e17s13 | 0 | 0 | Dark |
| e17s14 | e17s14 | 0 | 0 | Dark |
| e17s15 | e17s15 | 0 | 0 | Dark |
| e17s16 | e17s16 | 0 | 0 | Dark |
| e17s17 | e17s17 | 0 | 0 | Dark |
| e17s18 | e17s18 | 0 | 0 | Dark |
| e43s01 | e43s01 | 1 | 1 | Covered |
| e47s01 | Wire rate limiter to auth public endpoints | 0 | 0 | Dark |
| e48s01 | Block .git exposure and harden /health endpoint | 0 | 0 | Dark |
| e48s02 | Add missing security headers and CI scanning | 0 | 0 | Dark |
| e48s03 | DAST baseline scan and scheduled automation | 0 | 0 | Dark |
| e49s01 | Fix anonymous tokens — add org_id context | 0 | 0 | Dark |
| e49s02 | Fix OAuth redirect URI — use configured base URL | 0 | 0 | Dark |
| e49s03 | Add path traversal defense to file downloads | 0 | 0 | Dark |
| e50s01 | Persist JWT secret — env var with fallback | 0 | 0 | Dark |
| e50s02 | Configurable token lifetimes | 0 | 0 | Dark |
| e50s03 | Refresh token revocation — per-user invalidation | 0 | 0 | Dark |
| e51s01 | Design Tokens & Theme System | 0 | 0 | Dark |
| e51s02 | Core Component Library | 0 | 0 | Dark |
| e51s03 | Layout Components | 0 | 0 | Dark |
| e51s04 | Console Page Templates | 0 | 0 | Dark |
| e51s05 | Accessibility & Responsive Audit | 0 | 0 | Dark |
| e51s06 | Extended Component Library | 0 | 0 | Dark |
| e51s07 | Data Visualization & Editor Components | 0 | 0 | Dark |
| e56s01 | Move OTP store from in-memory to database | 0 | 0 | Dark |
| e56s02 | Audit log for security-sensitive operations | 0 | 0 | Dark |
| e57s01 | Kernel Interface Hardening | 0 | 0 | Dark |
| e57s02 | Projects table — backend CRUD | 0 | 0 | Dark |
| e57s03 | Database isolation per project | 0 | 0 | Dark |
| e57s04 | Auth namespacing and Site→Project backfill | 0 | 0 | Dark |
| e57s05 | Deepen: EnvResolver seam for projects | 0 | 0 | Dark |
| e58s01 | Project Scoping Admin UI | 0 | 0 | Dark |
| e59s01 | SQL-over-HTTP Transport (Neon wire format) | 0 | 0 | Dark |
| e59s02 | Database Branching (SQLite snapshot, blocks e65) | 0 | 0 | Dark |
| e59s03 | Better Auth Backend Engine (wire-compat endpoints) | 0 | 0 | Dark |
| e59s04 | MCP Database Migration Tools | 0 | 0 | Dark |
| e60s01 | Rename /deploy/* → /sites/* routes with 301 redirect shim | 0 | 0 | Dark |
| e61s01 | Encrypted project secrets store & CRUD API | 0 | 0 | Dark |
| e61s02 | Project Secrets admin UI tab | 0 | 0 | Dark |
| e62s01 | e62s01 | 0 | 0 | Dark |
| e62s02 | e62s02 | 0 | 0 | Dark |
| e63s01 | e63s01 | 0 | 0 | Dark |
| e63s02 | e63s02 | 0 | 0 | Dark |
| e64s01 | e64s01 | 0 | 0 | Dark |
| e64s02 | e64s02 | 0 | 0 | Dark |
| e65s01 | e65s01 | 0 | 0 | Dark |
| e65s02 | e65s02 | 0 | 0 | Dark |
| e66s01 | e66s01 | 0 | 0 | Dark |
| e66s02 | e66s02 | 0 | 0 | Dark |
| e66s03 | e66s03 | 0 | 0 | Dark |
| e67s01 | MCP create_repo tool — register a GitHub repo with BigBase | 0 | 0 | Dark |
| e67s02 | MCP create_site tool — provision a site and get site_id back | 0 | 0 | Dark |
| e67s03 | MCP provision_ci_credentials — site-scoped keys via existing API key infrastructure | 0 | 0 | Dark |
| e67s04 | MCP get_ci_template — return canonical CI/CD workflow YAML | 0 | 0 | Dark |
| e70s01 | Persist deploy defaults on site record | 0 | 0 | Dark |
| e70s02 | bigbase.toml manifest parsing in repository root | 0 | 0 | Dark |
| e70s03 | Parameterized CI templates with deploy defaults | 0 | 0 | Dark |
| e70s04 | static-sidecar app profile | 0 | 0 | Dark |
| e71s01 | Site auth policy schema | 1 | 0 | Covered |
| e71s02 | Proxy JWT validation for protected static paths | 1 | 0 | Covered |
| e71s03 | Passthrough auth injection — X-BigBase-User-ID header | 1 | 0 | Covered |
| e71s04 | MCP set_site_auth_policy tool for agent config | 1 | 0 | Covered |
| e72s01 | MCP HTTP Bearer middleware — validate bb_ org API keys | 0 | 0 | Dark |
| e72s02 | Scope-gated provisioning — mcp:provision scope required for write tools | 0 | 0 | Dark |
| e72s03 | Read-only public tier — list_services, get_ci_template without auth | 0 | 0 | Dark |
| e72s04 | provision_mcp_credentials tool — issue scoped MCP access keys for agents | 0 | 0 | Dark |
| e73s01 | pyproject.toml detection + uv package manager | 0 | 0 | Dark |
| e73s02 | ASGI server (uvicorn) runtime | 0 | 0 | Dark |
| e73s03 | Writable disk + health check polling | 0 | 0 | Dark |
| e73s04 | System dependencies + long-running subprocesses | 0 | 0 | Dark |
| e74s01 | REST endpoints for site deploy keys | 0 | 0 | Dark |
| e74s02 | Deploy Keys tab on Site Detail page | 0 | 0 | Dark |
| e76s01 | Token Lifecycle E2E Tests | 1 | 1 | Covered |
| e77s01 | API Surface E2E Tests | 1 | 1 | Covered |
| e78s01 | Login & Authentication UI — browser E2E | 6 | 6 | Covered |
| e78s02 | Dashboard & Navigation — browser E2E | 2 | 2 | Covered |
| e78s03 | Sites & Deploy UI — browser E2E | 2 | 2 | Covered |
| e78s04 | Data, SQL & Storage — browser E2E | 2 | 2 | Covered |
| e78s05 | Secondary Pages (Functions, Messaging, Users, Settings) — browser E2E | 3 | 3 | Covered |
| e78s06 | DevOps Pages (CI/CD, Monitoring, Forge, Realtime, Events, Git) — browser E2E | 5 | 5 | Covered |
| e79s01 | Consolidate CI/CD workflows with template migration | 0 | 0 | Dark |
| e80s01 | Ubuntu OS security hardening | 0 | 0 | Dark |
| e80s02 | BigBase monitoring alerts and backup automation | 0 | 0 | Dark |
| e80s03 | Contabo VPS operations — snapshots and API monitoring | 0 | 0 | Dark |
| e82s01 | Multipart deploy API + empty-artifact error/log | 0 | 0 | Dark |
| e82s02 | Engine unpack/start; skip host build | 0 | 0 | Dark |
| e82s03 | bigbase-deploy action uploads CI artifact | 0 | 0 | Dark |
| e82s04 | Admin UI upload + deploy log feedback parity | 0 | 0 | Dark |
| e84s01 | Site model + API layer — add csp_policy to DeployDefaults | 0 | 0 | Dark |
| e84s02 | Proxy per-site CSP override at request time | 0 | 0 | Dark |
| e84s03 | Manifest security.csp field + deploy-time propagation | 0 | 0 | Dark |
| e85s01 | Landing page theme bootstrap — inline script + accent ramp + CSS selector refactor | 0 | 0 | Dark |
| e85s02 | Accent ramp drift guard + cross-surface parity test | 1 | 1 | Covered |
| e86s01 | Paginated log search API + UI (keyset cursor) | 0 | 0 | Dark |
| e86s02 | Live log streaming via SSE + Logs tab subscribe | 0 | 0 | Dark |
| e86s03 | Org-scoped logs — org_id migration + isolation | 0 | 0 | Dark |
| e86s04 | Deploy-component automatic log ingestion | 0 | 0 | Dark |
| e87s01 | AAA color contrast tokens (1.4.6 Contrast Enhanced) | 0 | 0 | Dark |
| e87s02 | Focus appearance enhanced (2.4.13) | 0 | 0 | Dark |
| e87s03 | Target size enhanced — 44px (2.5.5) | 1 | 1 | Covered |
| e87s04 | Abbreviations, unusual words, reading level (3.1.3/3.1.4/3.1.5) | 0 | 0 | Dark |
| e87s05 | Session timeout warning + re-authentication (2.2.5/2.2.6) | 1 | 1 | Covered |
| e87s06 | Help text + error prevention (3.3.5/3.3.6) | 0 | 0 | Dark |
| e87s07 | Visual presentation + location (1.4.8/2.4.8) | 0 | 0 | Dark |
| e88s01 | Accent-theme light-mode link contrast >=7:1 (1.4.6) | 0 | 0 | Dark |
| e88s02 | Formal conformance exceptions (1.4.8 partial + 2.2.3 security) | 0 | 0 | Dark |
| e88s03 | AAA re-certification — matrix + axe + conformance statement | 0 | 0 | Dark |
| e89s01 | Harden existing site-secret storage and delivery | 0 | 0 | Dark |
| e89s02 | Add organization-scoped Projects and Environments | 0 | 0 | Dark |
| e89s03 | Add versioned envelope-encrypted project secrets | 0 | 0 | Dark |
| e89s04 | Add secret REST API, policies, and audit events | 0 | 0 | Dark |
| e89s05 | Add Admin UI for project secrets | 0 | 0 | Dark |
| e89s06 | Resolve project and site secrets in deployments | 0 | 0 | Dark |
| e89s07 | Secure MCP secret-management tools | 0 | 0 | Dark |

## Gaps

- e17s01: no story-tagged implementation files
- e17s02: no story-tagged implementation files
- e17s03: no story-tagged implementation files
- e17s04: no story-tagged implementation files
- e17s05: no story-tagged implementation files
- e17s06: no story-tagged implementation files
- e17s07: no story-tagged implementation files
- e17s08: no story-tagged implementation files
- e17s09: no story-tagged implementation files
- e17s10: no story-tagged implementation files
- e17s11: no story-tagged implementation files
- e17s12: no story-tagged implementation files
- e17s13: no story-tagged implementation files
- e17s14: no story-tagged implementation files
- e17s15: no story-tagged implementation files
- e17s16: no story-tagged implementation files
- e17s17: no story-tagged implementation files
- e17s18: no story-tagged implementation files
- e47s01: no story-tagged implementation files
- e48s01: no story-tagged implementation files
- e48s02: no story-tagged implementation files
- e48s03: no story-tagged implementation files
- e49s01: no story-tagged implementation files
- e49s02: no story-tagged implementation files
- e49s03: no story-tagged implementation files
- e50s01: no story-tagged implementation files
- e50s02: no story-tagged implementation files
- e50s03: no story-tagged implementation files
- e51s01: no story-tagged implementation files
- e51s02: no story-tagged implementation files
- e51s03: no story-tagged implementation files
- e51s04: no story-tagged implementation files
- e51s05: no story-tagged implementation files
- e51s06: no story-tagged implementation files
- e51s07: no story-tagged implementation files
- e56s01: no story-tagged implementation files
- e56s02: no story-tagged implementation files
- e57s01: no story-tagged implementation files
- e57s02: no story-tagged implementation files
- e57s03: no story-tagged implementation files
- e57s04: no story-tagged implementation files
- e57s05: no story-tagged implementation files
- e58s01: no story-tagged implementation files
- e59s01: no story-tagged implementation files
- e59s02: no story-tagged implementation files
- e59s03: no story-tagged implementation files
- e59s04: no story-tagged implementation files
- e60s01: no story-tagged implementation files
- e61s01: no story-tagged implementation files
- e61s02: no story-tagged implementation files
- e62s01: no story-tagged implementation files
- e62s02: no story-tagged implementation files
- e63s01: no story-tagged implementation files
- e63s02: no story-tagged implementation files
- e64s01: no story-tagged implementation files
- e64s02: no story-tagged implementation files
- e65s01: no story-tagged implementation files
- e65s02: no story-tagged implementation files
- e66s01: no story-tagged implementation files
- e66s02: no story-tagged implementation files
- e66s03: no story-tagged implementation files
- e67s01: no story-tagged implementation files
- e67s02: no story-tagged implementation files
- e67s03: no story-tagged implementation files
- e67s04: no story-tagged implementation files
- e70s01: no story-tagged implementation files
- e70s02: no story-tagged implementation files
- e70s03: no story-tagged implementation files
- e70s04: no story-tagged implementation files
- e72s01: no story-tagged implementation files
- e72s02: no story-tagged implementation files
- e72s03: no story-tagged implementation files
- e72s04: no story-tagged implementation files
- e73s01: no story-tagged implementation files
- e73s02: no story-tagged implementation files
- e73s03: no story-tagged implementation files
- e73s04: no story-tagged implementation files
- e74s01: no story-tagged implementation files
- e74s02: no story-tagged implementation files
- e79s01: no story-tagged implementation files
- e80s01: no story-tagged implementation files
- e80s02: no story-tagged implementation files
- e80s03: no story-tagged implementation files
- e82s01: no story-tagged implementation files
- e82s02: no story-tagged implementation files
- e82s03: no story-tagged implementation files
- e82s04: no story-tagged implementation files
- e84s01: no story-tagged implementation files
- e84s02: no story-tagged implementation files
- e84s03: no story-tagged implementation files
- e85s01: no story-tagged implementation files
- e86s01: no story-tagged implementation files
- e86s02: no story-tagged implementation files
- e86s03: no story-tagged implementation files
- e86s04: no story-tagged implementation files
- e87s01: no story-tagged implementation files
- e87s02: no story-tagged implementation files
- e87s04: no story-tagged implementation files
- e87s06: no story-tagged implementation files
- e87s07: no story-tagged implementation files
- e88s01: no story-tagged implementation files
- e88s02: no story-tagged implementation files
- e88s03: no story-tagged implementation files
- e89s01: no story-tagged implementation files
- e89s02: no story-tagged implementation files
- e89s03: no story-tagged implementation files
- e89s04: no story-tagged implementation files
- e89s05: no story-tagged implementation files
- e89s06: no story-tagged implementation files
- e89s07: no story-tagged implementation files

## Summary

Stories: 16 covered / 110 dark / 126 total.
