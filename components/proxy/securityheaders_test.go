package proxy_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/proxy"
	"github.com/danielvm/bigbase/kernel"
)

func TestSecurityHeaders(t *testing.T) {
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

	t.Run("home page and docs receive permissive CSP for styles and fonts", func(t *testing.T) {
		paths := []string{"/", "/docs"}
		for _, path := range paths {
			resp, err := http.Get("http://localhost:" + port + path)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			defer func() { _ = resp.Body.Close() }()

			csp := resp.Header.Get("Content-Security-Policy")
			if !strings.Contains(csp, "'unsafe-inline'") {
				t.Errorf("expected CSP for %s to contain 'unsafe-inline', got %q", path, csp)
			}
			if !strings.Contains(csp, "fonts.googleapis.com") {
				t.Errorf("expected CSP for %s to contain fonts.googleapis.com, got %q", path, csp)
			}
			if !strings.Contains(csp, "fonts.gstatic.com") {
				t.Errorf("expected CSP for %s to contain fonts.gstatic.com, got %q", path, csp)
			}
		}
	})

	t.Run("home page contains style block", func(t *testing.T) {
		resp, err := http.Get("http://localhost:" + port + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "<style>") {
			t.Error("expected home page to contain <style> tag")
		}
	})

	t.Run("API routes receive strict CSP", func(t *testing.T) {
		routes := []string{"/health", "/api/version"}
		for _, path := range routes {
			resp, err := http.Get("http://localhost:" + port + path)
			if err != nil {
				t.Fatalf("GET %s: %v", path, err)
			}
			defer func() { _ = resp.Body.Close() }()

			csp := resp.Header.Get("Content-Security-Policy")
			if csp != "default-src 'self'" {
				t.Errorf("expected strict CSP for %s, got %q", path, csp)
			}
		}
	})

	t.Run("standard security headers are present", func(t *testing.T) {
		resp, err := http.Get("http://localhost:" + port + "/health")
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		headers := map[string]string{
			"Strict-Transport-Security": "max-age=63072000; includeSubDomains",
			"X-Frame-Options":           "DENY",
			"X-Content-Type-Options":    "nosniff",
			"Referrer-Policy":           "strict-origin-when-cross-origin",
		}

		for k, v := range headers {
			if got := resp.Header.Get(k); got != v {
				t.Errorf("expected header %s: %q, got %q", k, v, got)
			}
		}
	})

	t.Run("Permissions-Policy header is present with restrictive defaults", func(t *testing.T) {
		resp, err := http.Get("http://localhost:" + port + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		pp := resp.Header.Get("Permissions-Policy")
		if pp == "" {
			t.Fatal("expected Permissions-Policy header")
		}
		if !strings.Contains(pp, "accelerometer=()") {
			t.Errorf("expected Permissions-Policy to restrict accelerometer, got %q", pp)
		}
		if !strings.Contains(pp, "camera=()") {
			t.Errorf("expected Permissions-Policy to restrict camera, got %q", pp)
		}
		if !strings.Contains(pp, "geolocation=()") {
			t.Errorf("expected Permissions-Policy to restrict geolocation, got %q", pp)
		}
		if !strings.Contains(pp, "microphone=()") {
			t.Errorf("expected Permissions-Policy to restrict microphone, got %q", pp)
		}
	})

	t.Run("Cache-Control no-store on health endpoint", func(t *testing.T) {
		resp, err := http.Get("http://localhost:" + port + "/health")
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		cc := resp.Header.Get("Cache-Control")
		if cc != "no-store" {
			t.Errorf("expected Cache-Control: no-store on /health, got %q", cc)
		}
	})

	t.Run("Cache-Control no-store on API routes", func(t *testing.T) {
		resp, err := http.Get("http://localhost:" + port + "/api/version")
		if err != nil {
			t.Fatalf("GET /api/version: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		cc := resp.Header.Get("Cache-Control")
		if cc != "no-store" {
			t.Errorf("expected Cache-Control: no-store on /api/version, got %q", cc)
		}
	})

	t.Run("Cache-Control absent on home page", func(t *testing.T) {
		resp, err := http.Get("http://localhost:" + port + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		cc := resp.Header.Get("Cache-Control")
		if cc != "" {
			t.Errorf("expected no Cache-Control on /, got %q", cc)
		}
	})
}
