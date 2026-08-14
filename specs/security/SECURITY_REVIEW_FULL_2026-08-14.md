# Full-Repository Security Review — 2026-08-14

**Branch:** `main` (clean working tree)
**Commit:** `7ecb579cd` (fix(deploy): atomic terminal transitions)
**Reviewer:** AI agent, `security-review` skill 5-phase scan (confidence threshold ≥ 8/10)
**Scope:** Full cybersecurity assessment — Go backend (21 components), React UI (`ui/`), auth SDK packages (`packages/`), developer tooling (`scripts/`), CI/CD (`.github/workflows/`), infrastructure (Terraform + cloud-init)
**Method:** 4 parallel domain scans (backend auth, backend HTTP surface, frontend/SDK, dev tooling/infra) + direct source verification of every HIGH+ finding

---

## Phase 1 — Scope Resolution

| Domain | Path(s) | Size / Notes |
|--------|---------|--------------|
| Go backend | `main.go`, `adapters.go`, `kernel/`, `components/*` (21 components) | ~1,208 Go files |
| Frontend | `ui/src` | React SPA, embedded via `ui/embed.go` |
| SDK packages | `packages/auth*` (7 packages) | Published auth SDKs (react, vue, svelte, astro, next) |
| Dev tooling | `scripts/` (30+ files), `.github/workflows/` (4) | Shell, Python, JS |
| Infrastructure | `*.tf`, `cloud-init.yaml.tpl`, `infra/observability/` | OCI + VPS provisioning |
| Local artifacts | `bigbase.db`, `.envrc`, `.gitleaks.toml` | Checked for committed secrets |

---

## Phase 2 — Context Research

The platform is a multi-tenant PaaS (orgs, sites, deployments) with:

- **Auth model:** JWT (HS256) + org API keys (`bb_`) + site deploy keys (`bb_dep_`) + anonymous scoped tokens; bcrypt credentials; OAuth (Google) with signed state; magic links; phone OTP
- **Data:** SQLite (`bigbase.db`), org-scoped rows (`org_id`), secrets behind AES-256-GCM envelope encryption
- **Execution surfaces:** CI runner (`components/cici` — tenant workflow YAML), deploy engine (clone/build/serve), functions runtime (goja), reverse proxy to loopback deployments
- **Prior hardening precedent:** `PublicURLOrDefault` (CWE-601 fix for OAuth), BUG-129 SQL OR-bypass fix, BUG-143 alert org-scoping fix, SHA-pinned GitHub Actions, gitleaks preflight

---

## Phase 3 + 4 — Findings (false-positive filtered, confidence ≥ 8/10)

Findings below survived the exclusion rules (DOS, rate-limiting, theoretical hardening, docs-only, etc. are suppressed) and were each verified directly in source.

### 🔴 CRITICAL

---

#### C-1 — CI step execution is tenant-controlled `sh -c` on the host (RCE)

**`components/cici/runs.go:103-110`** — Severity: CRITICAL — CWE-78 (OS Command Injection) — Confidence: 10/10

**Description:** `executeStep` runs tenant-authored workflow `steps[].run` values via `exec.CommandContext(ctx, "sh", "-c", command)`. The only guard is a regex blacklist (`cici.go:26` `dangerousCmdPatterns`) matching `curl|wget|nc|ncat|netcat|ssh| | sh| | bash|eval|exec|$( |backtick`. Word-boundary regex blacklists are trivially bypassed by constructs the pattern does not name: `python3 -c '…'`, `node -e`, `perl -e`, `php -r`, `cu""rl`, `cu''rl`, `busybox nc`, `printf <b64> | python3` (only pipes to `sh`/`bash` are blocked, not to other interpreters), `awk 'system(…)'`, `find -exec`, and many more.

**Exploit scenario:** Any authenticated tenant who owns a git repo (`verifyRepoOwnership` passes for their own repo) does `PUT /api/cici/{repo}/workflows` with YAML containing `run: python3 -c 'import os; os.system("id > /tmp/pwned")'`, then `POST /api/cici/{repo}/workflows/{id}/run`. The step executes unsandboxed as the bigbase server process user on the shared host. Impact: full platform compromise — read `bigbase.db` (all tenants), exfiltrate `BIGBASE_ROOT_ENCRYPTION_KEY` and site env secrets from process environment, pivot into every deployment, establish persistence.

**Reachability:** Direct. Both endpoints registered in `cici.Handler()` (cici.go:168-176), routed at `main.go:590` behind `authComp.Middleware`.

