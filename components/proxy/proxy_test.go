package proxy_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/proxy"
	"github.com/danielvm/bigbase/kernel"
)

func TestRequestIDMiddleware(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)

	port := freePort(t)
	p := proxy.New(proxy.Options{
		Port:   port,
		Kernel: k,
		Logger: logger,
	})

	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer func() { _ = p.Stop(&kernel.Context{}) }()

	waitForServer(t, port, "/health")

	t.Run("generates request ID when header absent", func(t *testing.T) {
		resp, err := http.Get("http://localhost:" + port + "/health")
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		rid := resp.Header.Get("X-Request-ID")
		if rid == "" {
			t.Fatal("expected X-Request-ID header in response")
		}
		if len(rid) < 16 {
			t.Fatalf("expected request ID of at least 16 hex chars, got %q (len=%d)", rid, len(rid))
		}
	})

	t.Run("preserves client-provided request ID", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://localhost:"+port+"/health", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		req.Header.Set("X-Request-ID", "my-test-id-123")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		rid := resp.Header.Get("X-Request-ID")
		if rid != "my-test-id-123" {
			t.Fatalf("expected X-Request-ID 'my-test-id-123', got %q", rid)
		}
	})

	t.Run("request ID is present on all routes", func(t *testing.T) {
		resp, err := http.Get("http://localhost:" + port + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		rid := resp.Header.Get("X-Request-ID")
		if rid == "" {
			t.Fatal("expected X-Request-ID on home page route")
		}
	})
}

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

type testComponents struct{}

func (t *testComponents) Name() string                  { return "testcomp" }
func (t *testComponents) Version() string               { return "0.0.1" }
func (t *testComponents) Dependencies() []string         { return nil }
func (t *testComponents) ConfigSchema() json.RawMessage  { return nil }
func (t *testComponents) Init(ctx *kernel.Context, config json.RawMessage) error { return nil }
func (t *testComponents) Start(ctx *kernel.Context) error { return nil }
func (t *testComponents) Stop(ctx *kernel.Context) error  { return nil }
func (t *testComponents) Hooks() []kernel.HookDef         { return nil }

func TestProxyImplementsComponent(t *testing.T) {
	var _ kernel.Component = &proxy.Proxy{}
}

func TestProxyName(t *testing.T) {
	p := &proxy.Proxy{}
	if got := p.Name(); got != "proxy" {
		t.Fatalf("expected Name()='proxy', got '%s'", got)
	}
}

func TestProxyVersion(t *testing.T) {
	p := &proxy.Proxy{}
	if p.Version() == "" {
		t.Fatal("expected non-empty Version()")
	}
}

