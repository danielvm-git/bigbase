package auth

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	// Integration test: CORS via auth.CORSMiddleware().
	// Tests that Options.CORSAllowedOrigins flows through to the middleware.
	a := &Auth{corsAllowedOrigins: []string{"https://example.com"}}
	cors := a.CORSMiddleware()

	handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/api/auth/me", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("expected Allow-Origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSOptions(t *testing.T) {
	// Verify CORSAllowedOrigins is stored from Options.
	tdb := &testDB{}
	opts := Options{
		DB:                 tdb,
		Logger:             noopLogger{},
		Secret:             "test-secret",
		CORSAllowedOrigins: []string{"https://a.example.com", "https://b.example.com"},
	}
	a := New(opts)
	if len(a.corsAllowedOrigins) != 2 {
		t.Errorf("expected 2 origins, got %d", len(a.corsAllowedOrigins))
	}
	if a.corsAllowedOrigins[0] != "https://a.example.com" || a.corsAllowedOrigins[1] != "https://b.example.com" {
		t.Errorf("unexpected origins: %v", a.corsAllowedOrigins)
	}
	// Default: no origins means nil.
	def := New(Options{DB: tdb, Logger: noopLogger{}, Secret: "test"})
	if len(def.corsAllowedOrigins) != 0 {
		t.Errorf("expected 0 origins for default, got %d", len(def.corsAllowedOrigins))
	}
}

// testDB is a minimal DBer for unit tests that don't need database access.
type testDB struct{}

func (d *testDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}
func (d *testDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}
func (d *testDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}
func (d *testDB) Exec(query string, args ...any) (sql.Result, error) {
	return nil, nil
}
func (d *testDB) Query(query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}
func (d *testDB) QueryRow(query string, args ...any) *sql.Row {
	return nil
}
func (d *testDB) Migrate(migration string) error {
	return nil
}

func TestCORSDefaultClosed(t *testing.T) {
	// When no origins configured, CORS middleware is a no-op.
	// Requests pass through unchanged, no Access-Control-* headers set.
	var allowedOrigins []string
	cors := CORS(allowedOrigins)

	handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
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

func TestCORSDenied(t *testing.T) {
	allowedOrigins := []string{"https://my-spa.example.com"}

	// Preflight from disallowed origin → 403.
	t.Run("preflight from disallowed origin", func(t *testing.T) {
		cors := CORS(allowedOrigins)
		handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("OPTIONS", "/api/auth/me", nil)
		req.Header.Set("Origin", "https://evil.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("expected no CORS headers on denied preflight")
		}
	})

	// Non-preflight request from disallowed origin → 403.
	t.Run("non-preflight from disallowed origin", func(t *testing.T) {
		cors := CORS(allowedOrigins)
		handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		req.Header.Set("Origin", "https://evil.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("expected no CORS headers on denied request")
		}
	})

	// Allowed origin GET passes through with CORS headers.
	t.Run("non-preflight from allowed origin", func(t *testing.T) {
		cors := CORS(allowedOrigins)
		handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		req.Header.Set("Origin", "https://my-spa.example.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "https://my-spa.example.com" {
			t.Errorf("expected Allow-Origin %q, got %q", "https://my-spa.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
		}
		if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("expected Access-Control-Allow-Credentials: true")
		}
	})
}