**Recommendation:** The blacklist is not a control and cannot be made one. Tenant-supplied commands must execute in an isolated sandbox (container/namespace per run, no host network, no host mounts, restricted egress) or be replaced by a fixed allowlisted step vocabulary (`npm run build`, `go build ./...`) executed with fixed argv via `exec.Command`. No middle ground preserves the current trust model.

---

### 🟠 HIGH

---

#### H-1 — Site deploy keys (`bb_dep_`) bypass every ownership check on `/api/collections/` (cross-tenant IDOR)

**`components/auth/middleware.go:64-75` + `components/api/api.go:233-244, 305, 399, 433-445`** — Severity: HIGH — CWE-639 (IDOR) / CWE-284 — Confidence: 9/10

**Description:** In `auth.Middleware`, a `bb_dep_` token resolves to a site ID and proceeds with only `kernel.WithSiteID(ctx, siteID)` — **no org ID and no user identity** in context. Every ownership guard in the collections API is written conditionally: `if orgID, ok := OrgIDFromContext(...); ok { … }`. When org context is absent, the guard is **skipped entirely**, not failed:

- `listRecords` (api.go:233-244): `WHERE org_id = ?` clause omitted → returns **every tenant's records**
- `getRecord` (api.go:305): ownership comparison skipped → reads any org's record by ID
- `updateRecord` (api.go:399) / `deleteRecord` (api.go:433-445): same bypass → modify/delete any org's records

The read-only restriction in middleware applies only to `role == "anonymous"` JWTs, not to site keys, so all methods are permitted. New records created this way are tagged `org_id = 0`.

**Exploit scenario:** A CI bot key minted for one site (`POST /api/sites/{id}/deploy-keys`) — intended to be scoped to a single site — is effectively a platform-wide master key. Holder does `GET /api/collections/<any-collection>` with `Authorization: Bearer bb_dep_…` and receives all tenants' data.

**Reachability:** Direct — any request to `/api/collections/...` with a valid site deploy key.

**Recommendation:** Fail closed. In all four handlers, reject requests that carry neither org identity nor a site-scoped authorization for a collection explicitly bound to that site. The secrets component already does this correctly (`secrets_api.go:337-347` requires user or org context and rejects contextless callers) — mirror that pattern.

```go
orgID, hasOrg := OrgIDFromContext(r.Context())
siteID, hasSite := kernel.SiteIDFromContext(r.Context())
if !hasOrg && !hasSite {
    kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
    return
}
```

---

#### H-2 — Stored XSS via storage upload served `inline` as `text/html` on the platform origin

**`components/storage/storage.go:17-26, 396-398`** — Severity: HIGH — CWE-79 (Stored XSS) — Confidence: 9/10

**Description:** `allowedMimePrefixes` includes `"text/"` (and `"image/"`, which admits `image/svg+xml`). `detectMIME` uses `http.DetectContentType`, which sniffs files whose first 512 bytes look like HTML as `text/html; charset=utf-8`. Downloads are then served with the stored `Content-Type` and `Content-Disposition: inline` (`storage.go:396-398`) on the **main platform origin** — the same origin as the admin SPA and all `/api/*` endpoints.

**Exploit scenario:** Any authenticated user uploads a file beginning `<!DOCTYPE html><script>…</script>` via `POST /api/storage/upload`; it passes the `text/` prefix check. Victim navigates to `/api/storage/files/{id}` → attacker JavaScript executes in the platform origin and can call every authenticated API with the victim's credentials (the proxy auth path additionally accepts a `token` cookie, `hosts.go:277`, enabling cookie-scoped credential theft).

**Reachability:** Direct — upload then share the file URL.

**Recommendation:** (a) Force `Content-Disposition: attachment` for all storage downloads, or serve file content from a separate origin/domain; (b) remove `text/html` and `image/svg+xml` from the inline-servable set (serve them as `text/plain` / sanitized, or attachment-only).

---

#### H-3 — Magic-link emails built from attacker-controlled `Host` header → login-token theft

**`components/auth/magiclink.go:94-101`** — Severity: HIGH — CWE-601 (Host-header poisoning) — Confidence: 9/10

**Description:** The magic-link URL is built from the raw request Host header:

```go
scheme := "http"
if r.TLS != nil { scheme = "https" }
link := fmt.Sprintf("%s://%s/api/auth/magic-link/verify?token=%s", scheme, r.Host, token)
```

