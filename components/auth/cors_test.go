package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSDefaultClosed(t *testing.T) {
	// When no origins configured, CORS middleware is a no-op.
	// Requests pass through unchanged, no Access-Control-* headers set.
	var allowedOrigins []string
	cors := CORS(allowedOrigins)

	handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.Header.Set("Origin", "https://any-spa.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no Access-Control-Allow-Origin header, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	body := rec.Body.String()
	if body != `{"ok":true}` {
		t.Errorf("expected body %q, got %q", `{"ok":true}`, body)
	}
}
