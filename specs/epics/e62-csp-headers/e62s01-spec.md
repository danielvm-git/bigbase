# e62s01: CSP Header Generation Per-Site
## Story ID: e62s01 | Epic: e62 | BCPs: 2 | Status: planned

### Story e62s01: CSP Header Generation Per-Site — Implementation Steps

**type:** feat
**context:** domain
**Context**: This story implements per-site configurable Content-Security-Policy (CSP) headers with cryptographic nonce support. Currently, BigBase proxy applies a global CSP which can break deployed sites. By configuring CSP at the site level, developers can use frameworks like Next.js or SvelteKit which require `unsafe-inline` or specific domains, while allowing the proxy to inject a `{nonce}` dynamically per-request.

## Zoom-Out Check
- **Purpose**: Allow per-site CSP overrides via the Proxy, with dynamic nonce generation to secure injected metadata scripts.
- **Callers**: Proxy HTTP router (`securityHeadersMiddleware`), Deploy supervisor (`RegisterDeploymentHost`).
- **Contracts**: BigBase internal API routes (`/api/*`) MUST retain their strict `default-src 'self'` policy. The injected `__BIGBASE_METADATA__` script tag MUST use the generated nonce so the browser does not block it.

## Steps

1. Update `Site` model in `components/sites/sites.go` to include `CSPTemplate string json:"csp_template"`. In `Start()`, add `ALTER TABLE sites ADD COLUMN csp_template TEXT NOT NULL DEFAULT ''` (ignoring duplicate column errors) and update SQL queries in CRUD methods. → verify: `go test ./components/sites/...`
2. In `components/proxy/hosts.go`, add `CSPTemplate string` to `hostInfo` and update `RegisterDeploymentHost` signature to accept `cspTemplate`. Update all callers in `deploy` and `proxy` test files to pass empty strings. → verify: `go build ./... && go test ./components/proxy/... ./components/deploy/...`
3. Modify `securityHeadersMiddleware` in `components/proxy/securityheaders.go` to lookup the deployment host using `p.GetDeploymentHostInfo(r.Host)`. If found and `CSPTemplate` is not empty, generate a 16-byte base64 nonce. Replace `{nonce}` with this nonce in the template and set the `Content-Security-Policy` header. Attach the nonce to `r.Context()`. Keep the existing strict/permissive logic for non-deployment routes. → verify: `go test ./components/proxy/... -run TestSecurityHeaders`
4. Update `injectMetadata` and `deploymentHostMiddleware` in `components/proxy/hosts.go` to retrieve the nonce from `r.Context()`. If present, add `nonce="<nonce>"` to the `<script>` tag injected for `__BIGBASE_METADATA__`. → verify: `go test ./components/proxy/... -run TestMetadataInjection`
5. In `components/deploy/deploy.go`, when registering the host in `DeploySite` and `supervisor.go`, retrieve the site's `CSPTemplate` from the database and pass it to `RegisterDeploymentHost`. → verify: `go test ./components/deploy/...`

## Verification Script (Step-by-Step)

1. Boot BigBase.
2. Use the CLI or Admin UI to set a `csp_template` for a test site: `default-src 'self'; script-src 'self' 'nonce-{nonce}'`
3. Deploy a simple HTML file to the site.
4. Perform `curl -v http://<test-site-domain>`
5. Observe the `Content-Security-Policy` header contains a dynamic nonce (e.g. `'nonce-aBcD123...'`).
6. Observe the injected `__BIGBASE_METADATA__` `<script>` tag includes `nonce="aBcD123..."`.
7. Refresh and observe the nonce changes.

## Out of scope
- Admin UI policy builder (covered in e62s02).
- Nonce generation for internal BigBase UI (only covers deployment hosts).

## Risks
- Browser cache policies might cache HTML with a used nonce if `Cache-Control` is not configured properly by the deployed site.
- Deployments might break if they write invalid CSP syntax.
