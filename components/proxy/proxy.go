package proxy

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/danielvm/bigbase/kernel"
)

// requestIDKey is the context key for request-scoped request IDs.
type requestIDKey struct{}

// RequestIDFromContext extracts the request ID from a context, returning empty string if absent.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// generateRequestID produces a 32-hex-char random string (16 bytes) for request tracing.
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

var templateFuncs = template.FuncMap{
	"join": strings.Join,
}

const version = "0.1.0"

type RequestLogger interface {
	RecordRequestLog(siteID, method, path string, status int, duration time.Duration)
}

type DBer = kernel.QueryExecDBer

type Options struct {
	Port               string
	HTTPSPort          string
	Kernel             *kernel.Kernel
	Logger             kernel.Logger
	RequestLogger      RequestLogger
	CORSAllowedOrigins []string
	CertDir            string
	ACMEEmail          string
	HealthToken        string
	DB                 DBer
	ValidateToken      func(token string) (userID int64, role string, err error)
	ValidateSiteKey    func(siteID string, token string) error
}

type Proxy struct {
	port               string
	httpsPort          string
	kernel             *kernel.Kernel
	logger             kernel.Logger
	requestLogger      RequestLogger
	server             *http.Server
	httpsServer        *http.Server
	mux                *http.ServeMux
	corsAllowedOrigins []string
	acmeManager        *autocert.Manager
	certDir            string
	acmeEmail          string
	healthToken        string
	db                 DBer
	validateToken      func(token string) (userID int64, role string, err error)
	validateSiteKey    func(siteID string, token string) error

	starsMu        sync.Mutex
	starsVal       string
	starsTime      time.Time
	deployHostsMu  sync.RWMutex
	deployHosts    map[string]hostInfo
	serviceHosts   map[string]int
	authPoliciesMu sync.RWMutex
	authPolicies   map[string]*AuthPolicy
	cspPoliciesMu  sync.RWMutex
	cspPolicies    map[string]string
}

var _ kernel.Component = (*Proxy)(nil)

func (p *Proxy) GitHubStars() string {
	p.starsMu.Lock()
	defer p.starsMu.Unlock()
	if p.starsVal != "" && time.Since(p.starsTime) < 5*time.Minute {
		return p.starsVal
	}
	stars, err := p.fetchStars()
	if err != nil {
		if p.starsVal == "" {
			return ""
		}
		return p.starsVal
	}
	p.starsVal = stars
	p.starsTime = time.Now()
	return p.starsVal
}

func (p *Proxy) fetchStars() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/danielvm-git/bigbase")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var repo struct {
		StargazersCount int `json:"stargazers_count"`
	}
	if err := json.Unmarshal(body, &repo); err != nil {
		return "", err
	}
	if repo.StargazersCount == 0 {
		return "", fmt.Errorf("no stars")
	}
	return fmt.Sprintf("%d", repo.StargazersCount), nil
}

func New(opts Options) *Proxy {
	return &Proxy{
		port:               opts.Port,
		httpsPort:          opts.HTTPSPort,
		kernel:             opts.Kernel,
		logger:             opts.Logger,
		requestLogger:      opts.RequestLogger,
		mux:                http.NewServeMux(),
		certDir:            opts.CertDir,
		acmeEmail:          opts.ACMEEmail,
		corsAllowedOrigins: opts.CORSAllowedOrigins,
		healthToken:        opts.HealthToken,
		db:                 opts.DB,
		validateToken:      opts.ValidateToken,
		validateSiteKey:    opts.ValidateSiteKey,
		authPolicies:       make(map[string]*AuthPolicy),
		cspPolicies:        make(map[string]string),
	}
}

// Port returns the HTTP listen port (resolved after Start if

func (p *Proxy) SetRequestLogger(rl RequestLogger) {
	p.requestLogger = rl
}

func (p *Proxy) SetDB(db DBer) {
	p.db = db
}

func (p *Proxy) SetValidators(validateToken func(token string) (int64, string, error), validateSiteKey func(siteID string, token string) error) {
	p.validateToken = validateToken
	p.validateSiteKey = validateSiteKey
}

// Port returns the HTTP listen port (resolved after Start if port was "0").
func (p *Proxy) Port() string {
	if p.server != nil {
		return portFromAddr(p.server.Addr)
	}
	return p.port
}

// HTTPSAddr returns the HTTPS listen address (host:port), empty if not configured.
func (p *Proxy) HTTPSAddr() string {
	if p.httpsServer != nil {
		return p.httpsServer.Addr
	}
	return ""
}

func (p *Proxy) Name() string                  { return "proxy" }
func (p *Proxy) Version() string               { return version }
func (p *Proxy) Dependencies() []string        { return nil }

func (p *Proxy) Init(ctx *kernel.Context, config json.RawMessage) error {
	if p.port == "" {
		p.port = "8080"
	}
	portNum, err := strconv.Atoi(p.port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("invalid port %q: must be 1-65535", p.port)
	}
	return nil
}

func (p *Proxy) Handler() http.Handler {
	h := p.corsMiddleware(p.httpsRedirectMiddleware(p.securityHeadersMiddleware(p.loggingMiddleware(p.requestIDMiddleware(p.gitPathBlockMiddleware(p.deploymentHostMiddleware(p.mux)))))))
	return h
}

func (p *Proxy) Start(ctx *kernel.Context) error {
	p.mux.HandleFunc("/", p.handleHome)
	p.mux.HandleFunc("/docs", p.handleDocs)
	p.mux.HandleFunc("/health", p.handleHealth)
	p.mux.HandleFunc("/api/version", p.handleVersion)
	p.mux.HandleFunc("/api/internal/caddy-allow", p.handleCaddyAllow)
	p.mux.HandleFunc("/.well-known/mcp.json", p.handleMCPDiscovery)

	// Seed service hosts from package-level defaults (don't overwrite
	// entries already added via AddServiceHost before Start).
	if p.serviceHosts == nil {
		p.serviceHosts = make(map[string]int)
	}
	for host, port := range defaultServiceHosts {
		if _, exists := p.serviceHosts[host]; !exists {
			p.serviceHosts[host] = port
		}
	}

	p.server = &http.Server{
		Addr:    ":" + p.port,
		Handler: p.Handler(),
	}

	go func() {
		p.logger.Info("proxy listening", "port", p.port)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			p.logger.Error("proxy server error", "error", err)
		}
	}()

	// Start HTTPS server with autocert if HTTPS port is configured.
	if p.httpsPort != "" {
		certDir := p.certDir
		if certDir == "" {
			certDir = "data/certs"
		}
		p.acmeManager = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: p.acmeHostPolicy,
			Cache:      autocert.DirCache(certDir),
			Email:      p.acmeEmail,
		}

		p.httpsServer = &http.Server{
			Addr:      ":" + p.httpsPort,
			Handler:   p.acmeManager.HTTPHandler(p.Handler()),
			TLSConfig: p.acmeManager.TLSConfig(),
		}

		go func() {
			p.logger.Info("proxy listening for HTTPS", "port", p.httpsPort, "cert_dir", certDir)
			if err := p.httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				p.logger.Error("https server error", "error", err)
			}
		}()
	}

	return nil
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (p *Proxy) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = generateRequestID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey{}, rid)
		w.Header().Set("X-Request-ID", rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// gitPathBlockMiddleware returns 404 for any request path containing "/.git" to prevent
// exposure of repository metadata.
func (p *Proxy) gitPathBlockMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lowerPath := strings.ToLower(r.URL.Path)
		if strings.Contains(lowerPath, "/.git") || strings.Contains(lowerPath, "\\.git") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (p *Proxy) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rid := RequestIDFromContext(r.Context())
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		args := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration", time.Since(start).String(),
		}
		if rid != "" {
			args = append(args, "request_id", rid)
		}
		p.logger.Info("HTTP request", args...)
	})
}

func (p *Proxy) Handle(pattern string, handler http.HandlerFunc) {
	if p.mux != nil {
		p.mux.HandleFunc(pattern, handler)
	}
}

