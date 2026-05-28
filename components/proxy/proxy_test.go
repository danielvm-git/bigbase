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
	if !strings.Contains(bodyStr, "0.1.0") {
		t.Fatal("expected page to show version '0.1.0'")
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
