# Threat Model — e48: Live Surface Hardening

## Epic Scope

| Story | Description | BCPs |
|-------|-------------|------|
| e48s01 | Block `.git` exposure and harden `/health` endpoint | 2 |
| e48s02 | Add missing security headers and CI scanning | 2 |
| e48s03 | DAST baseline scan and scheduled automation | 2 |

**Story in focus:** e48s01 (this cycle)

## Attack Surface Analysis

### Surface Area

| Entry Point | Protocol | Auth Required | Current Risk |
|-------------|----------|---------------|--------------|
| `/.git/*` paths | HTTP GET | No | **HIGH** — git metadata exposure |
| `/health` | HTTP GET | No | **MEDIUM** — component architecture reconnaissance |
| `/` (home) | HTTP GET | No | LOW — public landing page |
| `/docs` | HTTP GET | No | LOW — public docs |
| `/api/*` | HTTP | Varies | MEDIUM (per-component) |

### Assets at Risk

| Asset | Impact of Compromise |
|-------|---------------------|
| Git repository metadata | Source code structure, CI config exposure |
| Component architecture map | Reconnaissance enabler for targeted attacks |
| Service availability | `/health` endpoint used by monitoring/load balancers |

## Vulnerability Categories

### 1. Information Disclosure — `.git` Path Exposure (HIGH)

**CWE-200:** Exposure of Sensitive Information to an Unauthorized Actor

Attackers can probe for `.git/config`, `.git/HEAD`, `.git/objects/` and related paths. While a live server should not have the actual `.git/` directory accessible, the attack probe itself is a reconnaissance vector:

- **Exploit scenario:** Automated scanner probes `/.git/config` → returns 200 with git repo metadata if `.git/` is mirror-deployed, or 404 if blocked after fix
- **Without fix:** If the `.git/` directory somehow exists in the deployment, attackers can extract the full commit history, source code, and embedded secrets
- **With fix (404):** No information leakage; probe returns indistinguishable 404 from other 404 paths
- **Confidence:** 9/10 — well-known attack pattern, concrete exploit path
- **Severity:** HIGH (before fix) / LOW (after fix, residual: timing side-channel)

### 2. Information Disclosure — `/health` Component Map (MEDIUM)

**CWE-200:** Exposure of Sensitive Information

The `/health` endpoint currently returns a complete component architecture map including 19 components with their names, versions, running status, dependencies, and registered hooks — all without authentication.

- **Exploit scenario:** `curl https://bigbase.click/health` → returns JSON with full architecture. Attacker maps the exact component versions (e.g., `proxy: 0.1.0`, `auth: 0.1.0`) to look up known CVEs for specific versions
- **Without fix:** Any internet client can enumerate the full internal architecture
- **With fix (Bearer token):** Only clients possessing `HEALTH_TOKEN` can access the detailed component map. Unauthenticated requests return 401
- **Confidence:** 8/10 — clear reconnaissance pattern, no chained exploit required
- **Severity:** MEDIUM (reconnaissance enabler, not direct compromise)

### 3. Auth Bypass — Token Implementation (MEDIUM)

**CWE-287:** Improper Authentication

The HEALTH_TOKEN is an optional bearer token. Implementation risks:

- **Timing attack:** Token comparison uses `strings.EqualFold` or `==` instead of `crypto/subtle.ConstantTimeCompare` → attacker can brute-force token by measuring response timing
- **Token in logs:** If the token appears in request logs (e.g., if HEALTH_TOKEN is passed via environment variable that gets logged at startup), it's exposed retroactively
- **Token rotation:** No built-in token rotation mechanism; long-lived static tokens increase exposure window
- **Confidence:** 8/10 — timing attack vector is standard; logging risk depends on implementation
- **Severity:** MEDIUM

### 4. Middleware Bypass — `/.git` Path Variants (MEDIUM)

**CWE-172:** Encoding Error

