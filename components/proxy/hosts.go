package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// RegisterDeploymentHost maps a public hostname to a local deployment port.
func (p *Proxy) RegisterDeploymentHost(host string, port int) error {
	host = normalizeHost(host)
	if host == "" || port < 1 || port > 65535 {
		return fmt.Errorf("invalid deployment host or port")
	}
	p.deployHostsMu.Lock()
	defer p.deployHostsMu.Unlock()
	if p.deployHosts == nil {
		p.deployHosts = make(map[string]int)
	}
	if existing, ok := p.deployHosts[host]; ok && existing != port {
		return fmt.Errorf("host %q already registered", host)
	}
	p.deployHosts[host] = port
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

func (p *Proxy) deploymentHostMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := normalizeHost(r.Host)
		p.deployHostsMu.RLock()
		port, ok := p.deployHosts[host]
		p.deployHostsMu.RUnlock()
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
		if err != nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(rw http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(rw, "deployment unavailable", http.StatusBadGateway)
		}
		proxy.ServeHTTP(w, r)
	})
}
