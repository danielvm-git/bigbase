package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

var templateFuncs = template.FuncMap{
	"join": strings.Join,
}

const version = "0.1.0"

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type Options struct {
	Port   string
	Kernel *kernel.Kernel
	Logger Logger
}

type Proxy struct {
	port   string
	kernel *kernel.Kernel
	logger Logger
	server *http.Server
	mux    *http.ServeMux

	starsMu   sync.Mutex
	starsVal  string
	starsTime time.Time
}

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
		port:   opts.Port,
		kernel: opts.Kernel,
		logger: opts.Logger,
		mux:    http.NewServeMux(),
	}
}

func (p *Proxy) Name() string                     { return "proxy" }
func (p *Proxy) Version() string                  { return version }
func (p *Proxy) Dependencies() []string            { return nil }
func (p *Proxy) ConfigSchema() json.RawMessage     { return nil }
func (p *Proxy) Hooks() []kernel.HookDef           { return nil }

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

func (p *Proxy) Start(ctx *kernel.Context) error {
	p.mux.HandleFunc("/", p.handleHome)
	p.mux.HandleFunc("/health", p.handleHealth)

	p.server = &http.Server{
		Addr:    ":" + p.port,
		Handler: p.loggingMiddleware(p.mux),
	}

	go func() {
		p.logger.Info("proxy listening", "port", p.port)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			p.logger.Error("proxy server error", "error", err)
		}
	}()

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

func (p *Proxy) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		p.logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration", time.Since(start).String(),
		)
	})
}

func (p *Proxy) Handle(pattern string, handler http.HandlerFunc) {
	if p.mux != nil {
		p.mux.HandleFunc(pattern, handler)
	}
}

func (p *Proxy) Stop(ctx *kernel.Context) error {
	if p.server != nil {
		return p.server.Shutdown(context.Background())
	}
	return nil
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
	}{
		Version:     kernel.Version,
		Components:  components,
		GitHubStars: starsDisplay,
		Port:        p.port,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		p.logger.Error("template error", "error", err)
	}
}

func (p *Proxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := "ok"
	compCount := len(p.kernel.Components())
	_, _ = fmt.Fprintf(w, `{"status":"%s","components":%d}`, status, compCount)
}

var homeTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>BigBase — Open-Source BaaS</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
<style>
:root {
  --neutral-0: #fff; --neutral-25: #fafafb; --neutral-40: #f4f4f7;
  --neutral-50: #ededf0; --neutral-100: #e4e4e7; --neutral-200: #d8d8db;
  --neutral-300: #adadb0; --neutral-400: #97979b; --neutral-500: #818186;
  --neutral-600: #6c6c71; --neutral-700: #56565c; --neutral-800: #2d2d31;
  --neutral-900: #19191c;

  --brand-500: #4f46e5; --brand-600: #4338ca; --brand-700: #3730a3;
  --success: #22c55e; --error: #ef4444; --warning: #f59e0b;

  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  --font-mono: 'Fira Code', 'SF Mono', Monaco, Consolas, monospace;

  --bg: var(--neutral-25); --surface: var(--neutral-0);
  --fg: var(--neutral-800); --fg-secondary: var(--neutral-600); --fg-tertiary: var(--neutral-400);
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
    --fg-tertiary: var(--neutral-500);
    --border: var(--neutral-700); --border-strong: var(--neutral-600);
  }
}

*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body { min-height: 100vh; }
a { color: var(--brand-500); text-decoration: none; }
a:hover { opacity: 0.8; }