The `.git` middleware uses string matching rather than canonical path inspection. Bypass techniques:

- **Double encoding:** `/%2e%67%69%74/config` (URL-encoded `.git`)
- **Path traversal:** `/foo/../.git/config` (if normalised before middleware)
- **Case variation:** `/.GIT/config`, `/.Git/config` (if filesystem is case-insensitive)
- **Confidence:** 7/10 — depends on middleware position and path normalization
- **Severity:** LOW-MEDIUM (addressed by middleware placement after request ID but before mux, and using `strings.Contains` on raw `r.URL.Path`)

### 5. Availability — Health Probe Breakage (LOW)

**CWE-770:** Allocation of Resources Without Limits or Throttling

Adding auth to `/health` could break existing monitoring if HEALTH_TOKEN is not communicated to monitoring tools.

- **Exploit scenario:** Monitoring system probes `/health` → 401 → monitoring marks service as DOWN → paging, alert fatigue, potential self-DoS
- **Mitigation:** Token is optional. Deployments that need external health probes can leave HEALTH_TOKEN unset
- **Confidence:** 6/10 — operational risk, not a security vulnerability
- **Severity:** LOW

## Risk Scoring

| Vulnerability | CWE | Confidence | Severity | Risk Score |
|--------------|-----|-----------|----------|------------|
| `.git` path exposure (pre-fix) | 200 | 9/10 | HIGH | 9 |
| `/health` component info leak (pre-fix) | 200 | 8/10 | MEDIUM | 6 |
| Token timing attack | 287 | 8/10 | MEDIUM | 6 |
| Middleware bypass (path variants) | 172 | 7/10 | LOW | 4 |
| Health probe breakage | 770 | 6/10 | LOW | 2 |

**Overall Epic Risk:** MEDIUM-HIGH (reconnaissance surface for future targeted attacks)

## Mitigation Guidance

### Required (in-scope for e48s01)

| Finding | Mitigation | Verification |
|---------|-----------|-------------|
| `.git` path exposure | `gitPathBlockMiddleware` returning 404 for any path containing `/.git` | `curl localhost:9999/.git/config → 404` |
| `/health` info leak | Bearer token auth via `HEALTH_TOKEN` env var | `curl -H "Authorization: Bearer $TOKEN" localhost:9999/health → 200` |
| Token timing attack | Use `crypto/subtle.ConstantTimeCompare` for token comparison | Code review + test coverage |
| Token in logs | Ensure HEALTH_TOKEN is NOT logged at startup; only log `token configured: true/false` | Code review |

### Recommended (future hardening)

| Finding | Epic Assignment | Priority |
|---------|----------------|---------|
| Security headers (CSP, HSTS, XFO) | e48s02 | Immediate |
| CI secret scanning (gitleaks) | e48s02 | Immediate |
| DAST baseline scan | e48s03 | Short-term |
| Token rotation mechanism | Future epic | Medium-term |
| Structured secret management | e59 | Long-term |

### Hard Exclusions (not reported)

- Rate limiting on `/health` — operational, not a code vulnerability (exclusion #3)

## Security Task Annotations

For `plan-work` and `develop-tdd`:

| Task | Security Field | Threat Addressed |
|------|---------------|------------------|
| Add HealthToken to Options + Proxy struct; wire in New() | none | Enabler for token auth |
| Add gitPathBlockMiddleware returning 404 for `/.git` | medium | `.git` exposure, middleware bypass |
| Add Bearer-token auth to handleHealth | medium | `/health` info leak, timing attack |
| Update existing health endpoint tests for no-token mode | low | Health probe breakage |
| Full proxy test suite | low | Regression prevention |

## Review History

| Date | Reviewer | Notes |
|------|----------|-------|
| 2026-06-27 | build-epic (automated) | Initial threat model for e48s01 |

---

*Generated during build-epic Step 0. See [security-review reference docs](REFERENCE-false-positives.md) for exclusion rules.*