func TestProxyInit(t *testing.T) {
	t.Run("default port", func(t *testing.T) {
		p := &proxy.Proxy{}
		if err := p.Init(nil, nil); err != nil {
			t.Fatalf("unexpected Init error: %v", err)
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		p := proxy.New(proxy.Options{Port: "abc"})
		ctx := &kernel.Context{}
		if err := p.Init(ctx, nil); err == nil {
			t.Fatal("expected error for invalid port 'abc'")
		}
	})

	t.Run("out of range port", func(t *testing.T) {
		p := proxy.New(proxy.Options{Port: "99999"})
		ctx := &kernel.Context{}
		if err := p.Init(ctx, nil); err == nil {
			t.Fatal("expected error for out-of-range port '99999'")
		}
	})
}

func TestProxyServeHomePage(t *testing.T) {
	comp := &testComponents{}
	logger := testLogger{}
	k := kernel.New(logger)
	k.Register(comp)

	if err := k.Start(); err != nil {
		t.Fatalf("failed to start kernel: %v", err)
	}
	defer func() { _ = k.Stop() }()

	port := freePort(t)
	p := proxy.New(proxy.Options{
		Port:   port,
		Kernel: k,
		Logger: logger,
	})

	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer func() { _ = p.Stop(&kernel.Context{}) }()

	waitForServer(t, port, "/health")

	resp, err := http.Get("http://localhost:" + port + "/")
	if err != nil {
		t.Fatalf("failed to GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	bodyStr := string(body)
	if !strings.Contains(bodyStr, "BigBase") {
		t.Fatal("expected page to contain 'BigBase'")
	}
	if !strings.Contains(bodyStr, "testcomp") {
		t.Fatal("expected page to list 'testcomp' component")
	}
	if !strings.Contains(bodyStr, kernel.Version) {
		t.Fatalf("expected page to show version %q", kernel.Version)
	}
}

func TestProxyHandle(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)

	port := freePort(t)
	p := proxy.New(proxy.Options{
		Port:   port,
		Kernel: k,
		Logger: logger,
	})

	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer func() { _ = p.Stop(&kernel.Context{}) }()

	waitForServer(t, port, "/health")

	var handled bool
	p.Handle("/api/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handled = true
		w.WriteHeader(http.StatusOK)
	}))

	resp, err := http.Get("http://localhost:" + port + "/api/test")
	if err != nil {
		t.Fatalf("failed to GET /api/test: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !handled {
		t.Fatal("expected handler to be called")
	}
}

func TestProxyGitHubStars(t *testing.T) {
	t.Run("starts with empty cache", func(t *testing.T) {
		p := &proxy.Proxy{}
		if stars := p.GitHubStars(); stars != "" {
			t.Fatalf("expected empty stars before fetch, got %q", stars)
		}
	})
}

func TestProxyHomePageCommercialContent(t *testing.T) {
	comp := &testComponents{}
	logger := testLogger{}
	k := kernel.New(logger)
	k.Register(comp)

	if err := k.Start(); err != nil {
		t.Fatalf("failed to start kernel: %v", err)
	}
	defer func() { _ = k.Stop() }()

	port := freePort(t)
	p := proxy.New(proxy.Options{
		Port:   port,
		Kernel: k,
		Logger: logger,
	})

	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer func() { _ = p.Stop(&kernel.Context{}) }()

	waitForServer(t, port, "/health")

	resp, err := http.Get("http://localhost:" + port + "/")
	if err != nil {
		t.Fatalf("failed to GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	bodyStr := string(body)

	if !strings.Contains(bodyStr, "text/html") &&
		!strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatal("expected Content-Type text/html")
	}

	if !strings.Contains(bodyStr, "BigBase") {
		t.Fatal("expected page to contain 'BigBase'")
	}
	if !strings.Contains(bodyStr, "testcomp") {
		t.Fatal("expected page to list 'testcomp' component")
	}

	if !strings.Contains(bodyStr, "Launch Admin") {
		t.Fatal("expected page to contain 'Launch Admin' CTA button")
	}

	if !strings.Contains(bodyStr, "open-source BaaS") {
		t.Fatal("expected page to contain hero headline")
	}

	if !strings.Contains(bodyStr, "Auth") {
		t.Fatal("expected page to contain feature card 'Auth'")
	}

	if !strings.Contains(bodyStr, "Why BigBase") {
		t.Fatal("expected page to contain differentiators section")
	}

	if !strings.Contains(bodyStr, "Product") && !strings.Contains(bodyStr, "Resources") {
		t.Fatal("expected page to contain footer with columns")
	}
}

func TestProxyDocsPage(t *testing.T) {
	comp := &testComponents{}
	logger := testLogger{}
	k := kernel.New(logger)
	k.Register(comp)

	if err := k.Start(); err != nil {
		t.Fatalf("failed to start kernel: %v", err)
	}
	defer func() { _ = k.Stop() }()

	port := freePort(t)
	p := proxy.New(proxy.Options{
		Port:   port,
		Kernel: k,
		Logger: logger,
	})

	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer func() { _ = p.Stop(&kernel.Context{}) }()

	waitForServer(t, port, "/health")

	resp, err := http.Get("http://localhost:" + port + "/docs")
	if err != nil {
		t.Fatalf("failed to GET /docs: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	bodyStr := string(body)

	if !strings.Contains(bodyStr, "BigBase") {
		t.Fatal("expected docs page to contain 'BigBase'")
	}
	if !strings.Contains(bodyStr, "Quick Start") {
		t.Fatal("expected docs page to contain 'Quick Start' section")
	}
	if !strings.Contains(bodyStr, "CLI Reference") {
		t.Fatal("expected docs page to have CLI Reference")
	}
	if !strings.Contains(bodyStr, "bigbase serve") {
		t.Fatal("expected docs page to mention 'bigbase serve' command")
	}
	if !strings.Contains(bodyStr, "--port") {
		t.Fatal("expected docs page to mention flags")
	}
	if !strings.Contains(bodyStr, "API Reference") {
		t.Fatal("expected docs page to have API Reference section")
	}
	if !strings.Contains(bodyStr, "BigPowers") {
		t.Fatal("expected docs page to explain architecture")
	}
}

func TestProxyHealthEndpoint(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)

	port := freePort(t)
	p := proxy.New(proxy.Options{
		Port:   port,
		Kernel: k,
		Logger: logger,
	})

	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer func() { _ = p.Stop(&kernel.Context{}) }()

	waitForServer(t, port, "/health")

	resp, err := http.Get("http://localhost:" + port + "/health")
	if err != nil {
		t.Fatalf("failed to GET /health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	if len(body) == 0 {
		t.Fatal("expected non-empty health response")
	}

	var health struct {
		Status     string `json:"status"`
		Components int    `json:"components"`
		Running    int    `json:"running"`
	}
	if err := json.Unmarshal(body, &health); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if health.Running > health.Components {
		t.Fatalf("expected running <= components, got running=%d components=%d", health.Running, health.Components)
	}
	if health.Components > 0 && health.Running < 1 {
		t.Fatalf("expected running >= 1 when components registered, got running=%d components=%d", health.Running, health.Components)
	}
	if health.Status != "ok" && health.Status != "degraded" {
		t.Fatalf("unexpected status %q", health.Status)
	}
}

func TestPerComponentHealth(t *testing.T) {
	comp := &testComponents{}
	logger := testLogger{}
	k := kernel.New(logger)
	k.Register(comp)

	if err := k.Start(); err != nil {
		t.Fatalf("failed to start kernel: %v", err)
	}
	defer func() { _ = k.Stop() }()

	port := freePort(t)
	p := proxy.New(proxy.Options{
		Port:   port,
		Kernel: k,
		Logger: logger,
	})

	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer func() { _ = p.Stop(&kernel.Context{}) }()

	waitForServer(t, port, "/health")

	resp, err := http.Get("http://localhost:" + port + "/health")
	if err != nil {
		t.Fatalf("failed to GET /health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Status             string `json:"status"`
		Running            int    `json:"running"`
		ComponentStatusMap []struct {
			Name         string   `json:"name"`
			Version      string   `json:"version"`
			Running      bool     `json:"running"`
			Dependencies []string `json:"dependencies"`
			Hooks        []string `json:"hooks"`
		} `json:"component_status_map"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode health: %v", err)
	}

	if len(result.ComponentStatusMap) == 0 {
		t.Fatal("expected non-empty component_status_map")
	}

	found := false
	for _, c := range result.ComponentStatusMap {
		if c.Name == "testcomp" {
			found = true
			if !c.Running {
				t.Error("expected testcomp to be running")
			}
			if c.Version == "" {
				t.Error("expected non-empty version for testcomp")
			}
			break
		}
	}
	if !found {
		t.Fatal("expected testcomp in component_status_map")
	}

	if result.Running != len(result.ComponentStatusMap) {
		t.Fatalf("expected running=%d to match component_status_map count=%d", result.Running, len(result.ComponentStatusMap))
	}
}

func TestProxyVersionEndpoint(t *testing.T) {
	comp := &testComponents{}
	logger := testLogger{}
	k := kernel.New(logger)
	k.Register(comp)

	if err := k.Start(); err != nil {
		t.Fatalf("failed to start kernel: %v", err)
	}
	defer func() { _ = k.Stop() }()

	port := freePort(t)
	p := proxy.New(proxy.Options{
		Port:   port,
		Kernel: k,
		Logger: logger,
	})

	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer func() { _ = p.Stop(&kernel.Context{}) }()

	waitForServer(t, port, "/api/version")

	resp, err := http.Get("http://localhost:" + port + "/api/version")
	if err != nil {
		t.Fatalf("failed to GET /api/version: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var result struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse JSON: %v (body: %s)", err, string(body))
	}

	if result.Version == "" {
		t.Fatal("expected non-empty version in response")
	}
}

func TestGitPathBlocked(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)

	port := freePort(t)
	p := proxy.New(proxy.Options{
		Port:   port,
		Kernel: k,
		Logger: logger,
	})

	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer func() { _ = p.Stop(&kernel.Context{}) }()

	waitForServer(t, port, "/health")

	tests := []struct {
		name       string
		path       string
		expected   int
	}{
		{".git/config returns 404", "/.git/config", http.StatusNotFound},
		{".git/HEAD returns 404", "/.git/HEAD", http.StatusNotFound},
		{".git//objects returns 404", "/.git//objects", http.StatusNotFound},
		{"/not-git unaffected", "/not-git", http.StatusOK}, // catch-all handler serves home
		{"/api/version unaffected", "/api/version", http.StatusOK},
		{"root unaffected", "/", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get("http://localhost:" + port + tt.path)
			if err != nil {
				t.Fatalf("GET %s: %v", tt.path, err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.expected {
				t.Errorf("GET %s: expected status %d, got %d", tt.path, tt.expected, resp.StatusCode)
			}
		})
	}
}

func TestHealthAuth(t *testing.T) {
	t.Run("no token set — health is public", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)

		port := freePort(t)
		p := proxy.New(proxy.Options{
			Port:        port,
			Kernel:      k,
			Logger:      logger,
			HealthToken: "",
		})

		if err := p.Start(&kernel.Context{}); err != nil {
			t.Fatalf("failed to start proxy: %v", err)
		}
		defer func() { _ = p.Stop(&kernel.Context{}) }()

		waitForServer(t, port, "/health")

		resp, err := http.Get("http://localhost:" + port + "/health")
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 without token when HEALTH_TOKEN is empty, got %d", resp.StatusCode)
		}
	})

	t.Run("token set, no header — returns 401", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)

		port := freePort(t)
		p := proxy.New(proxy.Options{
			Port:        port,
			Kernel:      k,
			Logger:      logger,
			HealthToken: "secret123",
		})

		if err := p.Start(&kernel.Context{}); err != nil {
			t.Fatalf("failed to start proxy: %v", err)
		}
		defer func() { _ = p.Stop(&kernel.Context{}) }()

		waitForServer(t, port, "/")

		resp, err := http.Get("http://localhost:" + port + "/health")
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "unauthorized") {
			t.Fatalf("expected body to contain 'unauthorized', got %s", string(body))
		}
	})

	t.Run("token set, wrong token — returns 401", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)

		port := freePort(t)
		p := proxy.New(proxy.Options{
			Port:        port,
			Kernel:      k,
			Logger:      logger,
			HealthToken: "secret123",
		})

		if err := p.Start(&kernel.Context{}); err != nil {
			t.Fatalf("failed to start proxy: %v", err)
		}
		defer func() { _ = p.Stop(&kernel.Context{}) }()

		waitForServer(t, port, "/")

		req, _ := http.NewRequest("GET", "http://localhost:"+port+"/health", nil)
		req.Header.Set("Authorization", "Bearer wrong-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 with wrong token, got %d", resp.StatusCode)
		}
	})

	t.Run("token set, correct token — returns 200", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)

		port := freePort(t)
		p := proxy.New(proxy.Options{
			Port:        port,
			Kernel:      k,
			Logger:      logger,
			HealthToken: "secret123",
		})

		if err := p.Start(&kernel.Context{}); err != nil {
			t.Fatalf("failed to start proxy: %v", err)
		}
		defer func() { _ = p.Stop(&kernel.Context{}) }()

		waitForServer(t, port, "/")

		req, _ := http.NewRequest("GET", "http://localhost:"+port+"/health", nil)
		req.Header.Set("Authorization", "Bearer secret123")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 with correct token, got %d", resp.StatusCode)
		}
	})

	t.Run("token set, other routes unaffected", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)

		port := freePort(t)
		p := proxy.New(proxy.Options{
			Port:        port,
			Kernel:      k,
			Logger:      logger,
			HealthToken: "secret123",
		})

		if err := p.Start(&kernel.Context{}); err != nil {
			t.Fatalf("failed to start proxy: %v", err)
		}
		defer func() { _ = p.Stop(&kernel.Context{}) }()

		waitForServer(t, port, "/")

		resp, err := http.Get("http://localhost:" + port + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on / without token, got %d", resp.StatusCode)
		}
	})
}

func freePort(t testing.TB) string {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	port := fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port)
	_ = l.Close()
	return port
}

func waitForServer(t testing.TB, port, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:" + port + path)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server on port %s did not respond within 3s", port)
}
