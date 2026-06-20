package proxy

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type hostInfo struct {
	Port             int
	SiteID           string
	PassthroughPaths []string
	Metadata         map[string]string // injected into __BIGBASE_METADATA__
}

// RegisterDeploymentHost maps a public hostname to a local deployment port.
// passthroughPaths are URL path prefixes that bypass BigBase's /api/* intercept
// and are forwarded directly to the deployed app.
func (p *Proxy) RegisterDeploymentHost(host string, port int, siteID string, passthroughPaths []string, metadata map[string]string) error {
	host = normalizeHost(host)
	if host == "" || port < 1 || port > 65535 {
		return fmt.Errorf("invalid deployment host or port")
	}
	if loopbackHosts[host] || net.ParseIP(host) != nil {
		return fmt.Errorf("cannot register loopback address %q as deployment host", host)
	}
	p.deployHostsMu.Lock()
	defer p.deployHostsMu.Unlock()
	if p.deployHosts == nil {
		p.deployHosts = make(map[string]hostInfo)
	}
	// Allow replacing an existing registration — subsequent deployments for the
	// same host update the port in place, enabling zero-downtime redeployment.
	p.deployHosts[host] = hostInfo{Port: port, SiteID: siteID, PassthroughPaths: passthroughPaths, Metadata: metadata}
	return nil
}

// UnregisterDeploymentHost removes a deployment hostname route.
func (p *Proxy) UnregisterDeploymentHost(host string) {
	host = normalizeHost(host)
	p.deployHostsMu.Lock()
	delete(p.deployHosts, host)
	p.deployHostsMu.Unlock()
}

func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	return host
}

func (p *Proxy) isDeploymentHostRegistered(host string) bool {
	host = normalizeHost(host)
	p.deployHostsMu.RLock()
	_, ok := p.deployHosts[host]
	p.deployHostsMu.RUnlock()
	return ok
}

// GetDeploymentHostInfo returns the current port and site ID for a deployment host.
func (p *Proxy) GetDeploymentHostInfo(host string) (hostInfo, bool) {
	host = normalizeHost(host)
	p.deployHostsMu.RLock()
	info, ok := p.deployHosts[host]
	p.deployHostsMu.RUnlock()
	return info, ok
}

// handleCaddyAllow implements Caddy on_demand_tls ask (GET ?domain=host).
// Returns 200 when the host is a loopback host, a registered deployment, or
// explicitly allowed (e.g. mcp.bigbase.click for public MCP access).
func (p *Proxy) handleCaddyAllow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	domain := normalizeHost(r.URL.Query().Get("domain"))
	if domain == "" {
		http.Error(w, "missing domain", http.StatusBadRequest)
		return
	}
	// Allow loopback hosts (bigbase.click, localhost, etc.) and explicit
	// allowlist entries (mcp.bigbase.click) without a running deployment.
	if allowedHosts[domain] {
		w.WriteHeader(http.StatusOK)
		return
	}
	if p.isDeploymentHostRegistered(domain) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "forbidden", http.StatusForbidden)
}

// allowedHosts are hosts that can get TLS certs via on_demand_tls even without
// being a running deployment host. Used for MCP server, admin extensions, etc.
var allowedHosts = map[string]bool{
	"mcp.bigbase.click": true,
}

// defaultServiceHosts maps hostnames to backend ports for internal services
// that run alongside the proxy. These are seeded into the Proxy instance on
// Start() and can be extended at runtime via AddServiceHost.
var defaultServiceHosts = map[string]int{
	"mcp.bigbase.click": 3900,
}

// serviceExcludePaths are URL paths that the proxy handles directly for
// service hosts, rather than proxying to the backend.
var serviceExcludePaths = map[string]bool{
	"/.well-known/mcp.json": true,
}

// AddServiceHost registers a hostname → backend port mapping for a permanent
// internal service (not a deployment). The proxy will route requests with this
// Host header to the backend, except for paths in serviceExcludePaths.
func (p *Proxy) AddServiceHost(host string, port int) {
	host = normalizeHost(host)
	if p.serviceHosts == nil {
		p.serviceHosts = make(map[string]int)
	}
	p.serviceHosts[host] = port
}

// RemoveServiceHost unregisters a service host.
func (p *Proxy) RemoveServiceHost(host string) {
	host = normalizeHost(host)
	if p.serviceHosts != nil {
		delete(p.serviceHosts, host)
	}
}

// loopbackHosts are local addresses that should never be routed as deployment hosts.
var loopbackHosts = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"::1":       true,
}