func (p *Proxy) Stop(ctx *kernel.Context) error {
	if p.httpsServer != nil {
		_ = p.httpsServer.Shutdown(context.Background())
	}
	if p.server != nil {
		return p.server.Shutdown(context.Background())
	}
	return nil
}

// ACME returns the autocert manager, or nil if HTTPS is not configured.
func (p *Proxy) ACME() *autocert.Manager {
	return p.acmeManager
}

// CertInfo returns the expiry of the cached Let's Encrypt certificate for host.
// ok is false when HTTPS is disabled, no cert is cached yet (still provisioning),
// or the cache entry can't be parsed. It reads the autocert cache directly and
// does not trigger provisioning.
func (p *Proxy) CertInfo(ctx context.Context, host string) (time.Time, bool) {
	if p.acmeManager == nil || p.acmeManager.Cache == nil {
		return time.Time{}, false
	}
	host = normalizeHost(host)
	data, err := p.acmeManager.Cache.Get(ctx, host)
	if err != nil {
		return time.Time{}, false
	}
	// Cache entries are PEM bundles (private key followed by the cert chain).
	// The first CERTIFICATE block is the leaf — its NotAfter is the expiry.
	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		return cert.NotAfter, true
	}
	return time.Time{}, false
}

// acmeHostPolicy limits certificate provisioning to registered deployment hosts
// and explicitly allowed hosts. Loopback addresses and IPs are denied.
func (p *Proxy) acmeHostPolicy(_ context.Context, host string) error {
	host = normalizeHost(host)
	if host == "" {
		return fmt.Errorf("empty host")
	}
	// Loopback hosts don't need Let's Encrypt certs.
	if loopbackHosts[host] || net.ParseIP(host) != nil {
		return fmt.Errorf("acme: skipping loopback host %q", host)
	}
	// Allow explicitly whitelisted service hosts (e.g. mcp.bigbase.click).
	if allowedHosts[host] {
		return nil
	}
	// Allow registered deployment hosts (custom domains registered via RegisterDeploymentHost).
	if p.isDeploymentHostRegistered(host) {
		return nil
	}
	return fmt.Errorf("acme: host %q not allowed for certificate provisioning", host)
}

// httpsRedirectMiddleware redirects HTTP requests for registered deployment hosts
// to HTTPS when the HTTPS server is configured. Loopback hosts are not redirected.
func (p *Proxy) httpsRedirectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := normalizeHost(r.Host)
		// Only redirect custom domains when HTTPS is configured.
		if p.httpsPort != "" && host != "" && !loopbackHosts[host] && net.ParseIP(host) == nil {
			if p.isDeploymentHostRegistered(host) {
				target := "https://" + host + r.URL.RequestURI()
				http.Redirect(w, r, target, http.StatusMovedPermanently)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// portFromAddr extracts the port portion from an address string like ":8080".
func portFromAddr(addr string) string {
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		return addr[idx+1:]
	}
	return addr
}

func (p *Proxy) handleDocs(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("docs").Funcs(templateFuncs).Parse(docsTemplate))

	stars := p.GitHubStars()
	starsDisplay := "GitHub"
	if stars != "" {
		starsDisplay = "★ " + stars
	}

	data := struct {
		Version     string
		GitHubStars string
		Port        string
	}{
		Version:     kernel.Version,
		GitHubStars: starsDisplay,
		Port:        p.port,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		p.logger.Error("template error", "error", err)
	}
}

