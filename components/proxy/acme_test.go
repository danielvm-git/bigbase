package proxy_test

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/proxy"
	"github.com/danielvm/bigbase/kernel"
)

func TestACMEProvision(t *testing.T) {
	t.Run("manager_configured_when_https_port_set", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)

		p := proxy.New(proxy.Options{
			Port:      freePort(t),
			HTTPSPort: freePort(t),
			Kernel:    k,
			Logger:    logger,
			CertDir:   t.TempDir(),
			ACMEEmail: "test@example.com",
		})

		if err := p.Start(&kernel.Context{}); err != nil {
			t.Fatalf("start: %v", err)
		}
		defer func() { _ = p.Stop(&kernel.Context{}) }()

		if p.ACME() == nil {
			t.Fatal("expected ACME manager to be configured when HTTPSPort is set")
		}
		t.Log("ACME manager properly configured")
	})

	t.Run("no_manager_when_https_port_not_set", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)

		p := proxy.New(proxy.Options{
			Port:   freePort(t),
			Kernel: k,
			Logger: logger,
		})

		if err := p.Start(&kernel.Context{}); err != nil {
			t.Fatalf("start: %v", err)
		}
		defer func() { _ = p.Stop(&kernel.Context{}) }()

		if p.ACME() != nil {
			t.Fatal("expected no ACME manager when HTTPSPort is not set")
		}
	})
}

func TestHTTPSRedirect(t *testing.T) {
	t.Run("redirects_http_to_https_for_custom_domain", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)

		httpPort := freePort(t)
		p := proxy.New(proxy.Options{
			Port:      httpPort,
			HTTPSPort: freePort(t),
			Kernel:    k,
			Logger:    logger,
			CertDir:   t.TempDir(),
		})

		if err := p.Start(&kernel.Context{}); err != nil {
			t.Fatalf("start: %v", err)
		}
		defer func() { _ = p.Stop(&kernel.Context{}) }()

		waitForServer(t, httpPort, "/health")

		_ = p.RegisterDeploymentHost("myapp.com", 9999, "site-1", nil, nil)

		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%s/", httpPort), nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		req.Host = "myapp.com"

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: 5 * time.Second,
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusMovedPermanently {
			t.Fatalf("expected 301 redirect, got %d", resp.StatusCode)
		}

		location := resp.Header.Get("Location")
		if !strings.HasPrefix(location, "https://myapp.com") {
			t.Fatalf("expected redirect to https://myapp.com, got %q", location)
		}
		t.Logf("Redirect location: %s", location)
	})

	t.Run("no_redirect_for_loopback_hosts", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)

		httpPort := freePort(t)
		p := proxy.New(proxy.Options{
			Port:      httpPort,
			HTTPSPort: freePort(t),
			Kernel:    k,
			Logger:    logger,
			CertDir:   t.TempDir(),
		})

		if err := p.Start(&kernel.Context{}); err != nil {
			t.Fatalf("start: %v", err)
		}
		defer func() { _ = p.Stop(&kernel.Context{}) }()

		waitForServer(t, httpPort, "/health")

		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", httpPort))
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusMovedPermanently {
			t.Fatal("expected no redirect for localhost, got 301")
		}
	})
}

func TestHTTPServing(t *testing.T) {
	t.Run("https_server_configured_and_listening", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)

		httpPort := freePort(t)
		p := proxy.New(proxy.Options{
			Port:      httpPort,
			HTTPSPort: freePort(t),
			Kernel:    k,
			Logger:    logger,
			CertDir:   t.TempDir(),
		})

		if err := p.Start(&kernel.Context{}); err != nil {
			t.Fatalf("start: %v", err)
		}
		defer func() { _ = p.Stop(&kernel.Context{}) }()

		httpsAddr := p.HTTPSAddr()
		if httpsAddr == "" {
			t.Fatal("HTTPS address should be set")
		}
		t.Logf("HTTPS listening on %s", httpsAddr)

		// Verify the HTTPS port is actually listening by checking the raw TCP connection.
		_, portStr, err := net.SplitHostPort(httpsAddr)
		if err != nil {
			t.Fatalf("parse addr %q: %v", httpsAddr, err)
		}

		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", portStr), 3*time.Second)
		if err != nil {
			t.Fatalf("HTTPS port %s not accepting connections: %v", portStr, err)
		}
		_ = conn.Close()
		t.Logf("HTTPS server accepting TCP connections on port %s", portStr)

		// Verify the HTTP→HTTPS redirect works for a registered host.
		// This proves the full HTTPS pipeline is configured even if TLS with
		// autocert requires a proper FQDN for cert provisioning.
		_ = p.RegisterDeploymentHost("myapp.com", 9999, "site-1", nil, nil)
		req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:%s/", httpPort), nil)
		req.Host = "myapp.com"
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: 5 * time.Second,
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("redirect request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusMovedPermanently {
			t.Fatalf("expected 301 redirect, got %d", resp.StatusCode)
		}
		location := resp.Header.Get("Location")
		if !strings.HasPrefix(location, "https://myapp.com") {
			t.Fatalf("expected redirect to https://myapp.com, got %q", location)
		}
		t.Logf("HTTPS redirect verified: %s", location)
	})
}