The codebase already fixed this exact class for OAuth: `PublicURLOrDefault` (auth.go:134-141) documents "It never falls back to the request Host header (CWE-601 open redirect)" — but `magiclink.go` was never migrated. The `redirect_to` parameter is allowlist-checked; the link host is not.

**Exploit scenario:** `POST /api/auth/magic-link/send` with body `{"email":"victim@example.com"}` and `Host: attacker.com`. The victim receives a login email whose link points to `http://attacker.com/api/auth/magic-link/verify?token=<256-bit token>`. On click, the token is delivered to the attacker's server; the attacker redeems it at the real server (`GET /api/auth/magic-link/verify?token=…`) and receives a full `user` JWT for the victim's account (cookie and `redirect_to` SPA flows both usable). Prerequisites: email sending configured, and a deployment that routes arbitrary Host values to the app (typical for most reverse proxies).

**Reachability:** Direct, on the public magic-link endpoint.

**Recommendation:** Build the link from the configured public URL, identical to OAuth:

```go
base := a.PublicURLOrDefault(r) // never trusts r.Host
link := fmt.Sprintf("%s/api/auth/magic-link/verify?token=%s", base, token)
```

---

### 🟡 MEDIUM

---

#### M-1 — Cross-tenant deploy trigger: no repo ownership check

**`components/deploy/gateway.go:54-82` + `components/deploy/engine.go:25-30`** — Severity: MEDIUM-HIGH — CWE-862 / CWE-639 — Confidence: 9/10

**Description:** `HandleCreate` validates ownership only of the optional `site_id` field. When `req.SiteID` is empty (JWT auth) or the caller uses site-deploy-key auth, no check binds `repo_id` to the caller's org. `Trigger` (engine.go:25) only checks that the repo exists (`SELECT name FROM git_repos WHERE id = ?`).

**Exploit scenario:** An authenticated user in org A sends `POST /api/deploy` with org B's `repo_id` and an arbitrary `site_name`. The engine clones, builds, and serves the other tenant's private repository on the shared host, registers a proxy host named after the attacker-chosen `site_name` (host-name squatting — `deploymentURL` derives from it), and allocates a port. Also enables audit-attribution confusion and resource exhaustion.

**Recommendation:** In `HandleCreate`/`Trigger`, verify `git_repos.owner_id` (org) against the caller's org context, mirroring `cici.verifyRepoOwnership`. For site-key auth, require the repo to be bound to the key's site.

---

#### M-2 — Email-verification token has 24 bits of entropy; endpoint unthrottled

**`components/auth/emailverify.go:16-17, 100-143`** — Severity: MEDIUM — CWE-330 / CWE-307 — Confidence: 8/10

**Description:** `verifyTokenLen = 6` hex chars = 16,777,216 possible values. Lookup is `WHERE email_verify_token = ?` with no attempt counter or endpoint-specific rate limit (the per-IP auth rate limit, 60/min/IP default-on, slows but does not bound a distributed effort, and is disableable via `BIGBASE_RATE_LIMIT_ENABLED=false`). Email verification is a security gate: `middleware.go:116-125` blocks all requests from unverified users when an email sender is configured.

**Exploit scenario:** An attacker registers an account under an email they don't control, then brute-forces the 24-bit token (16.7M requests) to mark the account verified, defeating the email-ownership gate.

**Recommendation:** Raise to 32 hex chars (128 bits) to match reset tokens, and add an attempt counter with lockout (the OTP path already implements 3-attempt lockout — reuse it).

---

#### M-3 — Phone OTP written to logs in plaintext ("dev mode" not enforced)

**`components/auth/phone.go:67`** — Severity: MEDIUM — CWE-532 (credentials in logs) — Confidence: 8/10

**Description:** When `phoneSender` is nil but `emailSender` is configured (which is also the only gate enabling `/api/auth/phone/send`, phone.go:23), the OTP is logged verbatim: `a.logger.Info("phone OTP (dev mode...)", "phone", phone, "code", code)`. That code is a full login credential: presented at `POST /api/auth/phone/verify` it mints a JWT for the phone-user account (5-minute validity).

**Exploit scenario:** Anyone with log access (log aggregation, the monitoring log API — see M-8, disk) logs in as any phone number for which a code was requested. The "dev mode" label is not enforced; the branch is live in production whenever email is configured.

**Recommendation:** Gate the log line behind an explicit dev-mode flag that defaults off in production, or remove it entirely.

---

#### M-4 — Password reset does not revoke existing sessions/refresh tokens

