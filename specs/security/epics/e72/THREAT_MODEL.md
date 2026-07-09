# Threat Model — e72: Monitoring — AI-Assisted Incident Response & Deploy Observability

**Generated:** 2026-07-08  
**Epic:** e72 — Monitoring — AI-Assisted Incident Response & Deploy Observability  
**Stories:** e72s01–e72s04 (10 BCPs)  
**Risk Level:** **MEDIUM** (LLM data exposure + cross-tenant read paths; no new public write surface)  
**ADR:** `specs/adr/0007-e72-observability-seams.md`

---

## Executive Summary

Epic e72 adds deploy pipeline timing, correlated event timelines, AI-assisted deploy failure diagnosis, and automated alert investigation. Most stories extend existing authenticated API surfaces (`GET /api/deploy/:id`, monitoring incident endpoints). Primary risks: **LLM prompt leakage** (build logs, metrics), **IDOR on deploy/incident reads**, and **bus event enrichment exposing tenant data to wrong subscribers**.

**Verdict:** Proceed with implementation. Deny-by-default auth on all new endpoints; scope queries by org/site; never send secrets to LLM; redact tokens in logs.

---

## Surface Area

| Story | Component | Endpoints / Events | Attack Vectors |
|-------|-----------|-------------------|----------------|
| e72s02 | `components/deploy` | `GET /api/deploy/:id` (+ `pipeline_timeline`) | Info disclosure via IDOR on deploy ID |
| e72s03 | `components/internal/eventrecorder`, monitoring | Event bus `Record`/`Query`, SSE fan-out | Cross-tenant event leakage if `site_id` filter bypassed |
| e72s01 | `components/monitoring`, `internal/llm` | `GET /api/deploy/:id/diagnosis`, `deploy.failed` hook | LLM exfiltration of build logs/secrets; prompt injection |
| e72s04 | `components/monitoring` | `GET /api/monitoring/incidents/:id/investigation`, `alert.triggered` | IDOR on incidents; unbounded evidence gather SSRF/log scrape |

**Non-exposed:** Pipeline timeline JSON is read-only metadata; no new mutating routes in e72s02.

---

## Threat Analysis by Story

### e72s02 — Deploy Pipeline Timeline (current story)

| Threat | Severity | Mitigation |
|--------|----------|------------|
| **IDOR — read another org's deploy timeline** | **HIGH** | Reuse existing deploy auth gate on `handleDeployByID`; scope by org/site |
| **Timeline JSON injection / oversized payload** | LOW | Marshal via `encoding/json`; column TEXT with reasonable size; no user-controlled keys |
| **Timing oracle on internal stages** | LOW | Same access as deploy status today |

### e72s01 — AI Deploy Failure Diagnosis

| Threat | Severity | Mitigation |
|--------|----------|------------|
| **Secrets in build logs sent to LLM** | **HIGH** | Strip `API_KEY`, `SECRET`, `TOKEN`, `PASSWORD` lines before prompt; document in llm package |
| **Prompt injection via build output** | MEDIUM | System prompt treats build log as untrusted data; no tool execution from LLM |
| **LLM API key exposure** | **HIGH** | Env-only (`BIGBASE_LLM_API_KEY`); never log request bodies |
| **Unauthenticated diagnosis endpoint** | **HIGH** | Same auth as deploy GET; 404 when reader nil |

### e72s03 — Correlated Event Timeline

| Threat | Severity | Mitigation |
|--------|----------|------------|
| **Cross-site event correlation** | **HIGH** | Filter `Query` by `site_id`; reject empty site scope for tenant queries |
| **Event store DoS (unbounded writes)** | MEDIUM | FIFO cap on recorder; bounded retention |

### e72s04 — Alert Investigation

| Threat | Severity | Mitigation |
|--------|----------|------------|
| **IDOR on incident investigation** | **HIGH** | Scope by org; incident FK integrity |
| **Evidence gather over-breadth** | MEDIUM | Fixed time window + metric scope; no arbitrary URL fetch |

---

## Vulnerability Categories (Scan Checklist)

| Category | Applicable? | Notes |
|----------|-------------|-------|
| Auth bypass | **YES** | New GET routes must inherit deploy/monitoring auth |
| IDOR / cross-tenant | **YES** | Primary concern for all read endpoints |
| SQL injection | Checked | Bound parameters; JSON column marshaled in Go |
| Secrets exposure | **YES** | Build logs → LLM (e72s01); Bearer tokens in event payloads |
| SSRF | LOW | e72s04 evidence gather — SQL/log only, no arbitrary HTTP |
| Prompt injection | **YES** | e72s01/e72s04 LLM paths |

---

## Mitigation Summary

1. **Auth inheritance** — new fields/endpoints behind existing deploy and monitoring auth middleware.
2. **Tenant scoping** — all queries include org/site from authenticated context.
3. **LLM hygiene** — secret-line stripping, env-only API keys, no logging of prompts/responses at info level.
4. **Event bus contracts** — enrich with `site_id` where available; subscribers must not log raw payloads with secrets.
5. **Integration tests** — 401/403 matrix for new routes; IDOR negative tests where feasible.

---

## Out of Scope (Accepted Risks)

- Rate limiting on diagnosis/investigation endpoints
- OpenTelemetry distributed tracing
- Per-org LLM budget caps

---

## Implementation Guidance

| Step | Security requirement |
|------|---------------------|
| e72s02 | No new route — extend existing GET; verify auth unchanged |
| e72s01 | Redact secrets before `llm.Complete`; wire reader only when LLM configured |
| e72s03 | `site_id` required on correlation queries |
| e72s04 | Investigation keyed to `incident_id`, not rule ID |

**Verdict:** ✅ Proceed — e72s02 is low-risk (read-only metadata on existing endpoint).