func (p *Proxy) deploymentHostMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := normalizeHost(r.Host)
		if loopbackHosts[host] {
			next.ServeHTTP(w, r)
			return
		}
		if net.ParseIP(host) != nil {
			next.ServeHTTP(w, r)
			return
		}
		
		// Service hosts (MCP, admin, etc.) — proxy to backend, but let the
		// proxy itself serve discovery / well-known endpoints.
		if backendPort, ok := p.serviceHosts[host]; ok {
			if serviceExcludePaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", backendPort))
			if err != nil {
				http.Error(w, "bad gateway", http.StatusBadGateway)
				return
			}
			rp := httputil.NewSingleHostReverseProxy(target)
			rp.ErrorHandler = func(rw http.ResponseWriter, _ *http.Request, _ error) {
				http.Error(rw, "service unavailable", http.StatusBadGateway)
			}
			rp.ServeHTTP(w, r)
			return
		}

		info, ok := p.GetDeploymentHostInfo(host)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		// API calls to deployment hosts that match BigBase's own API are
		// handled locally. Passthrough paths (configured per-deployment) skip
		// this intercept and are forwarded to the deployed app.
		passthrough := false
		for _, pp := range info.PassthroughPaths {
			pp = strings.TrimSuffix(pp, "/*")
			if strings.HasPrefix(r.URL.Path, pp) {
				passthrough = true
				break
			}
		}
		if !passthrough && strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", info.Port))
		if err != nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(rw http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(rw, "deployment unavailable", http.StatusBadGateway)
		}
		// Inject __BIGBASE_METADATA__ script into HTML responses.
		proxy.ModifyResponse = func(resp *http.Response) error {
			p.injectMetadata(resp, info.Metadata)
			return nil
		}

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		proxy.ServeHTTP(rw, r)

		if p.requestLogger != nil {
			p.requestLogger.RecordRequestLog(info.SiteID, r.Method, r.URL.Path, rw.status, time.Since(start))
		}
	})
}

// injectMetadata adds a __BIGBASE_METADATA__ <script> tag to HTML responses.
// Supports gzip-compressed responses. Non-HTML and empty metadata are no-ops.
func (p *Proxy) injectMetadata(resp *http.Response, metadata map[string]string) {
	if len(metadata) == 0 {
		return
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		return
	}

	// Build the injection script.
	var buf bytes.Buffer
	buf.WriteString("<script>window.__BIGBASE_METADATA__ = {")
	first := true
	for k, v := range metadata {
		if !first {
			buf.WriteString(",")
		}
		first = false
		fmt.Fprintf(&buf, `"%s":"%s"`, k, escapeJSON(v))
	}
	buf.WriteString("};</script>")
	script := buf.Bytes()

	// Read the body, optionally decompress gzip.
	var bodyBytes []byte
	var err error
	if resp.Header.Get("Content-Encoding") == "gzip" {
		var gr *gzip.Reader
		gr, err = gzip.NewReader(resp.Body)
		if err != nil {
			return
		}
		bodyBytes, err = io.ReadAll(gr)
		_ = gr.Close()
	} else {
		bodyBytes, err = io.ReadAll(resp.Body)
	}
	if err != nil || len(bodyBytes) == 0 {
		resp.Body = io.NopCloser(strings.NewReader(""))
		// Re-create a nil body to avoid data loss on error.
		if len(bodyBytes) > 0 {
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
		return
	}
	_ = resp.Body.Close()

	// Inject after </head> or before </body>, preferring </head>.
	var injected []byte
	if idx := bytes.LastIndex(bodyBytes, []byte("</head>")); idx >= 0 {
		injected = make([]byte, 0, len(bodyBytes)+len(script))
		injected = append(injected, bodyBytes[:idx+len("</head>")]...)
		injected = append(injected, script...)
		injected = append(injected, bodyBytes[idx+len("</head>"):]...)
	} else if idx := bytes.LastIndex(bodyBytes, []byte("</body>")); idx >= 0 {
		injected = make([]byte, 0, len(bodyBytes)+len(script))
		injected = append(injected, bodyBytes[:idx]...)
		injected = append(injected, script...)
		injected = append(injected, bodyBytes[idx:]...)
	} else {
		// No recognizable insertion point — prepend.
		injected = make([]byte, 0, len(bodyBytes)+len(script))
		injected = append(injected, script...)
		injected = append(injected, bodyBytes...)
	}

	// Re-compress if original was gzipped.
	if resp.Header.Get("Content-Encoding") == "gzip" {
		var gzBuf bytes.Buffer
		gw := gzip.NewWriter(&gzBuf)
		if _, err := gw.Write(injected); err != nil {
			_ = gw.Close()
			return
		}
		_ = gw.Close()
		injected = gzBuf.Bytes()
	}

	resp.Body = io.NopCloser(bytes.NewReader(injected))
	resp.ContentLength = int64(len(injected))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(injected)))
}

// escapeJSON escapes a string for safe inclusion in a JSON value.
func escapeJSON(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
