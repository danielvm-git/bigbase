package proxy

import "net/http"

const (
	// strictCSP is for API and JSON routes.
	strictCSP = "default-src 'self'"

	// permissiveCSP is for HTML routes (home, docs) that need inline styles and Google Fonts.
	permissiveCSP = "default-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com"
)

// securityHeadersMiddleware adds standard security headers to every response.
func (p *Proxy) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		csp := strictCSP
		if r.URL.Path == "/" || r.URL.Path == "/docs" {
			csp = permissiveCSP
		}

		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
