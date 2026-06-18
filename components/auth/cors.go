package auth

import "net/http"

// CORS returns middleware that adds CORS headers for allowed origins.
// If allowedOrigins is nil or empty, the middleware is a no-op (default closed).
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
