package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupAuth(t *testing.T) (*auth.Auth, http.Handler) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})

	k.Register(a)
	k.Register(d)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	return a, a.Handler()
}

func parseResponse(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	return resp
}

func TestAuthImplementsComponent(t *testing.T) {
	var _ kernel.Component = &auth.Auth{}
}

func TestAuthName(t *testing.T) {
	a := &auth.Auth{}
	if got := a.Name(); got != "auth" {
		t.Fatalf("expected Name()='auth', got '%s'", got)
	}
}

func TestRegister(t *testing.T) {
	_, handler := setupAuth(t)

	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"alice@test.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w.Body.Bytes())
	if _, ok := resp["token"]; !ok {
		t.Fatalf("expected token in response, got: %v", resp)
	}
	user, ok := resp["user"].(map[string]any)
	if !ok {
		t.Fatalf("expected user object, got: %v", resp)
	}
	if user["email"] != "alice@test.com" {
		t.Fatalf("expected email 'alice@test.com', got: %v", user)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	_, handler := setupAuth(t)

	body := `{"email":"bob@test.com","password":"secret123"}`
	req1 := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestRegisterMissingFields(t *testing.T) {
	_, handler := setupAuth(t)

	tests := []struct {
		name string
		body string
	}{
		{"empty email", `{"email":"","password":"secret123"}`},
		{"empty password", `{"email":"a@b.com","password":""}`},
		{"short password", `{"email":"a@b.com","password":"abc"}`},
		{"malformed json", `not json`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestLogin(t *testing.T) {
	_, handler := setupAuth(t)

	reg := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"carol@test.com","password":"secret123"}`))
	reg.Header.Set("Content-Type", "application/json")
	w0 := httptest.NewRecorder()
	handler.ServeHTTP(w0, reg)

	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"carol@test.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w.Body.Bytes())
	if _, ok := resp["token"]; !ok {
		t.Fatalf("expected token in response, got: %v", resp)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	_, handler := setupAuth(t)

	reg := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"dave@test.com","password":"secret123"}`))
	reg.Header.Set("Content-Type", "application/json")
	w0 := httptest.NewRecorder()
	handler.ServeHTTP(w0, reg)

	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"dave@test.com","password":"wrongpass"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoginUnknownUser(t *testing.T) {
	_, handler := setupAuth(t)

	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"nobody@test.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMiddlewareValidToken(t *testing.T) {
	a, h := setupAuth(t)

	var called bool
	protected := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if uid := r.Header.Get("X-User-ID"); uid == "" {
			t.Fatal("expected X-User-ID header")
		}
		if email := r.Header.Get("X-User-Email"); email == "" {
			t.Fatal("expected X-User-Email header")
		}
		w.WriteHeader(http.StatusOK)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"eve@test.com","password":"secret123"}`)))

	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"eve@test.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	resp := parseResponse(t, w.Body.Bytes())
	token, _ := resp["token"].(string)

	authReq := httptest.NewRequest("GET", "/api/protected", nil)
	authReq.Header.Set("Authorization", "Bearer "+token)
	authW := httptest.NewRecorder()
	protected.ServeHTTP(authW, authReq)

	if !called {
		t.Fatal("expected middleware to call next handler")
	}
	if authW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", authW.Code)
	}
}

func TestMiddlewareNoToken(t *testing.T) {
	a, _ := setupAuth(t)

	protected := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest("GET", "/api/protected", nil)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMiddlewareBadToken(t *testing.T) {
	a, _ := setupAuth(t)

	protected := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer badtoken")
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
