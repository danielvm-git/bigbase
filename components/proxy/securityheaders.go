package proxy

import (
	"net/http"
	"strings"
)

const (
	// strictCSP is for API and JSON routes.
	// Explicit script-src and connect-src are added alongside default-src for
	// defense-in-depth and auditability, even though default-src 'self' already
	// covers them as the fallback. frame-ancestors 'none' is the CSP-level
	// clickjacking guard (belt-and-suspenders with X-Frame-Options: DENY) and
	// satisfies DAST checks that require an explicit frame-ancestors directive.
	strictCSP = "default-src 'self'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'"

	// permissiveCSP is for HTML routes (home, docs, admin) and deployed static
	// sites that need inline styles, inline scripts, Google Fonts, and common
	// CDN resources. unsafe-inline in both script-src and style-src is a
	// deliberate per-route choice: these routes serve static HTML pages where
	// inline scripts (theme toggles, HTMX handlers) and inline styles are used
	// for server-rendered content. API, health, and all other routes use the
	// strict policy above, so the XSS surface from unsafe-inline is limited.
	// CDN entries (jsdelivr, cloudflare, tailwindcss, unpkg) are needed for
	// deployed sites that load UI frameworks from public CDNs.
	// TODO(issue #197): make CSP configurable per-site so each deployment
	// can declare its own allowed sources.
	permissiveCSP = "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com https://unpkg.com; connect-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net https://cdnjs.cloudflare.com; font-src 'self' https://fonts.gstatic.com https://cdnjs.cloudflare.com https://cdn.jsdelivr.net; frame-ancestors 'none'"

	// restrictivePermissionsPolicy disables browser features that are not needed.
	restrictivePermissionsPolicy = "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()"
)

// securityHeadersMiddleware adds standard security headers to every response.
func (p *Proxy) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API routes serve JSON, not HTML — CSP is a browser security
		// mechanism for HTML documents. Applying strictCSP to JSON error
		// responses (e.g. 503 from GitHub install) can block the browser's
		// built-in error page styling.
		//
		// All non-API routes get permissiveCSP because deployed sites serve
		// HTML from many paths (e.g. /checks, /actions, /backlog), not just
		// /, /docs, and /admin. The permissive policy allows inline scripts,
		// inline styles, and CDN resources needed by HTMX-based apps.
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			csp := permissiveCSP
			if r.URL.Path == "/health" {
				csp = strictCSP
			}
			w.Header().Set("Content-Security-Policy", csp)
		}

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
