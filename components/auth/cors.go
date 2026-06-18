package auth

import (
	"net/http"
)

// CORS returns middleware that adds CORS headers for allowed origins.
// If allowedOrigins is nil or empty, the middleware is a no-op (default closed).
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

			// Default closed: no origins configured → pass through.
			if len(allowedOrigins) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Origin header required for CORS processing.
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Deny disallowed origins.
			if !isAllowed(origin) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"origin not allowed"}`))
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