**`components/auth/passwordreset.go:196-211`** — Severity: MEDIUM — CWE-613 — Confidence: 8/10

**Description:** `handleResetPassword` updates the password hash and marks the reset token used, but never calls `invalidateAllUserTokens` (refreshtoken.go:204 — exists and is used by `/api/auth/logout-all`). Previously issued JWTs (24h default TTL) and refresh tokens (30 days) remain valid after a reset.

**Exploit scenario:** A user resets their password because a session/token was compromised; the attacker's stolen refresh token keeps minting new access tokens for up to 30 days via `POST /api/auth/refresh`.

**Recommendation:** Call `invalidateAllUserTokens(userID)` inside the reset transaction.

---

#### M-5 — Live GitHub token in plaintext `.envrc`; gitleaks allowlists the path

**`.envrc:2` + `.gitleaks.toml:30-38`** — Severity: MEDIUM — CWE-798 / CWE-312 — Confidence: 9/10

**Description:** A real `gho_`-prefixed OAuth token sits unencrypted in the working tree. Verified NOT tracked in git (`git ls-files`/`git log --all` empty; `.gitignore:66`). However, this repo is consumed by multiple AI agent harnesses (`.claude/`, `.opencode/`, `.mimocode/`), zips, and backups — any of which can exfiltrate it. Compounding: `.gitleaks.toml` allowlists `.envrc` (and `data/`, `.agents/`, `.cursor/`, `.vscode/`, `.pi/`, `mimocode.jsonc`, `components/deploy/data/`) from secret scanning, so an accidental future commit of `.envrc` would pass the preflight silently.

**Recommendation:** **Revoke the token immediately** (`gh auth status` → revoke). Move the value to a secret manager or direnv `dotenv_if_exists` outside the repo. Remove `.envrc` from the gitleaks allowlist (it is gitignored — scanning it costs nothing) and narrow path-based allowlists to exact regex+value pairs like the first three entries.

---

#### M-6 — PR-Agent third-party container pinned by mutable tag; triggerable by any commenter with full secrets

**`.github/workflows/pr-review.yaml:11-12, 25, 28-34`** — Severity: MEDIUM — CWE-1357 / CWE-284 — Confidence: 8/10

**Description:** `uses: docker://pragent/pr-agent:0.36.0-github_action` is a Docker Hub **tag**, not a digest — anyone with push access upstream can republish it. The workflow triggers on `issue_comment: [created]`, which (unlike fork `pull_request`) runs **with full secrets** (`DEEPSEEK_API_KEY`, `GITHUB_TOKEN` with `issues:write` + `pull-requests:write`) even when the comment author has no write access. The only guard is `sender.type != 'Bot'`.

**Exploit scenario:** (a) Supply chain: upstream tag overwritten with a malicious image → any GitHub user comments `/review` on any open PR → attacker code runs in a runner holding secrets and a repo-writable token. (b) Untrusted use: any commenter makes the third-party container process attacker-influenced content with those credentials. Every other action in this repo is SHA-pinned — this is the sole exception.

**Recommendation:** Pin by digest (`docker://pragent/pr-agent@sha256:…`) and require `github.event.comment.author_association` in `[OWNER, COLLABORATOR, MEMBER]` before running.

---

#### M-7 — SSH port 22 open to the entire internet (Terraform + cloud-init)

**`network.tf:37-44` + `cloud-init.yaml.tpl:13`** — Severity: MEDIUM — CWE-668 — Confidence: 8/10

**Description:** The OCI security list permits tcp/22 from `0.0.0.0/0`, and cloud-init adds a host-level iptables ACCEPT for port 22, so tightening the security list alone would not help. Key-only auth is relied on image defaults (no explicit `PasswordAuthentication no`).

**Exploit scenario:** Internet-wide brute-force/credential-stuffing scanners and sshd zero-day class exposure. Direct compromise requires key theft, but the surface is unnecessary.

**Recommendation:** Restrict the tcp/22 ingress to a `var.ssh_allowed_cidrs` (owner IP), remove the blanket port-22 iptables rule from cloud-init, and set `PasswordAuthentication no` explicitly. (Ports 80/443 open to 0.0.0.0/0 are correct for a web server — not a finding.)

---

#### M-8 — Platform-wide monitoring logs readable by any authenticated tenant

**`components/monitoring/monitoring.go:585-618`** — Severity: MEDIUM — CWE-639 — Confidence: 8/10