func (p *Proxy) handleHome(w http.ResponseWriter, r *http.Request) {
	components := p.kernel.ListComponents()

	tmpl := template.Must(template.New("home").Funcs(templateFuncs).Parse(homeTemplate))

	stars := p.GitHubStars()
	starsDisplay := "GitHub"
	if stars != "" {
		starsDisplay = "★ " + stars
	}

	data := struct {
		Version     string
		Components  []kernel.ComponentStatus
		GitHubStars string
		Port        string
		ThemeScript template.HTML
	}{
		Version:     kernel.Version,
		Components:  components,
		GitHubStars: starsDisplay,
		Port:        p.port,
		ThemeScript: landingThemeScript(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		p.logger.Error("template error", "error", err)
	}
}

func (p *Proxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	// When healthToken is configured, require Authorization: Bearer <token>.
	if p.healthToken != "" {
		auth := r.Header.Get("Authorization")
		if len(auth) < 7 || !strings.EqualFold(auth[:7], "bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprintf(w, `{"error":"unauthorized"}`+"\n")
			return
		}
		tokenHash := sha256.Sum256([]byte(p.healthToken))
		authHash := sha256.Sum256([]byte(auth[7:]))
		if subtle.ConstantTimeCompare(tokenHash[:], authHash[:]) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprintf(w, `{"error":"unauthorized"}`+"\n")
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	statuses := p.kernel.ListComponents()
	total := len(statuses)
	running := 0
	compMap := make([]map[string]any, 0, total)
	for _, s := range statuses {
		if s.Running {
			running++
		}
		deps := s.Dependencies
		if deps == nil {
			deps = []string{}
		}
		compMap = append(compMap, map[string]any{
			"name":         s.Name,
			"version":      s.Version,
			"running":      s.Running,
			"dependencies": deps,
		})
	}
	status := "ok"
	if total > 0 && running < total {
		status = "degraded"
	}
	resp := map[string]any{
		"status":               status,
		"components":           total,
		"running":              running,
		"component_status_map": compMap,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (p *Proxy) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"version":"%s"}`, kernel.Version)
}

func (p *Proxy) handleMCPDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"mcpServers":{"bigbase":{"url":"https://mcp.bigbase.click/mcp"}}}`+"\n")
}

var homeTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>BigBase — Open-Source BaaS</title>
{{.ThemeScript}}
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
<style>
:root {
  --neutral-0: #fff; --neutral-25: #fafafb; --neutral-40: #f4f4f7;
  --neutral-50: #ededf0; --neutral-100: #e4e4e7; --neutral-200: #d8d8db;
  --neutral-300: #adadb0; --neutral-400: #4b4b50; --neutral-500: #818186;
  --neutral-600: #6c6c71; --neutral-700: #56565c; --neutral-800: #2d2d31;
  --neutral-900: #19191c;

  --brand-500: #4f46e5; --brand-600: #4338ca; --brand-700: #3730a3; --brand-link: #3730a3;
  --success: #22c55e; --success-fg: #044e3a; --error: #ef4444; --error-fg: #781414; --warning: #f59e0b;

  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  --font-mono: 'Fira Code', 'SF Mono', Monaco, Consolas, monospace;

  --bg: var(--neutral-25); --surface: var(--neutral-0);
  --fg: var(--neutral-800); --fg-secondary: #4b4b50; --fg-tertiary: var(--neutral-400); --fg-on-accent: #fff;
  --border: var(--neutral-100); --border-strong: var(--neutral-200);

  --space-2: 2px; --space-4: 4px; --space-6: 6px; --space-8: 8px;
  --space-10: 10px; --space-12: 12px; --space-16: 16px; --space-20: 20px;
  --space-24: 24px; --space-32: 32px; --space-40: 40px; --space-48: 48px;
  --space-64: 64px; --space-96: 96px;

  --radius-s: 8px; --radius-m: 12px; --radius-l: 16px;

  font-family: var(--font-sans); color: var(--fg); background: var(--bg);
  -webkit-font-smoothing: antialiased;
}

[data-theme="dark"] {
  --bg: var(--neutral-900); --surface: var(--neutral-850, #1d1d21);
  --fg: var(--neutral-25); --fg-secondary: var(--neutral-300);
  --fg-tertiary: var(--neutral-300);
  --success-fg: #6ee7b7; --error-fg: #fca5a5;
  --border: var(--neutral-700); --border-strong: var(--neutral-600);
}
[data-theme="dark"] a { color: #a5b4fc; }
/* No-JS fallback: visitors whose browser never sets data-theme (script
   disabled) still follow the OS preference. JS users always get data-theme,
   so :not([data-theme]) excludes them and prevents a light-chosen user on a
   dark OS from inheriting the dark media query. */
@media (prefers-color-scheme: dark) {
  :root:not([data-theme]) {
    --bg: var(--neutral-900); --surface: var(--neutral-850, #1d1d21);
    --fg: var(--neutral-25); --fg-secondary: var(--neutral-300);
    --fg-tertiary: var(--neutral-300);
    --success-fg: #6ee7b7; --error-fg: #fca5a5;
    --border: var(--neutral-700); --border-strong: var(--neutral-600);
  }
  :root:not([data-theme]) a { color: #a5b4fc; }
}

*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body { min-height: 100vh; }
a { color: var(--brand-link); text-decoration: none; }
a:hover { opacity: 0.8; }
[data-theme="dark"] a { color: #a5b4fc; }

/* Nav */
.nav { position: sticky; top: 0; z-index: 100; background: rgba(255,255,255,0.95); backdrop-filter: blur(12px); border-bottom: 1px solid var(--border); }
[data-theme="dark"] .nav { background: rgba(25,25,28,0.95); }
@media (prefers-color-scheme: dark) { html:not([data-theme]) .nav { background: rgba(25,25,28,0.95); } }
.nav-inner { max-width: 1200px; margin: 0 auto; padding: 0 var(--space-24); display: flex; align-items: center; height: 56px; gap: var(--space-24); }
.nav-logo { display: flex; align-items: center; gap: var(--space-8); font-weight: 700; font-size: 1.125rem; color: var(--fg); text-decoration: none; }
.nav-logo-icon { width: 28px; height: 28px; background: var(--brand-link); border-radius: var(--radius-s); display: flex; align-items: center; justify-content: center; color: var(--fg-on-accent); font-size: 14px; font-weight: 700; }
.nav-links { display: flex; gap: var(--space-16); list-style: none; }
.nav-links a { font-size: 0.875rem; font-weight: 500; color: var(--fg-secondary); padding: var(--space-4) var(--space-8); border-radius: var(--radius-s); transition: background 0.15s; }
.nav-links a:hover { background: rgba(79,70,229,0.06); color: var(--fg); }
.nav-spacer { flex: 1; }
.nav-stars { font-size: 0.8rem; color: var(--fg-tertiary); white-space: nowrap; display: flex; align-items: center; gap: var(--space-4); }
.nav-cta { padding: var(--space-6) var(--space-12); background: var(--brand-link); color: var(--fg-on-accent) !important; border-radius: var(--radius-s); font-size: 0.8rem; font-weight: 600; border: none; cursor: pointer; text-decoration: none; transition: background 0.15s; white-space: nowrap; }
.nav-cta:hover { background: var(--brand-700); }

@media (max-width: 768px) {
  .nav-links { display: none; }
  .nav-inner { padding: 0 var(--space-16); }
}

/* Hero */
.hero { max-width: 1200px; margin: 0 auto; padding: var(--space-96) var(--space-24) var(--space-64); display: flex; flex-direction: column; align-items: center; text-align: center; }
.hero h1 { font-size: clamp(2rem, 5vw, 3.5rem); font-weight: 700; letter-spacing: -0.03em; line-height: 1.1; max-width: 720px; background: linear-gradient(135deg, var(--fg) 0%, var(--fg-secondary) 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }
.hero p { font-size: 1.125rem; color: var(--fg-secondary); max-width: 600px; margin-top: var(--space-16); line-height: 1.6; }
.hero-ctas { display: flex; gap: var(--space-12); margin-top: var(--space-32); }
.hero-cta-primary { padding: var(--space-12) var(--space-24); background: var(--brand-link); color: var(--fg-on-accent) !important; border-radius: var(--radius-s); font-weight: 600; font-size: 1rem; cursor: pointer; transition: background 0.15s; text-decoration: none; }
.hero-cta-primary:hover { background: var(--brand-700); }
.hero-cta-secondary { padding: var(--space-12) var(--space-24); background: var(--surface); color: var(--fg); border: 1px solid var(--border-strong); border-radius: var(--radius-s); font-weight: 500; font-size: 1rem; cursor: pointer; transition: background 0.15s; text-decoration: none; }
.hero-cta-secondary:hover { background: var(--neutral-25); }
[data-theme="dark"] .hero-cta-secondary:hover { background: var(--neutral-800); }

/* Rainbow accent (June) — gradient CTA fill matches the admin rainbow buttons. */
[data-accent-rainbow="true"] .hero-cta-primary,
[data-accent-rainbow="true"] .nav-cta {
  background: linear-gradient(90deg, #ef4444, #f59e0b, #22c55e, #3b82f6, #a855f7);
  text-shadow: 0 1px 2px rgba(0,0,0,0.3);
}

/* Dashboard Mockup */
.mockup { margin-top: var(--space-48); width: 100%; max-width: 800px; border: 1px solid var(--border); border-radius: var(--radius-l); overflow: hidden; background: var(--surface); box-shadow: 0 4px 24px rgba(0,0,0,0.06); }
.mockup-header { display: flex; align-items: center; gap: var(--space-8); padding: var(--space-12) var(--space-16); border-bottom: 1px solid var(--border); }
.mockup-dot { width: 10px; height: 10px; border-radius: 50%; }
.mockup-dot:nth-child(1) { background: #ef4444; }
.mockup-dot:nth-child(2) { background: #f59e0b; }
.mockup-dot:nth-child(3) { background: #22c55e; }
.mockup-body { padding: var(--space-16); display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-12); }
.mockup-card { background: var(--neutral-40); border-radius: var(--radius-s); padding: var(--space-12); }
.mockup-card-title { font-size: 0.65rem; text-transform: uppercase; letter-spacing: 0.06em; color: #4b4b50; margin-bottom: var(--space-6); }
.mockup-row { display: flex; align-items: center; gap: var(--space-6); padding: var(--space-4) 0; font-size: 0.72rem; color: #4b4b50; }
.mockup-row .dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.mockup-stat { font-size: 1.5rem; font-weight: 700; color: #4b4b50; }
.mockup-bar { height: 6px; background: var(--neutral-200); border-radius: 3px; overflow: hidden; margin-top: var(--space-4); }
.mockup-bar-fill { height: 100%; border-radius: 3px; background: var(--brand-500); width: 60%; }

@media (max-width: 768px) {
  .hero { padding: var(--space-48) var(--space-16) var(--space-32); }
  .hero-ctas { flex-direction: column; align-items: center; }
  .mockup-body { grid-template-columns: 1fr; }
}

/* Sections */
.section { max-width: 1200px; margin: 0 auto; padding: var(--space-64) var(--space-24); }
.section-title { text-align: center; font-size: clamp(1.25rem, 3vw, 1.75rem); font-weight: 700; letter-spacing: -0.02em; margin-bottom: var(--space-48); }

/* Features Grid */
.features-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: var(--space-16); }
.feature-card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-m); padding: var(--space-20); transition: border-color 0.15s, box-shadow 0.15s; }
.feature-card:hover { border-color: var(--brand-500); box-shadow: 0 2px 12px rgba(79,70,229,0.08); }
.feature-icon { width: 40px; height: 40px; border-radius: var(--radius-s); background: rgba(79,70,229,0.1); display: flex; align-items: center; justify-content: center; font-size: 1.25rem; margin-bottom: var(--space-12); }
.feature-name { font-size: 0.95rem; font-weight: 600; margin-bottom: var(--space-4); }
.feature-desc { font-size: 0.8rem; color: var(--fg-secondary); line-height: 1.5; }

@media (max-width: 900px) { .features-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 480px) { .features-grid { grid-template-columns: 1fr; } }

/* Component Table */
.comp-section { max-width: 1200px; margin: 0 auto; padding: 0 var(--space-24) var(--space-64); }
.comp-table { width: 100%; border-collapse: collapse; background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-s); overflow: hidden; }
.comp-table th { padding: var(--space-10) var(--space-12); text-align: left; font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; color: #4b4b50; background: var(--neutral-40); border-bottom: 1px solid var(--border); }
.comp-table td { padding: var(--space-8) var(--space-12); font-size: 0.8rem; border-bottom: 1px solid var(--border); }
.comp-table tr:last-child td { border-bottom: none; }
.status-running { color: var(--success-fg); font-weight: 500; }
.status-stopped { color: var(--error-fg); }

/* Differentiators */
.diff-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--space-16); }
.diff-card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-m); padding: var(--space-20); }
.diff-icon { font-size: 1.5rem; margin-bottom: var(--space-8); }
.diff-name { font-size: 0.9rem; font-weight: 600; margin-bottom: var(--space-4); }
.diff-desc { font-size: 0.8rem; color: var(--fg-secondary); line-height: 1.5; }

@media (max-width: 768px) { .diff-grid { grid-template-columns: 1fr; } }

/* CTA */
.cta-section { max-width: 1200px; margin: 0 auto; padding: var(--space-64) var(--space-24); text-align: center; }
.cta-section h2 { font-size: clamp(1.25rem, 3vw, 1.75rem); font-weight: 700; letter-spacing: -0.02em; margin-bottom: var(--space-16); }
.cta-section p { font-size: 1rem; color: var(--fg-secondary); margin-bottom: var(--space-24); }

/* Footer */
.footer { border-top: 1px solid var(--border); background: var(--surface); padding: var(--space-48) var(--space-24) var(--space-24); }
.footer-inner { max-width: 1200px; margin: 0 auto; display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--space-32); }
.footer-col h4 { font-size: 0.75rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.06em; color: var(--fg-tertiary); margin-bottom: var(--space-12); }
.footer-col ul { list-style: none; }
.footer-col li { margin-bottom: var(--space-6); }
.footer-col a { font-size: 0.8rem; color: var(--fg-secondary); }
.footer-col a:hover { color: var(--fg); }
.footer-bottom { max-width: 1200px; margin: var(--space-32) auto 0; padding-top: var(--space-16); border-top: 1px solid var(--border); display: flex; justify-content: space-between; font-size: 0.75rem; color: var(--fg-tertiary); }

@media (max-width: 768px) { .footer-inner { grid-template-columns: 1fr; } }
</style>
</head>
<body>

<!-- Nav -->
<nav class="nav">
  <div class="nav-inner">
    <a href="/" class="nav-logo"><span class="nav-logo-icon">B</span> BigBase</a>
    <ul class="nav-links">
      <li><a href="#features">Features</a></li>
      <li><a href="#components">Components</a></li>
      <li><a href="/docs">Docs</a></li>
      <li><a href="/admin/">Admin</a></li>
      <li><a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noreferrer">{{.GitHubStars}}</a></li>
    </ul>
    <div class="nav-spacer"></div>
    <a href="/admin/" class="nav-cta">Launch Admin</a>
  </div>
</nav>

<!-- Hero -->
<section class="hero">
  <h1>The open-source BaaS that runs anywhere</h1>
  <p>Single-binary platform with Auth, Database, Storage, Functions, Messaging, Deploy, Realtime, and Git Repos &mdash; backed by SQLite or PostgreSQL. One binary, zero dependencies.</p>
  <div class="hero-ctas">
    <a href="/admin/" class="hero-cta-primary">Launch Admin</a>
    <a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noreferrer" class="hero-cta-secondary">View on GitHub &rarr;</a>
  </div>
  <div class="mockup">
    <div class="mockup-header">
      <span class="mockup-dot"></span><span class="mockup-dot"></span><span class="mockup-dot"></span>
    </div>
    <div class="mockup-body">
      <div class="mockup-card">
        <div class="mockup-card-title">Activity</div>
        <div class="mockup-row"><span class="dot" style="background:var(--success)"></span> Deploy web-app completed</div>
        <div class="mockup-row"><span class="dot" style="background:var(--warning)"></span> Build api-service running</div>
        <div class="mockup-row"><span class="dot" style="background:var(--success)"></span> Auth DB migration done</div>
        <div class="mockup-row"><span class="dot" style="background:var(--error)"></span> Push notification failed</div>
      </div>
      <div style="display:flex;flex-direction:column;gap:var(--space-12)">
        <div class="mockup-card">
          <div class="mockup-card-title">Components</div>
          <div class="mockup-stat">12</div>
          <div class="mockup-row" style="color:#4b4b50">all running</div>
        </div>
        <div class="mockup-card">
          <div class="mockup-card-title">CPU</div>
          <div class="mockup-stat" style="font-size:1rem">23%</div>
          <div class="mockup-bar"><div class="mockup-bar-fill" style="width:23%"></div></div>
        </div>
        <div class="mockup-card">
          <div class="mockup-card-title">Memory</div>
          <div class="mockup-stat" style="font-size:1rem">512 MB</div>
          <div class="mockup-bar"><div class="mockup-bar-fill" style="width:42%;background:var(--warning)"></div></div>
        </div>
      </div>
    </div>
  </div>
</section>

<!-- Features -->
<section class="section" id="features">
  <h2 class="section-title">Everything you need to build and scale</h2>
  <div class="features-grid">
    <div class="feature-card">
      <div class="feature-icon">🔐</div>
      <div class="feature-name">Auth</div>
      <div class="feature-desc">Google OAuth, email/password, JWT tokens. Multi-factor authentication out of the box.</div>
    </div>
    <div class="feature-card">
      <div class="feature-icon">🗄</div>
      <div class="feature-name">Database</div>
      <div class="feature-desc">SQLite or PostgreSQL. Built-in Data Studio and SQL Editor for browsing and querying.</div>
    </div>
    <div class="feature-card">
      <div class="feature-icon">📦</div>
      <div class="feature-name">Storage</div>
      <div class="feature-desc">File upload, download, and management. Secure access controls.</div>
    </div>
    <div class="feature-card">
      <div class="feature-icon">⚡</div>
      <div class="feature-name">Functions</div>
      <div class="feature-desc">Serverless JavaScript runtime. HTTP triggers, scheduled execution, and event hooks.</div>
    </div>
    <div class="feature-card">
      <div class="feature-icon">✉️</div>
      <div class="feature-name">Messaging</div>
      <div class="feature-desc">Email, SMS, and Push notification channels. Send messages through a single API.</div>
    </div>
    <div class="feature-card">
      <div class="feature-icon">🚀</div>
      <div class="feature-name">Deploy</div>
      <div class="feature-desc">Git-based deployments with auto-polling. CI/CD pipelines built in.</div>
    </div>
    <div class="feature-card">
      <div class="feature-icon">🔄</div>
      <div class="feature-name">Realtime</div>
      <div class="feature-desc">WebSocket subscriptions for live events. Subscribe to any database or system event.</div>
    </div>
    <div class="feature-card">
      <div class="feature-icon">📂</div>
      <div class="feature-name">Git Repos</div>
      <div class="feature-desc">Built-in Git repository hosting. Create, manage, and deploy from your repos.</div>
    </div>
  </div>
</section>

<!-- Component Status -->
<section class="comp-section" id="components">
  <h2 class="section-title" style="margin-bottom:var(--space-24)">System Status</h2>
  {{if .Components}}
  <table class="comp-table">
    <thead><tr><th>Name</th><th>Version</th><th>Status</th><th>Dependencies</th></tr></thead>
    <tbody>
    {{range .Components}}
      <tr>
        <td><strong>{{.Name}}</strong></td>
        <td>{{.Version}}</td>
        <td>{{if .Running}}<span class="status-running">● running</span>{{else}}<span class="status-stopped">● stopped</span>{{end}}</td>
        <td>{{if .Dependencies}}{{join .Dependencies ", "}}{{else}}<span style="color:var(--fg-tertiary)">none</span>{{end}}</td>
      </tr>
    {{end}}
    </tbody>
  </table>
  {{else}}
  <p style="text-align:center;color:var(--fg-tertiary)">No components registered.</p>
  {{end}}
</section>

<!-- Why BigBase -->
<section class="section">
  <h2 class="section-title">Why BigBase</h2>
  <div class="diff-grid">
    <div class="diff-card">
      <div class="diff-icon">🎯</div>
      <div class="diff-name">Single Binary</div>
      <div class="diff-desc">One binary, zero dependencies. Run on any server, any OS. No npm install, no pip, no gem.</div>
    </div>
    <div class="diff-card">
      <div class="diff-icon">🔒</div>
      <div class="diff-name">No Vendor Lock-In</div>
      <div class="diff-desc">Self-hosted, own your data. No cloud dependency, no surprise bills. Your infrastructure, your rules.</div>
    </div>
    <div class="diff-card">
      <div class="diff-icon">🏗</div>
      <div class="diff-name">ECC Architecture</div>
      <div class="diff-desc">Pluggable component system. Add, remove, or customize components without touching the core.</div>
    </div>
    <div class="diff-card">
      <div class="diff-icon">📊</div>
      <div class="diff-name">Admin UI Included</div>
      <div class="diff-desc">Full dashboard with Data Studio, SQL Editor, user management, and monitoring. Ready out of the box.</div>
    </div>
    <div class="diff-card">
      <div class="diff-icon">🗃</div>
      <div class="diff-name">Flexible Storage</div>
      <div class="diff-desc">Start with SQLite for development, scale to PostgreSQL for production. No migration hassle.</div>
    </div>
    <div class="diff-card">
      <div class="diff-icon">🛡</div>
      <div class="diff-name">Built-in Auth</div>
      <div class="diff-desc">Google OAuth, email/password authentication, and JWT tokens. Ready to use in minutes.</div>
    </div>
  </div>
</section>

<!-- CTA -->
<section class="cta-section">
  <h2>Ready to build something great?</h2>
  <p>Get started in minutes. One binary, full platform.</p>
  <a href="/admin/" class="hero-cta-primary">Launch Admin &rarr;</a>
</section>

<!-- Footer -->
<footer class="footer">
  <div class="footer-inner">
    <div class="footer-col">
      <h4>Product</h4>
      <ul>
        <li><a href="#features">Auth</a></li>
        <li><a href="#features">Database</a></li>
        <li><a href="#features">Storage</a></li>
        <li><a href="#features">Functions</a></li>
        <li><a href="#features">Messaging</a></li>
        <li><a href="#features">Deploy</a></li>
        <li><a href="#features">Git Repos</a></li>
      </ul>
    </div>
    <div class="footer-col">
      <h4>Resources</h4>
      <ul>
        <li><a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noreferrer">Documentation</a></li>
        <li><a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noreferrer">API Reference</a></li>
        <li><a href="https://github.com/danielvm-git/bigbase/blob/main/CHANGELOG.md" target="_blank" rel="noreferrer">Changelog</a></li>
        <li><a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noreferrer">Source Code</a></li>
      </ul>
    </div>
    <div class="footer-col">
      <h4>Community</h4>
      <ul>
        <li><a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noreferrer">GitHub</a></li>
        <li><a href="https://github.com/danielvm-git/bigbase/issues" target="_blank" rel="noreferrer">Issues</a></li>
        <li><a href="https://github.com/danielvm-git/bigbase/discussions" target="_blank" rel="noreferrer">Discussions</a></li>
      </ul>
    </div>
  </div>
  <div class="footer-bottom">
    <span>&copy; 2026 BigBase &middot; MIT License &middot; v{{.Version}}</span>
    <span>Built with <a href="https://github.com/danielvm-git/bigpowers" target="_blank" rel="noreferrer">BigPowers</a> by <a href="https://github.com/danielvm-git" target="_blank" rel="noreferrer">danielvm-git</a></span>
  </div>
</footer>

</body>
</html>`

var docsTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>BigBase Docs — Open-Source BaaS</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
<style>
:root {
  --neutral-0: #fff; --neutral-25: #fafafb; --neutral-40: #f4f4f7;
  --neutral-50: #ededf0; --neutral-100: #e4e4e7; --neutral-200: #d8d8db;
  --neutral-300: #adadb0; --neutral-400: #4b4b50; --neutral-500: #818186;
  --neutral-600: #6c6c71; --neutral-700: #56565c; --neutral-800: #2d2d31;
  --neutral-900: #19191c;
  --brand-500: #4f46e5; --brand-600: #4338ca; --brand-700: #3730a3; --brand-link: #3730a3;
  --success: #22c55e; --success-fg: #044e3a; --error: #ef4444; --error-fg: #781414; --warning: #f59e0b;
  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  --font-mono: 'Fira Code', 'SF Mono', Monaco, Consolas, monospace;
  --bg: var(--neutral-25); --surface: var(--neutral-0);
  --fg: var(--neutral-800); --fg-secondary: #4b4b50; --fg-tertiary: var(--neutral-400); --fg-on-accent: #fff;
  --border: var(--neutral-100); --border-strong: var(--neutral-200);
  --space-2: 2px; --space-4: 4px; --space-6: 6px; --space-8: 8px;
  --space-10: 10px; --space-12: 12px; --space-16: 16px; --space-20: 20px;
  --space-24: 24px; --space-32: 32px; --space-40: 40px; --space-48: 48px;
  --space-64: 64px; --space-96: 96px;
  --radius-s: 8px; --radius-m: 12px; --radius-l: 16px;
  font-family: var(--font-sans); color: var(--fg); background: var(--bg);
  -webkit-font-smoothing: antialiased;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: var(--neutral-900); --surface: var(--neutral-850, #1d1d21);
    --fg: var(--neutral-25); --fg-secondary: var(--neutral-300);
    --fg-tertiary: var(--neutral-300); --success-fg: #6ee7b7; --error-fg: #fca5a5;
    --border: var(--neutral-700); --border-strong: var(--neutral-600);
  }
  :root a { color: #a5b4fc; }
}
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body { min-height: 100vh; }
a { color: var(--brand-link); text-decoration: none; }
a:hover { opacity: 0.8; }
[data-theme="dark"] a { color: #a5b4fc; }

/* Nav */
.nav { position: sticky; top: 0; z-index: 100; background: rgba(255,255,255,0.95); backdrop-filter: blur(12px); border-bottom: 1px solid var(--border); }
[data-theme="dark"] .nav { background: rgba(25,25,28,0.95); }
@media (prefers-color-scheme: dark) { html:not([data-theme]) .nav { background: rgba(25,25,28,0.95); } }
.nav-inner { max-width: 1200px; margin: 0 auto; padding: 0 var(--space-24); display: flex; align-items: center; height: 56px; gap: var(--space-24); }
.nav-logo { display: flex; align-items: center; gap: var(--space-8); font-weight: 700; font-size: 1.125rem; color: var(--fg); text-decoration: none; }
.nav-logo-icon { width: 28px; height: 28px; background: var(--brand-link); border-radius: var(--radius-s); display: flex; align-items: center; justify-content: center; color: var(--fg-on-accent); font-size: 14px; font-weight: 700; }
.nav-links { display: flex; gap: var(--space-16); list-style: none; }
.nav-links a { font-size: 0.875rem; font-weight: 500; color: var(--fg-secondary); padding: var(--space-4) var(--space-8); border-radius: var(--radius-s); transition: background 0.15s; }
.nav-links a:hover { background: rgba(79,70,229,0.06); color: var(--fg); }
.nav-spacer { flex: 1; }
.nav-cta { padding: var(--space-6) var(--space-12); background: var(--brand-link); color: var(--fg-on-accent) !important; border-radius: var(--radius-s); font-size: 0.8rem; font-weight: 600; border: none; cursor: pointer; text-decoration: none; transition: background 0.15s; white-space: nowrap; }
.nav-cta:hover { background: var(--brand-700); }
@media (max-width: 768px) { .nav-links { display: none; } .nav-inner { padding: 0 var(--space-16); } }

/* Docs Layout */
.docs-wrap { display: flex; max-width: 1200px; margin: 0 auto; padding: var(--space-32) var(--space-24); gap: var(--space-32); min-height: calc(100vh - 56px); }

/* Sidebar */
.docs-sidebar { width: 220px; flex-shrink: 0; position: sticky; top: 72px; align-self: start; }
.docs-sidebar-section { margin-bottom: var(--space-20); }
.docs-sidebar-title { font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.06em; color: var(--fg-tertiary); margin-bottom: var(--space-8); }
.docs-sidebar-nav { list-style: none; }
.docs-sidebar-nav li { margin-bottom: var(--space-2); }
.docs-sidebar-nav a { display: block; padding: var(--space-3) var(--space-8); border-radius: var(--radius-s); font-size: 0.8rem; color: var(--fg-secondary); transition: background 0.15s, color 0.15s; }
.docs-sidebar-nav a:hover { background: rgba(79,70,229,0.06); color: var(--fg); }
.docs-sidebar-nav a.active { background: rgba(79,70,229,0.1); color: var(--brand-600); font-weight: 500; }

/* Main Content */
.docs-content { flex: 1; min-width: 0; }
.docs-content h1 { font-size: 2rem; font-weight: 700; letter-spacing: -0.02em; margin-bottom: var(--space-6); line-height: 1.2; }
.docs-content h2 { font-size: 1.35rem; font-weight: 600; letter-spacing: -0.015em; margin: var(--space-40) 0 var(--space-12); padding-bottom: var(--space-6); border-bottom: 1px solid var(--border); }
.docs-content h3 { font-size: 1rem; font-weight: 600; margin: var(--space-24) 0 var(--space-8); }
.docs-content p { font-size: 0.88rem; line-height: 1.7; color: var(--fg-secondary); margin-bottom: var(--space-12); }
.docs-content code { font-family: var(--font-mono); font-size: 0.82em; background: var(--neutral-40); padding: var(--space-1) var(--space-4); border-radius: var(--radius-xs, 4px); color: #2d2d31; }
.docs-content pre { background: var(--neutral-900); color: #e4e4e7; padding: var(--space-16); border-radius: var(--radius-s); overflow-x: auto; margin: var(--space-12) 0; font-size: 0.78rem; line-height: 1.6; }
.docs-content p a { text-decoration: underline; }
.docs-content pre a { color: #a5b4fc; }
.docs-content pre code { background: none; padding: 0; color: inherit; font-size: inherit; }
.docs-content ul { margin: var(--space-8) 0 var(--space-16); padding-left: var(--space-20); }
.docs-content li { font-size: 0.88rem; line-height: 1.7; color: var(--fg-secondary); margin-bottom: var(--space-4); }
.docs-content strong { color: var(--fg); }
.docs-content .cmd { background: var(--neutral-900); color: #e4e4e7; padding: var(--space-12) var(--space-16); border-radius: var(--radius-s); font-family: var(--font-mono); font-size: 0.82rem; margin: var(--space-8) 0 var(--space-16); }
.docs-content .cmd span { color: var(--success-fg); }
.docs-content .flag { display: inline-block; background: rgba(79,70,229,0.1); color: var(--brand-600); padding: var(--space-1) var(--space-4); border-radius: var(--radius-xs, 4px); font-family: var(--font-mono); font-size: 0.8rem; }
.docs-content table { width: 100%; border-collapse: collapse; margin: var(--space-12) 0 var(--space-24); font-size: 0.82rem; }
.docs-content th { text-align: left; padding: var(--space-6) var(--space-8); font-weight: 600; font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--fg-tertiary); border-bottom: 2px solid var(--border); }
.docs-content td { padding: var(--space-6) var(--space-8); border-bottom: 1px solid var(--border); color: var(--fg-secondary); }
.docs-content .flag-table td:first-child { font-family: var(--font-mono); font-size: 0.8rem; color: var(--fg); }

@media (max-width: 768px) { .docs-sidebar { display: none; } .docs-wrap { padding: var(--space-16); } }

/* Footer */
.footer { border-top: 1px solid var(--border); background: var(--surface); padding: var(--space-32) var(--space-24) var(--space-16); }
.footer-inner { max-width: 1200px; margin: 0 auto; display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--space-24); }
.footer-col h4 { font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.06em; color: var(--fg-tertiary); margin-bottom: var(--space-8); }
.footer-col ul { list-style: none; }
.footer-col li { margin-bottom: var(--space-4); }
.footer-col a { font-size: 0.78rem; color: var(--fg-secondary); }
.footer-col a:hover { color: var(--fg); }
.footer-bottom { max-width: 1200px; margin: var(--space-20) auto 0; padding-top: var(--space-12); border-top: 1px solid var(--border); display: flex; justify-content: space-between; font-size: 0.72rem; color: var(--fg-tertiary); }
@media (max-width: 768px) { .footer-inner { grid-template-columns: 1fr; } }
</style>
</head>
<body>

<nav class="nav">
  <div class="nav-inner">
    <a href="/" class="nav-logo"><span class="nav-logo-icon">B</span> BigBase</a>
    <ul class="nav-links">
      <li><a href="/">Home</a></li>
      <li><a href="/docs">Docs</a></li>
      <li><a href="/admin/">Admin</a></li>
      <li><a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noreferrer">{{.GitHubStars}}</a></li>
    </ul>
    <div class="nav-spacer"></div>
    <a href="/admin/" class="nav-cta">Launch Admin</a>
  </div>
</nav>

<div class="docs-wrap">
  <aside class="docs-sidebar">
    <div class="docs-sidebar-section">
      <div class="docs-sidebar-title">Getting Started</div>
      <ul class="docs-sidebar-nav">
        <li><a href="#quick-start">Quick Start</a></li>
        <li><a href="#cli">CLI Reference</a></li>
        <li><a href="#configuration">Configuration</a></li>
      </ul>
    </div>
    <div class="docs-sidebar-section">
      <div class="docs-sidebar-title">Platform</div>
      <ul class="docs-sidebar-nav">
        <li><a href="#admin-ui">Admin UI</a></li>
        <li><a href="#api">API Reference</a></li>
        <li><a href="#architecture">Architecture</a></li>
      </ul>
    </div>
    <div class="docs-sidebar-section">
      <div class="docs-sidebar-title">Components</div>
      <ul class="docs-sidebar-nav">
        <li><a href="#auth">Auth</a></li>
        <li><a href="#database">Database</a></li>
        <li><a href="#storage">Storage</a></li>
        <li><a href="#functions">Functions</a></li>
        <li><a href="#messaging">Messaging</a></li>
        <li><a href="#deploy">Deploy</a></li>
        <li><a href="#realtime">Realtime</a></li>
      </ul>
    </div>
  </aside>

  <main class="docs-content">

    <h1>BigBase Documentation</h1>
    <p>Everything you need to build and scale applications with BigBase. Single-binary backend platform with Auth, Database, Storage, Functions, Messaging, Deploy, Realtime, and Git Repos.</p>

    <h2 id="quick-start">Quick Start</h2>
    <p>Download the binary and start the server. BigBase runs as a single binary with zero external dependencies.</p>

    <div class="cmd">
      <span>$</span> go run . serve --port 8080 --db bigbase.db
    </div>

    <p>Open <a href="/admin/">/admin/</a> to access the admin dashboard. Register an account and start building.</p>

    <p>For Google OAuth support:</p>
    <div class="cmd">
      <span>$</span> go run . serve --port 8080 --db bigbase.db --google-client-id YOUR_ID --google-client-secret YOUR_SECRET
    </div>

    <h3>Build from source</h3>
    <p>BigBase requires Go 1.22+.</p>
    <div class="cmd">
      <span>$</span> git clone https://github.com/danielvm-git/bigbase<br>
      <span>$</span> cd bigbase<br>
      <span>$</span> go build -o bigbase .<br>
      <span>$</span> ./bigbase serve --port 8080
    </div>

    <h2 id="configuration">Configuration</h2>
    <h3>Flags</h3>
    <table class="flag-table">
      <thead><tr><th>Flag</th><th>Default</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td>--port</td><td>8080</td><td>HTTP server port</td></tr>
        <tr><td>--db</td><td>bigbase.db</td><td>SQLite database path</td></tr>
        <tr><td>--google-client-id</td><td></td><td>Google OAuth client ID</td></tr>
        <tr><td>--google-client-secret</td><td></td><td>Google OAuth client secret</td></tr>
      </tbody>
    </table>

    <h3>Database</h3>
    <p>By default BigBase uses <strong>SQLite</strong>. To switch to <strong>PostgreSQL</strong>, rename or symlink <code>bigbase.db</code> to a PostgreSQL connection string (coming in a future release).</p>

    <h2 id="cli">CLI Reference</h2>
    <table>
      <thead><tr><th>Command</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td><code>bigbase serve</code></td><td>Start the HTTP server (flags: --port, --db, --google-client-id, --google-client-secret)</td></tr>
        <tr><td><code>bigbase version</code></td><td>Show the current version</td></tr>
        <tr><td><code>bigbase status</code></td><td>Show kernel and component status</td></tr>
        <tr><td><code>bigbase components list</code></td><td>List all registered ECC components</td></tr>
        <tr><td><code>bigbase help</code></td><td>Show help text</td></tr>
      </tbody>
    </table>

    <h2 id="admin-ui">Admin UI</h2>
    <p>BigBase ships with a full-featured admin dashboard at <code>/admin/</code>.</p>

    <h3>Pages</h3>
    <table>
      <thead><tr><th>Route</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td><code>#/</code></td><td>Dashboard with system stats, charts, and activity feed</td></tr>
        <tr><td><code>#/data</code></td><td>Data Studio — browse database collections and records</td></tr>
        <tr><td><code>#/sql</code></td><td>SQL Editor — run arbitrary SQL queries</td></tr>
        <tr><td><code>#/users</code></td><td>User management — list and delete users</td></tr>
        <tr><td><code>#/repos</code></td><td>Git Repos — CRUD for Git repositories</td></tr>
        <tr><td><code>#/deploy</code></td><td>Deployments — Git-based deployment with CI/CD</td></tr>
        <tr><td><code>#/storage</code></td><td>Storage — file upload, download, and management</td></tr>
        <tr><td><code>#/messaging</code></td><td>Messaging — email, SMS, and push notifications</td></tr>
        <tr><td><code>#/functions</code></td><td>Functions — serverless JavaScript runtime</td></tr>
        <tr><td><code>#/forge</code></td><td>Forge — issue tracker and Kanban board</td></tr>
        <tr><td><code>#/cici</code></td><td>CI/CD — workflow configuration and run logs</td></tr>
        <tr><td><code>#/monitoring</code></td><td>Monitoring — system metrics, logs, and alerts</td></tr>
      </tbody>
    </table>

    <h2 id="api">API Reference</h2>
    <p>All API endpoints are served under <code>/api/</code>. Authentication is handled via JWT tokens set as HTTP-only cookies.</p>

    <h3>Auth</h3>
    <table>
      <thead><tr><th>Method</th><th>Path</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td>POST</td><td><code>/api/auth/login</code></td><td>Sign in with email and password</td></tr>
        <tr><td>POST</td><td><code>/api/auth/register</code></td><td>Create a new account</td></tr>
        <tr><td>GET</td><td><code>/api/auth/me</code></td><td>Get current user info</td></tr>
        <tr><td>GET</td><td><code>/api/auth/users</code></td><td>List all users (requires auth)</td></tr>
        <tr><td>DELETE</td><td><code>/api/auth/users/{id}</code></td><td>Delete a user (requires auth)</td></tr>
        <tr><td>GET</td><td><code>/api/auth/oauth/google</code></td><td>Initiate Google OAuth flow</td></tr>
      </tbody>
    </table>

    <h3>Database</h3>
    <table>
      <thead><tr><th>Method</th><th>Path</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td>GET</td><td><code>/api/collections/</code></td><td>List all table-like collections</td></tr>
        <tr><td>GET</td><td><code>/api/collections/{name}</code></td><td>Get records from a collection</td></tr>
        <tr><td>POST</td><td><code>/api/sql</code></td><td>Execute arbitrary SQL query</td></tr>
      </tbody>
    </table>

    <h3>Storage</h3>
    <table>
      <thead><tr><th>Method</th><th>Path</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td>POST</td><td><code>/api/storage/upload</code></td><td>Upload a file</td></tr>
        <tr><td>GET</td><td><code>/api/storage/files</code></td><td>List all files</td></tr>
        <tr><td>GET</td><td><code>/api/storage/files/{id}</code></td><td>Download a file</td></tr>
        <tr><td>DELETE</td><td><code>/api/storage/files/{id}</code></td><td>Delete a file</td></tr>
      </tbody>
    </table>

    <h3>Functions</h3>
    <table>
      <thead><tr><th>Method</th><th>Path</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td>GET</td><td><code>/api/functions</code></td><td>List all functions</td></tr>
        <tr><td>POST</td><td><code>/api/functions</code></td><td>Create a new function</td></tr>
        <tr><td>PUT</td><td><code>/api/functions/{id}</code></td><td>Update a function</td></tr>
        <tr><td>DELETE</td><td><code>/api/functions/{id}</code></td><td>Delete a function</td></tr>
        <tr><td>POST</td><td><code>/api/functions/{id}/run</code></td><td>Execute a function</td></tr>
      </tbody>
    </table>

    <h3>Messaging</h3>
    <table>
      <thead><tr><th>Method</th><th>Path</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td>POST</td><td><code>/api/messaging/email</code></td><td>Send an email</td></tr>
        <tr><td>POST</td><td><code>/api/messaging/sms</code></td><td>Send an SMS</td></tr>
        <tr><td>POST</td><td><code>/api/messaging/push</code></td><td>Send a push notification</td></tr>
        <tr><td>GET</td><td><code>/api/messaging/messages</code></td><td>List message history</td></tr>
      </tbody>
    </table>

    <h3>Deploy</h3>
    <table>
      <thead><tr><th>Method</th><th>Path</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td>GET</td><td><code>/api/deploy</code></td><td>List all deployments</td></tr>
        <tr><td>POST</td><td><code>/api/deploy</code></td><td>Create a new deployment</td></tr>
      </tbody>
    </table>

    <h3>Git</h3>
    <table>
      <thead><tr><th>Method</th><th>Path</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td>GET</td><td><code>/api/git/repos</code></td><td>List all repos</td></tr>
        <tr><td>POST</td><td><code>/api/git/repos</code></td><td>Create a repo</td></tr>
        <tr><td>DELETE</td><td><code>/api/git/repos/{id}</code></td><td>Delete a repo</td></tr>
      </tbody>
    </table>

    <h3>Monitoring</h3>
    <table>
      <thead><tr><th>Method</th><th>Path</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td>GET</td><td><code>/api/monitoring/metrics</code></td><td>System metrics (CPU, memory, goroutines)</td></tr>
        <tr><td>GET</td><td><code>/api/monitoring/logs</code></td><td>Application logs</td></tr>
        <tr><td>GET</td><td><code>/api/monitoring/alerts</code></td><td>Alert configurations</td></tr>
        <tr><td>POST</td><td><code>/api/monitoring/alerts</code></td><td>Create an alert</td></tr>
      </tbody>
    </table>

    <h2 id="architecture">Architecture</h2>
    <p>BigBase uses the <strong>Entity-Component-Construct (ECC)</strong> pattern. This is a modular architecture where each backend feature is a pluggable <strong>Component</strong> that registers with a central <strong>Kernel</strong>.</p>

    <h3>Key Concepts</h3>
    <ul>
      <li><strong>Kernel</strong> — Central lifecycle manager. Discovers components, starts/stops them, manages an event bus.</li>
      <li><strong>Component</strong> — A self-contained module implementing a feature (e.g., Auth, Database, Storage). Each component has Init → Start → Stop lifecycle.</li>
      <li><strong>Event Bus</strong> — Components communicate via events, not direct imports. This keeps them decoupled and individually testable.</li>
    </ul>

    <div class="cmd">
      <span>$</span> bigbase components list
    </div>

    <p>Components are registered in <code>main.go</code> before kernel start. Each component can emit and listen to events from other components. For example, the Auth component emits <code>user:created</code> when a new user registers, and other components can react to it.</p>

    <h2 id="auth">Auth</h2>
    <p>Built-in authentication with email/password and Google OAuth. JWT tokens are set as HTTP-only cookies. The <code>--google-client-id</code> and <code>--google-client-secret</code> flags configure Google OAuth.</p>

    <h2 id="database">Database</h2>
    <p>SQLite by default. Access your data through the Data Studio at <code>#/data</code> or the SQL Editor at <code>#/sql</code>. The SQL Editor supports arbitrary SQL queries with result display.</p>

    <h2 id="storage">Storage</h2>
    <p>File upload and management. Upload files through the admin UI or the API. Files are stored on disk and served with proper MIME types.</p>

    <h2 id="functions">Functions</h2>
    <p>Serverless JavaScript runtime. Write functions in JavaScript, trigger them via HTTP or a cron schedule. View execution logs and results in the admin UI.</p>

    <h2 id="messaging">Messaging</h2>
    <p>Send email, SMS, and push notifications through a single API. Message history is stored in the database.</p>

    <h2 id="deploy">Deploy</h2>
    <p>Git-based deployment system. Link a Git repository, specify a branch, and deploy with one click. Auto-polls for status updates.</p>

    <h3>SPA Fallback & Passthrough</h3>
    <p>By default, BigBase intercepts <code>/api/*</code> requests on deployed hosts to route them to the BigBase API. Static sites use Go's <code>http.FileServer</code>, which serves <code>index.html</code> for any path not matching a file (classic SPA fallback).</p>
    <p>If your app needs its own API endpoints or custom static assets, use the <code>passthrough_paths</code> field when creating a deployment:</p>
    <div class="cmd">
      <span>$</span> curl -X POST http://localhost:9999/api/deploy \<br>
      &nbsp;&nbsp;-H "Content-Type: application/json" \<br>
      &nbsp;&nbsp;-d '{"repo_id":"abc123","branch":"main","passthrough_paths":["/api/my-app/*","/version.json"]}'
    </div>
    <p>Paths matching a passthrough rule are forwarded directly to your app, bypassing BigBase's <code>/api/*</code> intercept.</p>

    <h3>Environment Metadata</h3>
    <p>BigBase injects a <code>__BIGBASE_METADATA__</code> script into HTML responses. Your app can read it to get deployment context without a separate API call:</p>
    <pre><code>// In your app's JavaScript:
console.log(window.__BIGBASE_METADATA__);
// { version: "abc1234", deployedAt: "2026-06-20T12:00:00Z" }</code></pre>

    <h2 id="realtime">Realtime</h2>
    <p>WebSocket-based realtime events. Subscribe to database changes, system events, and custom channels at <code>/realtime</code>.</p>

  </main>
</div>

<footer class="footer">
  <div class="footer-inner">
    <div class="footer-col">
      <h4>Product</h4>
      <ul>
        <li><a href="/">Overview</a></li>
        <li><a href="/admin/">Admin UI</a></li>
        <li><a href="/docs">Documentation</a></li>
      </ul>
    </div>
    <div class="footer-col">
      <h4>Resources</h4>
      <ul>
        <li><a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noreferrer">GitHub</a></li>
        <li><a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noreferrer">Source Code</a></li>
        <li><a href="https://github.com/danielvm-git/bigbase/issues" target="_blank" rel="noreferrer">Issues</a></li>
      </ul>
    </div>
    <div class="footer-col">
      <h4>Community</h4>
      <ul>
        <li><a href="https://github.com/danielvm-git/bigbase/discussions" target="_blank" rel="noreferrer">Discussions</a></li>
        <li><a href="https://github.com/danielvm-git/bigbase" target="_blank" rel="noreferrer">GitHub</a></li>
      </ul>
    </div>
  </div>
  <div class="footer-bottom">
    <span>&copy; 2026 BigBase &middot; MIT License &middot; v{{.Version}}</span>
    <span>Built with <a href="https://github.com/danielvm-git/bigpowers" target="_blank" rel="noreferrer">BigPowers</a> by <a href="https://github.com/danielvm-git" target="_blank" rel="noreferrer">danielvm-git</a></span>
  </div>
</footer>

</body>
</html>`

func (p *Proxy) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		if len(p.corsAllowedOrigins) == 0 {
			if !authSameOrigin(r, origin) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"origin not allowed"}`))
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		allowed := false
		for _, o := range p.corsAllowedOrigins {
			if o == origin {
				allowed = true
				break
			}
		}
		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"origin not allowed"}`))
			return
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Vary", "Origin")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")
		next.ServeHTTP(w, r)
	})
}

