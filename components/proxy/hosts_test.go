package proxy_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/proxy"
	"github.com/danielvm/bigbase/kernel"
)

func TestProxyDeploymentHost(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("deployed-app"))
	}))
	t.Cleanup(backend.Close)

	u, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatalf("parse backend url: %v", err)
	}
	backendPort, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("backend port: %v", err)
	}

	logger := testLogger{}
	k := kernel.New(logger)
	port := freePort(t)
	p := proxy.New(proxy.Options{Port: port, Kernel: k, Logger: logger})
	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	t.Cleanup(func() { _ = p.Stop(&kernel.Context{}) })

	waitForServer(t, port, "/health")

	if err := p.RegisterDeploymentHost("myapp.bigbase.click", backendPort, "s1"); err != nil {
		t.Fatalf("register host: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+port+"/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = "myapp.bigbase.click"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	if string(body) != "deployed-app" {
		t.Fatalf("body = %q", body)
	}
}

func TestProxySiteIDMapping(t *testing.T) {
	logger := testLogger{}
	p := proxy.New(proxy.Options{Logger: logger})
	
	host := "site-1.bigbase.click"
	siteID := "s123"
	port := 12345
	
	// This will fail to compile first
	if err := p.RegisterDeploymentHost(host, port, siteID); err != nil {
		t.Fatalf("register failed: %v", err)
	}
}

func TestProxyUnregisterDeploymentHost(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	port := freePort(t)
	p := proxy.New(proxy.Options{Port: port, Kernel: k, Logger: logger})
	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	t.Cleanup(func() { _ = p.Stop(&kernel.Context{}) })

	waitForServer(t, port, "/health")

	if err := p.RegisterDeploymentHost("test-unreg.bigbase.click", 19999, "s-unreg"); err != nil {
		t.Fatalf("register host: %v", err)
	}

	p.UnregisterDeploymentHost("test-unreg.bigbase.click")

	denyURL := "http://127.0.0.1:" + port + "/api/internal/caddy-allow?domain=test-unreg.bigbase.click"
	resp, err := http.Get(denyURL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("after unregister status = %d, want 403", resp.StatusCode)
	}
}

func TestRegisterDeploymentHostReplacesExisting(t *testing.T) {
	logger := testLogger{}
	p := proxy.New(proxy.Options{Logger: logger})

	host := "replace-test.bigbase.click"

	// First registration — sets host -> 10001
	if err := p.RegisterDeploymentHost(host, 10001, "s1"); err != nil {
		t.Fatalf("first registration: %v", err)
	}

	// Second registration with different port — must replace, not error
	if err := p.RegisterDeploymentHost(host, 20002, "s2"); err != nil {
		t.Fatalf("re-registration with new port should succeed, got: %v", err)
	}

	// Verify replacement via GetDeploymentHostInfo (port 20002 not 10001)
	info, ok := p.GetDeploymentHostInfo(host)
	if !ok {
		t.Fatal("host should still be registered after re-registration")
	}
	if info.Port != 20002 {
		t.Fatalf("after re-registration: port=%d (want 20002)", info.Port)
	}
	if info.SiteID != "s2" {
		t.Fatalf("after re-registration: site_id=%q (want s2)", info.SiteID)
	}
}

func TestCaddyAllow(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	port := freePort(t)
	p := proxy.New(proxy.Options{Port: port, Kernel: k, Logger: logger})
	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	t.Cleanup(func() { _ = p.Stop(&kernel.Context{}) })

	waitForServer(t, port, "/health")
	base := "http://127.0.0.1:" + port + "/api/internal/caddy-allow"

	denyURL := base + "?domain=unknown.bigbase.click"
	resp, err := http.Get(denyURL)
	if err != nil {
		t.Fatalf("deny request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unregistered domain status = %d, want 403", resp.StatusCode)
	}

	if err := p.RegisterDeploymentHost("myapp.bigbase.click", 9999, "s-myapp"); err != nil {
		t.Fatalf("register host: %v", err)
	}

	allowURL := base + "?domain=myapp.bigbase.click"
	resp, err = http.Get(allowURL)
	if err != nil {
		t.Fatalf("allow request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("registered domain status = %d, want 200", resp.StatusCode)
	}
}

func TestCaddyAllowMCPHost(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	port := freePort(t)
	p := proxy.New(proxy.Options{Port: port, Kernel: k, Logger: logger})
	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	t.Cleanup(func() { _ = p.Stop(&kernel.Context{}) })

	waitForServer(t, port, "/health")

	// mcp.bigbase.click is an allowed host (for MCP TLS certs) even without a
	// registered deployment.
	allowURL := fmt.Sprintf("http://127.0.0.1:%s/api/internal/caddy-allow?domain=mcp.bigbase.click", port)
	resp, err := http.Get(allowURL)
	if err != nil {
		t.Fatalf("mcp host request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mcp.bigbase.click status = %d, want 200", resp.StatusCode)
	}
}

func TestMCPDiscoveryEndpoint(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	port := freePort(t)
	p := proxy.New(proxy.Options{Port: port, Kernel: k, Logger: logger})
	if err := p.Start(&kernel.Context{}); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	t.Cleanup(func() { _ = p.Stop(&kernel.Context{}) })

	waitForServer(t, port, "/health")

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/.well-known/mcp.json", port))
	if err != nil {
		t.Fatalf("GET /.well-known/mcp.json: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if !strings.Contains(string(body), "mcpServers") {
		t.Errorf("expected mcpServers in body, got: %s", string(body))
	}
	if !strings.Contains(string(body), "bigbase") {
		t.Errorf("expected bigbase in body, got: %s", string(body))
	}
	if !strings.Contains(string(body), "mcp.bigbase.click") {
		t.Errorf("expected mcp.bigbase.click in body, got: %s", string(body))
	}
	if !strings.Contains(string(body), "/mcp") {
		t.Errorf("expected /mcp path in body, got: %s", string(body))
	}
	t.Logf("MCP discovery: %s", string(body))
}
