package proxy

import (
	"net/http"
	"strings"
)

const (
	// strictCSP is for API and JSON routes.
	// Explicit script-src and connect-src are added alongside default-src for
	// defense-in-depth and auditability, even though default-src 'self' already
	// covers them as the fallback.
	strictCSP = "default-src 'self'; script-src 'self'; connect-src 'self'"

	// permissiveCSP is for HTML routes (home, docs, admin) that need inline styles
	// and Google Fonts. unsafe-inline in style-src is a deliberate per-route choice:
	// these routes serve static HTML pages where inline styles are used for
	// server-rendered content. API, health, and all other routes use the strict
	// policy above, so the XSS surface from unsafe-inline is limited.
	permissiveCSP = "default-src 'self'; script-src 'self'; connect-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com"

	// restrictivePermissionsPolicy disables browser features that are not needed.
	restrictivePermissionsPolicy = "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()"
)

// securityHeadersMiddleware adds standard security headers to every response.
func (p *Proxy) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		csp := strictCSP
		if r.URL.Path == "/" || r.URL.Path == "/docs" || r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/") {
			csp = permissiveCSP
		}

		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", restrictivePermissionsPolicy)

		// Set Cache-Control: no-store, no-cache, must-revalidate for API and health
		// endpoints to prevent sensitive response caching (CWE-525).
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		}

		next.ServeHTTP(w, r)
	})
}
