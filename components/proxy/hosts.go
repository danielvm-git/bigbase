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
	port   int
	siteID string
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
	if existing, ok := p.deployHosts[host]; ok && existing.port != port {
		return fmt.Errorf("host %q already registered", host)
	}
	p.deployHosts[host] = hostInfo{port: port, siteID: siteID}
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

func (p *Proxy) getDeploymentHostInfo(host string) (hostInfo, bool) {
	host = normalizeHost(host)
	p.deployHostsMu.RLock()
	info, ok := p.deployHosts[host]
	p.deployHostsMu.RUnlock()
	return info, ok
}

// handleCaddyAllow implements Caddy on_demand_tls ask (GET ?domain=host).
// Returns 200 only when the host is registered for a running deployment.
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
	if !p.isDeploymentHostRegistered(domain) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusOK)
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
		
		info, ok := p.getDeploymentHostInfo(host)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", info.port))
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
			p.requestLogger.RecordRequestLog(info.siteID, r.Method, r.URL.Path, rw.status, time.Since(start))
		}
	})
}