/* Nav */
.nav { position: sticky; top: 0; z-index: 100; background: rgba(255,255,255,0.8); backdrop-filter: blur(12px); border-bottom: 1px solid var(--border); }
.nav-inner { max-width: 1200px; margin: 0 auto; padding: 0 var(--space-24); display: flex; align-items: center; height: 56px; gap: var(--space-24); }
.nav-logo { display: flex; align-items: center; gap: var(--space-8); font-weight: 700; font-size: 1.125rem; color: var(--fg); text-decoration: none; }
.nav-logo-icon { width: 28px; height: 28px; background: var(--brand-500); border-radius: var(--radius-s); display: flex; align-items: center; justify-content: center; color: #fff; font-size: 14px; font-weight: 700; }
.nav-links { display: flex; gap: var(--space-16); list-style: none; }
.nav-links a { font-size: 0.875rem; font-weight: 500; color: var(--fg-secondary); padding: var(--space-4) var(--space-8); border-radius: var(--radius-s); transition: background 0.15s; }
.nav-links a:hover { background: rgba(79,70,229,0.06); color: var(--fg); }
.nav-spacer { flex: 1; }
.nav-stars { font-size: 0.8rem; color: var(--fg-tertiary); white-space: nowrap; display: flex; align-items: center; gap: var(--space-4); }
.nav-cta { padding: var(--space-6) var(--space-12); background: var(--brand-500); color: #fff !important; border-radius: var(--radius-s); font-size: 0.8rem; font-weight: 600; border: none; cursor: pointer; text-decoration: none; transition: background 0.15s; white-space: nowrap; }
.nav-cta:hover { background: var(--brand-600); }

@media (max-width: 768px) {
  .nav-links { display: none; }
  .nav-inner { padding: 0 var(--space-16); }
}

/* Hero */
.hero { max-width: 1200px; margin: 0 auto; padding: var(--space-96) var(--space-24) var(--space-64); display: flex; flex-direction: column; align-items: center; text-align: center; }
.hero h1 { font-size: clamp(2rem, 5vw, 3.5rem); font-weight: 700; letter-spacing: -0.03em; line-height: 1.1; max-width: 720px; background: linear-gradient(135deg, var(--fg) 0%, var(--fg-secondary) 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }
.hero p { font-size: 1.125rem; color: var(--fg-secondary); max-width: 600px; margin-top: var(--space-16); line-height: 1.6; }
.hero-ctas { display: flex; gap: var(--space-12); margin-top: var(--space-32); }
.hero-cta-primary { padding: var(--space-12) var(--space-24); background: var(--brand-500); color: #fff; border-radius: var(--radius-s); font-weight: 600; font-size: 1rem; cursor: pointer; transition: background 0.15s; text-decoration: none; }
.hero-cta-primary:hover { background: var(--brand-600); }
.hero-cta-secondary { padding: var(--space-12) var(--space-24); background: var(--surface); color: var(--fg); border: 1px solid var(--border-strong); border-radius: var(--radius-s); font-weight: 500; font-size: 1rem; cursor: pointer; transition: background 0.15s; text-decoration: none; }
.hero-cta-secondary:hover { background: var(--neutral-25); }

/* Dashboard Mockup */
.mockup { margin-top: var(--space-48); width: 100%; max-width: 800px; border: 1px solid var(--border); border-radius: var(--radius-l); overflow: hidden; background: var(--surface); box-shadow: 0 4px 24px rgba(0,0,0,0.06); }
.mockup-header { display: flex; align-items: center; gap: var(--space-8); padding: var(--space-12) var(--space-16); border-bottom: 1px solid var(--border); }
.mockup-dot { width: 10px; height: 10px; border-radius: 50%; }
.mockup-dot:nth-child(1) { background: #ef4444; }
.mockup-dot:nth-child(2) { background: #f59e0b; }
.mockup-dot:nth-child(3) { background: #22c55e; }
.mockup-body { padding: var(--space-16); display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-12); }
.mockup-card { background: var(--neutral-40); border-radius: var(--radius-s); padding: var(--space-12); }
.mockup-card-title { font-size: 0.65rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--fg-tertiary); margin-bottom: var(--space-6); }
.mockup-row { display: flex; align-items: center; gap: var(--space-6); padding: var(--space-4) 0; font-size: 0.72rem; }
.mockup-row .dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.mockup-stat { font-size: 1.5rem; font-weight: 700; color: var(--brand-500); }
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
.comp-table th { padding: var(--space-10) var(--space-12); text-align: left; font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; color: var(--fg-tertiary); background: var(--neutral-40); border-bottom: 1px solid var(--border); }
.comp-table td { padding: var(--space-8) var(--space-12); font-size: 0.8rem; border-bottom: 1px solid var(--border); }
.comp-table tr:last-child td { border-bottom: none; }
.status-running { color: var(--success); font-weight: 500; }
.status-stopped { color: var(--error); }

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
          <div class="mockup-row" style="color:var(--fg-secondary)">all running</div>
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
    <thead><tr><th>Name</th><th>Version</th><th>Status</th><th>Dependencies</th><th>Hooks</th></tr></thead>
    <tbody>
    {{range .Components}}
      <tr>
        <td><strong>{{.Name}}</strong></td>
        <td>{{.Version}}</td>
        <td>{{if .Running}}<span class="status-running">● running</span>{{else}}<span class="status-stopped">● stopped</span>{{end}}</td>
        <td>{{if .Dependencies}}{{join .Dependencies ", "}}{{else}}<span style="color:var(--fg-tertiary)">none</span>{{end}}</td>
        <td>{{if .Hooks}}{{join .Hooks ", "}}{{else}}<span style="color:var(--fg-tertiary)">none</span>{{end}}</td>
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
    <span>Built with Go &middot; ECC Architecture</span>
  </div>
</footer>

</body>
</html>`
