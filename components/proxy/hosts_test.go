package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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

	if err := p.RegisterDeploymentHost("myapp.bigbase.click", backendPort); err != nil {
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

	if err := p.RegisterDeploymentHost("test-unreg.bigbase.click", 19999); err != nil {
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

	if err := p.RegisterDeploymentHost("myapp.bigbase.click", 9999); err != nil {
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