func authSameOrigin(r *http.Request, origin string) bool {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return origin == scheme+"://"+r.Host
}

type AuthPolicy struct {
	Default        string   `json:"default"`
	ProtectedPaths []string `json:"protected_paths"`
	PublicPaths    []string `json:"public_paths"`
	Accept         []string `json:"accept"`
}

func (p *Proxy) SetSiteAuthPolicy(siteID string, policyJSON string) {
	p.authPoliciesMu.Lock()
	defer p.authPoliciesMu.Unlock()
	if p.authPolicies == nil {
		p.authPolicies = make(map[string]*AuthPolicy)
	}
	if policyJSON == "" || policyJSON == "{}" {
		delete(p.authPolicies, siteID)
		return
	}
	var policy AuthPolicy
	if err := json.Unmarshal([]byte(policyJSON), &policy); err == nil {
		p.authPolicies[siteID] = &policy
	}
}

func (p *Proxy) getSiteAuthPolicy(siteID string) *AuthPolicy {
	p.authPoliciesMu.RLock()
	policy, ok := p.authPolicies[siteID]
	p.authPoliciesMu.RUnlock()
	if ok {
		return policy
	}

	// Fallback: check database if DB is available
	if p.db != nil {
		var policyStr string
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err := p.db.QueryRowContext(ctx, "SELECT auth_policy FROM sites WHERE id = ?", siteID).Scan(&policyStr)
		if err == nil && policyStr != "" {
			var dbPolicy AuthPolicy
			if err := json.Unmarshal([]byte(policyStr), &dbPolicy); err == nil {
				// Cache it
				p.authPoliciesMu.Lock()
				if p.authPolicies == nil {
					p.authPolicies = make(map[string]*AuthPolicy)
				}
				p.authPolicies[siteID] = &dbPolicy
				p.authPoliciesMu.Unlock()
				return &dbPolicy
			}
		}
	}
	return nil
}