**Description:** `handleLogSearch` / `handleLogByID` apply no `org_id` filter — unlike alerts, which were org-scoped for BUG-143. Routed behind auth (`main.go:630-631`) but not org-scoped: every authenticated user can read and search the last 100 platform log entries across all tenants. This also amplifies M-3 (logged OTPs become readable cross-tenant).

**Recommendation:** Apply the same org-scoping fix pattern used for alerts in BUG-143.

---

#### LOW (reported for tracking)

| # | Finding | Location | CWE |
|---|---------|----------|-----|
| L-1 | Org invite `expires_at` stored but never checked on acceptance — stale invite links redeemable forever | `components/auth/membership.go:108-130`, `org_http.go:245-274` | CWE-613 |
| L-2 | GitHub webhook signature enforced only when secret configured; public callback upserts attacker-chosen `installation_id` rows influencing clone token selection | `components/github/github.go:403, 192-223` | CWE-346 |
| L-3 | `curl \| bash` installers run as root on the production VPS (nodesource, newrelic, get.docker.com) and unpinned in CI (uv) — repo already demonstrates the correct checksum-verified pattern for gitleaks | `scripts/setup-vps.sh:76,339`, `scripts/setup-newrelic.sh:50`, `.github/workflows/test-build-release.yml:381`, `cloud-init.yaml.tpl:19` | CWE-829 |

---

### ⚠️ Latent (currently unrouted — fix before wiring)

These components define HTTP handlers that are **not registered** in `main.go`/`adapters.go` today, so they are not reachable — but they are one composition-root change away from going live with no auth:

1. **`components/webhooks`** (CWE-918 / CWE-862): `createWebhook` stores client-supplied `url` **and** client-supplied `org_id`; `listWebhooks` returns every tenant's webhook URLs with no org filter; `deleteWebhook` has no ownership check; `Deliver` POSTs to stored URLs via a redirect-following client with no scheme/private-IP validation — a ready-made SSRF primitive. The MCP server already advertises webhooks as a service (`mcp/mcp.go:286`). Required before wiring: derive `org_id` from auth context (never body), add ownership checks, block private-IP/redirect targets in `Deliver`.
2. **`components/backup/handler.go`**: `/api/restore` would replay arbitrary SQL unauthenticated if ever wired as-is.
3. **`components/auth/envvars.go`**: no org scoping or role checks of its own; currently unreachable (grep-verified; only referenced by its own tests).

---

## Verified Clean (Phase 4 exclusions applied)

**Backend auth core**
- SQL injection: all queries in `auth`, `secrets`, `admin`, `api` use bound `?` parameters; the only `fmt.Sprintf`-interpolated fragments are table/field names produced exclusively by alnum+underscore allowlists (`sanitize()` api.go:749-759, `validateCollectionName`)
- JWT: HS256 with `alg` confusion blocked (`verifyJWT` rejects non-HMAC), expiry always set, weak/known-default secrets rejected in production, random secret generation
- Password hashing: bcrypt (DefaultCost) everywhere; no MD5/SHA1/plaintext paths; API keys stored as HMAC-SHA256, raw key returned exactly once, revocation enforced
- OAuth: random+HMAC-signed state cookie with constant-time compare; SPA redirect allowlist by exact scheme+host (incl. subdomain-spoofing prevention); callback URI fixed from configured `PublicURL`, fails closed; popup `postMessage` to exact origin only. Google ID token decoded without local signature verification — sound, since it arrives directly from Google's token endpoint over TLS with the client secret
- Password reset: 128-bit token, 1h expiry, single-use, anti-enumeration (identical responses)
- Refresh tokens: 256-bit, rotation with atomic single-use update and family invalidation on replay
- OTP/magic-link storage: SHA-256 hashes, attempt lockout, TTLs, send rate limits

