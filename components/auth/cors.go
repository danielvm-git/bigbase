package auth

import (
	"net/http"
	"strings"
)

// CORS returns middleware that adds CORS headers for allowed origins.
// If allowedOrigins is nil or empty, only same-origin requests are allowed
// (Origin empty or matching the request Host / PublicURL) — default closed
// for cross-origin (CWE-942).
// Allowed origins are matched exactly against the request's Origin header.
// Preflight OPTIONS requests receive 204 with appropriate headers.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	isAllowed := func(origin string) bool {
		for _, o := range allowedOrigins {
			if o == origin {
				return true
			}
		}
		return false
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Same-origin / non-browser: no Origin header → pass through.
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if len(allowedOrigins) == 0 {
				// Default closed for cross-origin; allow same-origin only.
				if !isSameOriginRequest(r, origin) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"error":"origin not allowed"}`))
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			if !isAllowed(origin) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"origin not allowed"}`))
				return
			}

			// Handle preflight.
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.Header().Set("Vary", "Origin")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Regular request: add CORS response headers.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			next.ServeHTTP(w, r)
		})
	}
}

// isSameOriginRequest reports whether origin matches the request's own host
// (scheme://host[:port]), so empty CORS allowlists still permit the admin UI.
func isSameOriginRequest(r *http.Request, origin string) bool {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return origin == scheme+"://"+r.Host
}