func (p *Proxy) SetSiteCSP(siteID string, cspPolicy string) {
	p.cspPoliciesMu.Lock()
	defer p.cspPoliciesMu.Unlock()
	if p.cspPolicies == nil {
		p.cspPolicies = make(map[string]string)
	}
	if cspPolicy == "" {
		delete(p.cspPolicies, siteID)
		return
	}
	p.cspPolicies[siteID] = cspPolicy
}

func (p *Proxy) getSiteCSP(siteID string) string {
	p.cspPoliciesMu.RLock()
	csp, ok := p.cspPolicies[siteID]
	p.cspPoliciesMu.RUnlock()
	if ok {
		return csp
	}

	// Fallback: read deploy_defaults from database
	if p.db != nil {
		var ddStr string
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err := p.db.QueryRowContext(ctx, "SELECT deploy_defaults FROM sites WHERE id = ?", siteID).Scan(&ddStr)
		if err == nil && ddStr != "" && ddStr != "{}" {
			var dd struct {
				CSPPolicy string `json:"csp_policy"`
			}
			if err := json.Unmarshal([]byte(ddStr), &dd); err == nil && dd.CSPPolicy != "" {
				p.cspPoliciesMu.Lock()
				if p.cspPolicies == nil {
					p.cspPolicies = make(map[string]string)
				}
				p.cspPolicies[siteID] = dd.CSPPolicy
				p.cspPoliciesMu.Unlock()
				return dd.CSPPolicy
			}
		}
	}
	return ""
}
