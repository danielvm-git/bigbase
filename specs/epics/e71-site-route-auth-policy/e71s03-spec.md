### Story e71s03: Passthrough auth injection — X-BigBase-User-ID header — Implementation Steps

**type:** feat
**risk:** P1
**context:** domain
**Context**: Protected routes often reverse-proxy/forward requests to backends (e.g. serverless functions or passthrough paths like `/mcp`). When routing authenticated requests, the proxy should inject headers containing the validated identity, so the backend doesn't need to re-verify the JWT. This story injects `X-BigBase-User-ID` and `X-BigBase-Site-ID` headers into HTTP requests forwarded to the deployed site's port.

## Steps

1. In `components/proxy/hosts.go` `deploymentHostMiddleware`, intercept the reverse proxy initialization. Wrap `proxy.Director` to inject the `X-BigBase-Site-ID` header with the current site ID. Tag Go standard library networking packages as `[OK]`. → verify: `go test -v -run TestProxySiteIDHeader ./components/proxy/...`
2. Update the `deploymentHostMiddleware` proxy director to also inject the `X-BigBase-User-ID` header if the incoming request was successfully authenticated via a valid JWT token. If the request was public/unauthenticated, do not inject the `X-BigBase-User-ID` header (or explicitly clear it to prevent spoofing). → verify: `go test -v -run TestProxyUserIDHeader ./components/proxy/...`
3. Write test scenarios in `components/proxy/hosts_test.go` checking that a mock backend receives both headers correctly for authenticated paths, and does not receive `X-BigBase-User-ID` for public paths. → verify: `go test -v -run TestProxyHeaderInjection ./components/proxy/...`

## Verification Script (Step-by-Step)

1. Run `go test ./components/proxy/...` to verify proxy header injection logic.
2. Run `go build -o bigbase .` to verify compilation.

## Out of scope

- MCP configuration tools (e71s04).

## Risks

- Header spoofing: a malicious client sends `X-BigBase-User-ID` header on a public path. Mitigated by explicitly removing or overriding `X-BigBase-User-ID` header in the proxy director for all incoming client requests.
