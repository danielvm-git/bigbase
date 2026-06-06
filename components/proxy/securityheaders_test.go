package proxy_test

import (
	"net/http"
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
		t.Fatalf("start proxy: %v", err)
	}
	defer func() { _ = p.Stop(&kernel.Context{}) }()

	waitForServer(t, port, "/health")

	routes := []string{"/", "/health", "/docs", "/api/version"}

	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			resp, err := http.Get("http://localhost:" + port + route)
			if err != nil {
				t.Fatalf("GET %s: %v", route, err)
			}
			defer func() { _ = resp.Body.Close() }()

			checks := []struct {
				header string
				want   string
			}{
				{"Content-Security-Policy", "default-src 'self'"},
				{"Strict-Transport-Security", "max-age=63072000; includeSubDomains"},
				{"X-Frame-Options", "DENY"},
				{"X-Content-Type-Options", "nosniff"},
				{"Referrer-Policy", "strict-origin-when-cross-origin"},
			}

			for _, c := range checks {
				got := resp.Header.Get(c.header)
				if got != c.want {
					t.Errorf("%s: %s = %q, want %q", route, c.header, got, c.want)
				}
			}
		})
	}
}
