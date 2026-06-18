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

func TestCORSPreflightAllowed(t *testing.T) {
	// When origin is in allowlist, OPTIONS preflight returns 204 with correct headers.
	allowedOrigins := []string{"https://my-spa.example.com"}
	cors := CORS(allowedOrigins)

	handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/api/auth/me", nil)
	req.Header.Set("Origin", "https://my-spa.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://my-spa.example.com" {
		t.Errorf("expected Allow-Origin %q, got %q", "https://my-spa.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
	if rec.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}
	if rec.Header().Get("Access-Control-Max-Age") == "" {
		t.Error("expected Access-Control-Max-Age header")
	}
	body := rec.Body.String()
	if body != "" {
		t.Errorf("expected empty body for 204, got %q", body)
	}
}