**Backend HTTP surface**
- Proxy: reverse-proxies only to `http://127.0.0.1:{registered-port}` (Host header never controls target); loopback/IP-literal host registration blocked; spoofable `X-BigBase-User-ID`/`X-Site-ID` headers stripped and re-derived from validated tokens; CORS default-closed; `X-Forwarded-For` never trusted for identity
- Path traversal: storage upload/download/thumbnail/delete use `filepath.Base` + `filepath.Abs` prefix containment; deploy tar extraction enforces `isUnderDir` (no zip slip); `LoadManifestPath` rejects absolute/escaping paths
- Command construction outside cici: all other `exec.Command` uses pass fixed argv, no shell, no string-built commands; `github.mirrorRepository` validates `full_name` against a strict regex, pins clone URL to `https://github.com/…` with `--` separator
- Functions runtime (goja): collection names typed/gated, SQL parameterized, `fetch` allowlist-gated per env
- MCP: bearer auth, per-tool tier + scope gates, site-binding derived from credential not arguments
- Realtime WebSocket: real JWT validation wired at main.go:487-493; `CheckOrigin` fails closed (empty-origin non-browser, explicit allowlist, exact same-origin only)
- Templates: all `template.HTML` usages render static developer-authored strings; `injectMetadata` escapes via JSON-string escaper
- `/api/sql`: admin-gated twice, read-only allowlist, internal-table blocklist, parenthesized org scoping (BUG-129 fix verified at api.go:698-711)
- Secrets component: AES-256-GCM envelope encryption, length-prefixed injective AAD binding scope, random nonces, fail-closed checks, transactional rotation; values never logged; masked previews only; `read_secret_value` permission gate
- Deserialization: yaml.v3 only (no arbitrary type instantiation); no gob/pickle patterns
- CICI run/log reads: org-scoped with cross-tenant rejection

**Frontend + SDKs — zero findings ≥ 8/10**
- Single `dangerouslySetInnerHTML` in the UI (`ui/src/components/StreamLog.tsx:137`) renders `ansiToHtml()` output that HTML-escapes all literal text with colors from a hardcoded table — no injection path
- Tokens memory-only (`useRef` in AuthContext.tsx:53, never written to storage); pending-route persistence validates `startsWith('/')`; no token ever parsed from URL; no postMessage surface anywhere; localStorage limited to theme/accent/tutorial state
- No markdown libraries, no `eval`/`new Function`/`document.write`, no embedded secrets, no user-controlled base URL; SDK `signIn.social` properly `encodeURIComponent`s the redirect; WebSocket URLs same-origin by construction
- Sub-threshold hardening notes (not findings): `auth-next` forwards client `Host` as `x-forwarded-host` (backend-dependent, and H-3 must be fixed regardless); `auth-svelte` middleware has a functional bug (token not set before `getSession`) and accepts-but-ignores `cookieSecret`; pending-route `startsWith('/')` admits protocol-relative `//evil.com` (requires XSS to exploit, which is already game over)

**Dev tooling / CI / Infra**
- No secrets in git history (all-history search for `gho_/ghp_/ghs_/github_pat_/AKIA/sk-/xox*/NRAK-/PRIVATE KEY` — nothing real); `bigbase.db` is NOT tracked (gitignored; all sensitive tables empty — local dev artifact)
- No `pull_request_target` anywhere; no untrusted `github.event.*` interpolation into `run:` blocks; `deploy.yml:63-67` correctly gates on `workflow_run.event == 'push' && head_branch == 'main'` (blocks fork-PR artifact poisoning); all other Actions pinned to 40-char SHAs; secrets never echoed (VPS `.env` written with `chmod 600`, key names only)
- `mcp-http-proxy.mjs/.sh`: stdio-only line-protocol forwarders — no network bind, no listener
- Terraform: no hardcoded cloud credentials, no IAM wildcards, no public database, default at-rest boot-volume encryption
- Shell scripts: consistent `set -euo pipefail`; secrets prompted with `read -s`; env keys guarded with `:?`; no `eval` on untrusted input
- Observability stack: Grafana/Prometheus/Uptime Kuma bind `127.0.0.1` only; no default `admin/admin`

---

## Risk Summary

| Severity | Count | IDs |
|----------|-------|-----|
| CRITICAL | 1 | C-1 |
| HIGH | 3 | H-1, H-2, H-3 |
| MEDIUM | 8 | M-1 … M-8 |
| LOW | 3 | L-1 … L-3 |
| Latent | 3 | webhooks, backup handler, envvars |

The frontend and SDK packages, the secrets component, and CI/CD pinning hygiene are notably strong. The systemic risks are: (1) unsandboxed tenant code execution (C-1), (2) authorization guards written to skip when identity context is absent rather than fail closed (H-1, M-1, M-8, and the latent webhooks component share this pattern).

---

## Remediation Priority

