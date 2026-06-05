# BUG-2026-06-04T152200: Deployed site HTTPS fails — Caddy wildcard TLS

## Problem

- **Actual:** `https://danielvm-git-big-dock-locker-site.bigbase.click` does not open. Browsers and `curl` fail during the TLS handshake (`tlsv1 alert internal error`, no peer certificate). The admin UI shows the deployment as **RUNNING** with the correct public URL. HTTP to the subdomain returns `308` redirect to HTTPS, then HTTPS fails.
- **Expected:** HTTPS serves the static site with a valid Let's Encrypt certificate, same as `https://bigbase.click`.
- **Reproduce:**
  1. On production (`bigbase.click`), create or redeploy a site (e.g. `big-dock-locker-site`, static type).
  2. Wait until status is **RUNNING** and URL is `https://<slug>.bigbase.click`.
  3. Open the URL or run: `curl -vI https://danielvm-git-big-dock-locker-site.bigbase.click`
  4. On the VPS: `journalctl -u caddy -n 50 | grep bigbase.click`

**Related prior work:** BUG-2026-06-04T114000 fixed localhost URLs and proxy host routing; this is a **follow-on ops/TLS issue**, not a recurrence of the localhost URL bug. Deploy and proxy layers behave correctly once TLS succeeds.

## Root Cause Analysis

**Phase 1 — Reproduce (verified):**

| Check | Result |
|-------|--------|
| DNS `danielvm-git-big-dock-locker-site.bigbase.click` | Resolves to `89.116.26.187` |
| `https://bigbase.click/health` | HTTP 200, valid LE cert (`CN=bigbase.click`) |
| `https://danielvm-git-big-dock-locker-site.bigbase.click` | TLS handshake failure |
| `http://…` | 308 → HTTPS (Caddy routing works on port 80) |
| BigBase loopback with `Host` header | HTTP 200 (app + proxy host registry OK) |
| `journalctl -u bigbase` | `serving static site` on assigned port with correct public URL |

**Phase 2 — Isolate:** Failure first appears at **edge TLS termination (Caddy)**, not in deploy build, static file server, or Go reverse-proxy host matching.

**Phase 3 — Hypotheses (ranked):**

1. **(Confirmed)** Caddy site block `*.bigbase.click` triggers ACME for a **wildcard** certificate, which requires **DNS-01**. Only HTTP-01 / TLS-ALPN-01 are configured → perpetual obtain failure for `*.bigbase.click`.
2. **(Partial)** Per-hostname HTTP-01 certs can still be issued for some slugs (e.g. another site returned HTTPS 200 after first access), but new slugs may never get a cert while the wildcard obtain keeps failing and the handshake aborts.
3. **(Rejected)** Deploy build failure — deployment is `running`, static server listening.
4. **(Rejected)** DNS wildcard A record — resolution is correct.

**Phase 4 — Verify (Caddy logs):**

```text
could not get certificate from issuer","identifier":"*.bigbase.click"
error":"[*.bigbase.click] solving challenges: *.bigbase.click: no solvers available
  for remaining challenges (configured=[http-01 tls-alpn-01] offered=[dns-01] remaining=[dns-01])"
```

Certificate store on VPS contains `bigbase.click` and one per-host cert for a different slug, but **not** `danielvm-git-big-dock-locker-site.bigbase.click`.

**Risk level:** Medium — affects all new site subdomains until Caddy TLS strategy is fixed; application code for deploy/proxy is largely correct.

## TDD Fix Plan

1. **RED:** Integration test (or script checked in CI) that documents required Caddy TLS behavior: site subdomains must not depend on a wildcard cert obtained via DNS-01 only. Assert setup template contains `on_demand` (or `tls.dns`) and does **not** rely on bare `*.domain` implicit wildcard cert without a DNS solver.  
   **GREEN:** Update VPS setup template so the sites block uses **on-demand TLS** with an `ask` URL served by BigBase (allow only registered deployment hosts), **or** configure DNS-01 with the DNS provider API token.  
   **verify:** `grep -E 'on_demand|tls\.dns' scripts/setup-vps.sh`

2. **RED:** Proxy/API test: `GET /api/internal/caddy-allow?domain=<slug>.bigbase.click` returns 200 only when that host is registered for a running deployment, 403 otherwise.  
   **GREEN:** Implement ask handler; register host on deploy finalize (existing host registry).  
   **verify:** `go test ./components/proxy/... -run TestCaddyAllow -count=1`

3. **RED:** Documented smoke step in DEPLOY.md: after Caddy reload, `curl -sfI https://<test-slug>.bigbase.click` must return 2xx/3xx within 120s of first request.  
   **GREEN:** Add `systemctl reload caddy` after Caddyfile write in setup script; add ops note in DEPLOY.md.  
   **verify:** `bash -n scripts/setup-vps.sh`

4. **REFACTOR:** Remove or stop retrying doomed `*.bigbase.click` wildcard cert orders if using per-host on-demand only; set ACME contact email in global Caddy options.

**Immediate ops workaround (no code):** On VPS, temporarily obtain a per-host cert by fixing Caddy config and reloading, or use DNS-01 for `*.bigbase.click` if the DNS provider supports it.

## Acceptance Criteria

- [x] `https://danielvm-git-big-dock-locker-site.bigbase.click` returns 200 (or expected static content) with a valid public CA certificate
- [x] New site slugs get HTTPS automatically on first visit without manual Caddy intervention
- [x] Caddy logs show no repeating `*.bigbase.click` DNS-01 solver errors (or DNS-01 succeeds if that path is chosen)
- [x] `go test ./...` passes after proxy/setup changes
- [x] Existing apex `https://bigbase.click` still works

## Resolution

**Fixed:** 2026-06-04  
**Root cause:** Caddy `*.bigbase.click` site block requested a wildcard cert (DNS-01 only); per-host certs were not issued reliably. Deploy did not restore proxy routes or static servers after `bigbase` restart.  
**Fix:** On-demand TLS in `scripts/setup-vps.sh` with `GET /api/internal/caddy-allow`; restore hosts on deploy start; resume static deployments from build artifacts (no DB writes in resume path).  
**Evidence:** `go test ./components/proxy/... ./components/deploy/... -count=1` PASS; production `curl -sfI https://danielvm-git-big-dock-locker-site.bigbase.click/` → 200; Caddy log `certificate obtained successfully` for site host.
