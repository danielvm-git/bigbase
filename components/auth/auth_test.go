package auth_test

import (
	"context"
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

func setupAuth(t *testing.T) (*auth.Auth, http.Handler, http.Handler) {
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

	return a, a.Handler(), a.ProtectedHandler()
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

func TestAuthConfigSchema(t *testing.T) {
	a := &auth.Auth{}
	if got := a.ConfigSchema(); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestAuthHooks(t *testing.T) {
	a := &auth.Auth{}
	if got := a.Hooks(); len(got) != 0 {
		t.Fatalf("expected empty hooks, got %v", got)
	}
}

func TestAuthVersion(t *testing.T) {
	a := &auth.Auth{}
	if got := a.Version(); got == "" {
		t.Fatal("expected non-empty version")
	}
}

func TestNewWithNilLogger(t *testing.T) {
	d := db.New(db.Options{Path: ":memory:", Logger: testLogger{}})
	a := auth.New(auth.Options{DB: d, Logger: nil, Secret: "test-secret-32-chars!!!"})
	k := kernel.New(testLogger{})
	k.Register(a)
	k.Register(d)
	if err := k.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
}

func TestUserIDFromContextMissing(t *testing.T) {
	if _, ok := auth.UserIDFromContext(context.Background()); ok {
		t.Fatal("expected false for missing user id")
	}
}

func TestUserEmailFromContextMissing(t *testing.T) {
	if _, ok := auth.UserEmailFromContext(context.Background()); ok {
		t.Fatal("expected false for missing email")
	}
}

func TestRegisterWrongMethod(t *testing.T) {
	_, handler, _ := setupAuth(t)

	req := httptest.NewRequest("GET", "/api/auth/register", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestLoginWrongMethod(t *testing.T) {
	_, handler, _ := setupAuth(t)

	req := httptest.NewRequest("PUT", "/api/auth/login", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestRegister(t *testing.T) {
	_, handler, _ := setupAuth(t)

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
	_, handler, _ := setupAuth(t)

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
	_, handler, _ := setupAuth(t)

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
	_, handler, _ := setupAuth(t)

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
	_, handler, _ := setupAuth(t)

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
	_, handler, _ := setupAuth(t)

	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"nobody@test.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterCaseInsensitiveEmail(t *testing.T) {
	_, handler, _ := setupAuth(t)

	body := `{"email":"Alice@Test.Com","password":"secret123"}`
	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w.Body.Bytes())
	user := resp["user"].(map[string]any)
	if user["email"] != "alice@test.com" {
		t.Fatalf("expected lowercased email 'alice@test.com', got '%v'", user["email"])
	}

	dupBody := `{"email":"alice@test.com","password":"secret123"}`
	dupReq := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(dupBody))
	dupReq.Header.Set("Content-Type", "application/json")
	dupW := httptest.NewRecorder()
	handler.ServeHTTP(dupW, dupReq)
	if dupW.Code != http.StatusConflict {
		t.Fatalf("expected 409 for case-insensitive duplicate, got %d", dupW.Code)
	}
}

func TestRegisterEmptyBody(t *testing.T) {
	_, handler, _ := setupAuth(t)

	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"a@b.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing password, got %d", w.Code)
	}
}

func TestMiddlewareValidToken(t *testing.T) {
	a, h, _ := setupAuth(t)

	regReq := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"eve@test.com","password":"secret123"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	h.ServeHTTP(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("registration failed: %d %s", regW.Code, regW.Body.String())
	}

	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"eve@test.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w.Body.Bytes())
	token, _ := resp["token"].(string)

	var called bool
	protected := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := auth.UserIDFromContext(r.Context()); !ok {
			t.Fatal("expected UserID in context")
		}
		if _, ok := auth.UserEmailFromContext(r.Context()); !ok {
			t.Fatal("expected UserEmail in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

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
	a, _, _ := setupAuth(t)

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

func TestMiddlewareEmptyBearerPrefix(t *testing.T) {
	a, _, _ := setupAuth(t)

	protected := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListUsers(t *testing.T) {
	_, handler, protected := setupAuth(t)

	for i, email := range []string{"u1@test.com", "u2@test.com", "u3@test.com"} {
		req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"`+email+`","password":"secret123"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("register %d: expected 201, got %d", i, w.Code)
		}
	}

	loginBody := `{"email":"u1@test.com","password":"secret123"}`
	loginReq := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginReq)
	resp := parseResponse(t, loginW.Body.Bytes())
	token, _ := resp["token"].(string)

	req := httptest.NewRequest("GET", "/api/auth/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string][]map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	users := body["data"]
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}
}

func TestListUsersUnauthenticated(t *testing.T) {
	_, _, protected := setupAuth(t)

	req := httptest.NewRequest("GET", "/api/auth/users", nil)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDeleteUser(t *testing.T) {
	_, handler, protected := setupAuth(t)

	for _, email := range []string{"del1@test.com", "del2@test.com"} {
		req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"`+email+`","password":"secret123"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	loginReq := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"del1@test.com","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginReq)
	resp := parseResponse(t, loginW.Body.Bytes())
	token, _ := resp["token"].(string)

	delReq := httptest.NewRequest("DELETE", "/api/auth/users/2", nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delW := httptest.NewRecorder()
	protected.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", delW.Code, delW.Body.String())
	}

	listReq := httptest.NewRequest("GET", "/api/auth/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	protected.ServeHTTP(listW, listReq)
	var listResp map[string][]map[string]any
	_ = json.NewDecoder(listW.Body).Decode(&listResp)
	if len(listResp["data"]) != 1 {
		t.Fatalf("expected 1 user after delete, got %d", len(listResp["data"]))
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	_, handler, protected := setupAuth(t)

	regReq := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"del@test.com","password":"secret123"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	handler.ServeHTTP(regW, regReq)

	loginReq := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"del@test.com","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginReq)
	resp := parseResponse(t, loginW.Body.Bytes())
	token, _ := resp["token"].(string)

	delReq := httptest.NewRequest("DELETE", "/api/auth/users/999", nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delW := httptest.NewRecorder()
	protected.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", delW.Code, delW.Body.String())
	}
}

func TestDeleteUserWrongMethod(t *testing.T) {
	_, handler, protected := setupAuth(t)

	regReq := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"wm@test.com","password":"secret123"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	handler.ServeHTTP(regW, regReq)

	loginReq := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"wm@test.com","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginReq)
	resp := parseResponse(t, loginW.Body.Bytes())
	token, _ := resp["token"].(string)

	req := httptest.NewRequest("GET", "/api/auth/users/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestMiddlewareBadToken(t *testing.T) {
	a, _, _ := setupAuth(t)

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