| # | Action | Effort | Notes |
|---|--------|--------|-------|
| 1 | **M-5:** Revoke the `.envrc` GitHub token today | minutes | Independent of code; also remove `.envrc` from gitleaks allowlist |
| 2 | **C-1:** Sandbox CI execution (containers/namespaces) or fixed step vocabulary | large | Design decision required; blacklist cannot be fixed |
| 3 | **H-1:** Fail-closed collections API for contextless callers | small | Mirror `secrets_api.go:337-347` pattern |
| 4 | **H-2:** `Content-Disposition: attachment` + drop `text/html`/`image/svg+xml` from inline set | small | |
| 5 | **H-3:** `PublicURLOrDefault` for magic links | trivial | One-line class, already fixed for OAuth |
| 6 | **M-1:** Repo ownership check in deploy `HandleCreate`/`Trigger` | small | Mirror `cici.verifyRepoOwnership` |
| 7 | **M-2, M-3, M-4:** Token entropy, OTP log gate, reset revocation | small each | |
| 8 | **M-6, M-7, M-8, L-1…L-3, latent items** | scheduled hardening | Fix latent components before wiring them |

---

## Cross-Validation — deepsec methodology (2026-08-14, addendum)

The findings above were re-reviewed against the methodology of [`vercel-labs/deepsec`](https://github.com/vercel-labs/deepsec) (agent-powered vulnerability scanner; source studied via opensrc at `~/.opensrc/repos/github.com/vercel-labs/deepsec/main`). deepsec's pipeline is: wide-net regex matchers (200, noise-tiered) → AI investigation ("think like an attacker; subtle logic flaws") → **adversarial revalidation** (fresh reviewer persona must assign a verdict — true-positive / false-positive / fixed / uncertain / duplicate — and "construct a concrete attack scenario, or it's likely a false positive") → triage (exploitability × impact → P0/P1/P2).

### Methodology gaps in the original review, closed per deepsec's category list

| deepsec matcher/slug | Check performed | Result |
|----------------------|-----------------|--------|
| `zod-passthrough-mass-assignment` (mass assignment) | `handleUpdateMe` (usermgmt.go:19-23) binds only `name`/`avatar_url` pointer fields — no role/org over-posting possible | **CLEAN** |
| `expensive-api-abuse` (LLM/paid API without abuse protection) | `internal/llm` is imported only by `monitoring` (internal client, no HTTP surface of its own) | **CLEAN** |
| `session-cookie-config` | All cookie issuance sites set `HttpOnly` + `SameSite` (Strict for credentials, Lax for OAuth flows) + `Secure` via `cookieSecure()` | **CLEAN** |
| `cache-key-poisoning` | No response-caching layer in the proxy (only the autocert certificate cache) | **CLEAN** |
| `test-header-bypass` | No `x-test-*`/`x-debug`/`x-internal` bypass headers anywhere | **CLEAN** (one instance of the sibling `dev-auth-bypass` pattern is M-3) |

### Adversarial revalidation verdicts (deepsec rubric applied)

| Finding | deepsec slug | Verdict | Adjusted severity | Revalidation reasoning |
|---------|-------------|---------|-------------------|------------------------|
| C-1 cici `sh -c` | `rce` | **true-positive** | CRITICAL (confirmed) | deepsec CRITICAL = RCE. No sandbox, no framework guard; `validateCommand` is a word-boundary regex blacklist, not a mitigation — a concrete bypass (`python3 -c`) was demonstrated in source. |
| H-1 collections IDOR | `cross-tenant-id`, `unverified-lookup` | **true-positive** | **CRITICAL (upgraded from HIGH)** | deepsec CRITICAL = "authentication bypass allowing full access". Any site-deploy-key holder reads **every tenant's records** and mutates/deletes by ID — that is full cross-tenant data access, not a limited IDOR. deepsec's "has auth, wrong auth" taxonomy (core prompt: "Cross-tenant access… Missing resource-level checks") describes this exactly. Note: deepsec's FP rule that "middleware must wrap the handler directly" *strengthens* the finding — the middleware does wrap the handler but establishes an incomplete identity, and every guard fails open on absent context. |
| H-2 storage XSS | `xss`, `dangerous-html` | **true-positive** | HIGH (confirmed) | deepsec HIGH = XSS. Concrete attack: upload HTML → victim navigates to inline-served file on platform origin. No framework auto-escaping applies to a raw `Content-Type` sink. |
| H-3 magic-link Host poisoning | `unsafe-redirect` | **true-positive** | HIGH (confirmed, see note) | deepsec's anchor category (open redirect) is MEDIUM, but the impact here is token theft → account takeover ("auth bypass allowing full access"). Chained severity justifies HIGH; the mitigation exists in-repo (`PublicURLOrDefault`) and is simply not applied — deepsec's `unsafe-redirect` matcher is designed to catch precisely "the codebase has a proper validation function; flag any path that doesn't use it." |
| M-1 cross-tenant deploy | `cross-tenant-id` | **true-positive** | **HIGH (upgraded from MEDIUM-HIGH)** | deepsec HIGH explicitly includes "missing authorization on sensitive operations". Cloning and executing another tenant's private repo is as sensitive as operations get; ownership check exists in sibling code (`cici.verifyRepoOwnership`) and is simply not consulted. |
| M-2 24-bit verify token | `insecure-crypto`, `rate-limit-bypass` | **true-positive** | MEDIUM (confirmed) | Matches deepsec MEDIUM (weak crypto + missing rate limiting). |
| M-3 OTP in logs | `secret-in-log`, `dev-auth-bypass` | **true-positive** | MEDIUM (confirmed) | deepsec has *two* dedicated first-class matchers for this pattern: credentials in logs, and dev-mode branches live in production. Chained with M-8 (cross-tenant log readability) deepsec would likely escalate to HIGH — recommend fixing together. |
| M-4 reset w/o revocation | auth logic bug | **true-positive** | MEDIUM (confirmed) | deepsec MEDIUM = "logic bugs in auth/permission checks". |
| M-5 `.envrc` token | `secrets-plaintext-exposure` | **true-positive** | MEDIUM (confirmed) | deepsec rates hardcoded secrets HIGH *in source code*; this is local-only and untracked, so MEDIUM stands. |
| M-6 mutable PR-Agent tag | `dockerfile-from-mutable-tag` (analog) | **true-positive** | MEDIUM (confirmed) | deepsec has a dedicated matcher for mutable-tag pinning; every other Action in this repo is SHA-pinned. |
| M-7 SSH 0.0.0.0/0 | `tf-public-ingress` | **true-positive** | MEDIUM (confirmed) | deepsec has a dedicated Terraform matcher for public ingress rules. |
| M-8 cross-tenant logs | `cross-tenant-id` | **true-positive** | MEDIUM (confirmed) | Same fail-open class as H-1/M-1 (see systemic note). |
| L-1 invite expiry | auth logic bug | **true-positive** | LOW (confirmed) | — |
| L-2 conditional webhook sig | `webhook-handler` | **true-positive** | LOW (confirmed, borderline MEDIUM) | deepsec treats "webhook endpoints without signature verification" as a first-class category; the conditional (`if secret != ""`) is the weak form. Exploit path is limited (mirror-refetch only), keeping it LOW. |
| L-3 `curl \| bash` | `dockerfile-curl-pipe-unverified` | **true-positive** | LOW (confirmed) | deepsec has a dedicated matcher; the repo's own gitleaks checksum pattern shows the correct fix. |
| Webhooks/backup/envvars (latent) | `webhook-handler`, `ssrf`, `public-endpoint` | **true-positive (latent)** | as stated | Unrouted today; deepsec scans all files regardless of routing, and would flag these — the platform difference is that deepsec cannot know they are unmounted. Fix-before-wiring requirement stands. |

**No false-positive or uncertain verdicts.** No duplicates per deepsec's rules (same class, different locations = separate findings) — but see the systemic note below.

### Systemic finding (deepsec "authorization gaps" lens)

H-1, M-1, M-8, and the latent webhooks component are four instances of one root pattern: **authorization guards written to skip (fail open) when identity context is absent**, plus **ownership checks omitted where sibling code already implements them**. deepsec's core prompt groups these under "Authorization Gaps (has auth, wrong auth)". Remediation should include a lint/review rule: *every* handler that consults `OrgIDFromContext`/`SiteIDFromContext` must fail closed on absence, and every `SELECT … WHERE id = ?` on a tenant-owned table must be co-located with an ownership predicate. This converts four fixes into one enforced invariant.

### Severity delta summary

| Finding | Original | After deepsec cross-validation |
|---------|----------|-------------------------------|
| H-1 | HIGH | **CRITICAL** (auth bypass w/ full cross-tenant data access) |
| M-1 | MEDIUM-HIGH | **HIGH** (missing authorization on sensitive operations) |
| All others | — | Confirmed unchanged |

---

*Report generated 2026-08-14 by the `security-review` skill (5-phase scan, confidence ≥ 8/10 threshold, false-positive exclusions applied). All HIGH+ findings verified directly in source at the listed line numbers. Cross-validation addendum: findings re-reviewed against vercel-labs/deepsec methodology (adversarial revalidation verdicts, severity taxonomy, matcher category coverage); two severity upgrades applied.*
