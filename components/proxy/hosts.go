package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type hostInfo struct {
	Port   int
	SiteID string
}

// RegisterDeploymentHost maps a public hostname to a local deployment port.
func (p *Proxy) RegisterDeploymentHost(host string, port int, siteID string) error {
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
	p.deployHosts[host] = hostInfo{Port: port, SiteID: siteID}
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
		
		info, ok := p.GetDeploymentHostInfo(host)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		// API calls to deployment hosts must reach BigBase, not the static
		// file server. Forward /api/* paths to the main handler.
		if strings.HasPrefix(r.URL.Path, "/api/") {
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

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		proxy.ServeHTTP(rw, r)

		if p.requestLogger != nil {
			p.requestLogger.RecordRequestLog(info.SiteID, r.Method, r.URL.Path, rw.status, time.Since(start))
		}
	})
}
