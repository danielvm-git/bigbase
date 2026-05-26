package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
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
}

func New(opts Options) *Proxy {
	return &Proxy{
		port:   opts.Port,
		kernel: opts.Kernel,
		logger: opts.Logger,
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
	p.mux = http.NewServeMux()
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

	data := struct {
		Version    string
		Components []kernel.ComponentStatus
	}{
		Version:    kernel.Version,
		Components: components,
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
<title>BigBase</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: system-ui, -apple-system, sans-serif; background: #f5f5f7; color: #1d1d1f; padding: 2rem; }
  h1 { font-size: 2rem; margin-bottom: 0.25rem; }
  .version { color: #86868b; margin-bottom: 2rem; }
  h2 { font-size: 1.25rem; margin: 1.5rem 0 0.75rem; }
  table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; }
  th, td { padding: 0.75rem 1rem; text-align: left; border-bottom: 1px solid #e5e5e7; }
  th { background: #f5f5f7; font-weight: 600; font-size: 0.85rem; text-transform: uppercase; letter-spacing: 0.05em; color: #86868b; }
  td { font-size: 0.9rem; }
  .running { color: #34c759; font-weight: 500; }
  .stopped { color: #ff3b30; }
  pre { background: #fff; padding: 1rem; border-radius: 8px; border: 1px solid #e5e5e7; font-size: 0.85rem; }
</style>
</head>
<body>
  <h1>BigBase</h1>
  <div class="version">v{{.Version}}</div>

  <h2>Components</h2>
  {{if .Components}}
  <table>
    <thead><tr><th>Name</th><th>Version</th><th>Status</th><th>Dependencies</th><th>Hooks</th></tr></thead>
    <tbody>
    {{range .Components}}
      <tr>
        <td><strong>{{.Name}}</strong></td>
        <td>{{.Version}}</td>
        <td>{{if .Running}}<span class="running">running</span>{{else}}<span class="stopped">stopped</span>{{end}}</td>
        <td>{{if .Dependencies}}{{join .Dependencies ", "}}{{else}}—{{end}}</td>
        <td>{{if .Hooks}}{{join .Hooks ", "}}{{else}}—{{end}}</td>
      </tr>
    {{end}}
    </tbody>
  </table>
  {{else}}
  <pre>No components registered.</pre>
  {{end}}

  <h2>Endpoints</h2>
  <table>
    <thead><tr><th>Path</th><th>Description</th></tr></thead>
    <tbody>
      <tr><td><code>/</code></td><td>Home page — this page</td></tr>
      <tr><td><code>/health</code></td><td>Health check — JSON status</td></tr>
    </tbody>
  </table>
</body>
</html>`
