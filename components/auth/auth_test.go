package auth_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

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
	a.SetOTPStore(auth.NewMapOTPStore())
	a.SetRateLimitStore(auth.NewMapRateLimitStore())
	a.SetLoginLockoutStore(auth.NewMapLoginLockoutStore())

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

func TestJWTCarriesOrgID(t *testing.T) {
	_, handler, _ := setupAuth(t)

	// Register creates a user + personal org; verify the JWT carries the org_id.
	req := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"orguser@test.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w.Body.Bytes())
	tokenStr, ok := resp["token"].(string)
	if !ok {
		t.Fatal("expected token in register response")
	}

	// Decode JWT to inspect claims (don't need signature verify — we trust our own server)
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("parse jwt: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}
	orgID, hasOrgID := claims["org_id"]
	if !hasOrgID {
		t.Fatal("JWT claims missing org_id")
	}
	if orgID.(float64) == 0 {
		t.Fatal("expected non-zero org_id in JWT claims")
	}

	// Also verify login returns a token with org_id
	loginReq := httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"email":"orguser@test.com","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, loginReq)
	if w2.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	resp2 := parseResponse(t, w2.Body.Bytes())
	tokenStr2, ok := resp2["token"].(string)
	if !ok {
		t.Fatal("expected token in login response")
	}

	token2, _, err := new(jwt.Parser).ParseUnverified(tokenStr2, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("parse login jwt: %v", err)
	}
	claims2, ok := token2.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}
	orgID2, hasOrgID2 := claims2["org_id"]
	if !hasOrgID2 {
		t.Fatal("login JWT claims missing org_id")
	}
	if orgID2.(float64) == 0 {
		t.Fatal("expected non-zero org_id in login JWT claims")
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

func TestMiddlewareSetsOrgID(t *testing.T) {
	a, h, _ := setupAuth(t)

	// Register user (creates personal org)
	regReq := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"orgmid@test.com","password":"secret123"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	h.ServeHTTP(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", regW.Code, regW.Body.String())
	}

	// Login to get token
	loginReq := httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"email":"orgmid@test.com","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	h.ServeHTTP(loginW, loginReq)
	resp := parseResponse(t, loginW.Body.Bytes())
	token, _ := resp["token"].(string)

	// Middleware should set OrgID in context
	var gotOrgID int64
	var gotOK bool
	protected := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrgID, gotOK = auth.OrgIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	authReq := httptest.NewRequest("GET", "/api/protected", nil)
	authReq.Header.Set("Authorization", "Bearer "+token)
	authW := httptest.NewRecorder()
	protected.ServeHTTP(authW, authReq)

	if authW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", authW.Code)
	}
	if !gotOK {
		t.Fatal("expected OrgIDFromContext to return true")
	}
	if gotOrgID == 0 {
		t.Fatal("expected non-zero org_id in context")
	}
}

func TestMiddlewareRejectsNoOrg(t *testing.T) {
	a, _, _ := setupAuth(t)

	// Forge a JWT with org_id=0 using the same secret as setupAuth.
	claims := auth.Claims{}
	claims.UserID = 999
	claims.Email = "noorganization@test.com"
	claims.Role = "user"
	claims.OrgID = 0 // explicitly zero
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(1 * time.Hour))
	claims.IssuedAt = jwt.NewNumericDate(time.Now())
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("test-secret-32-chars!!!"))
	if err != nil {
		t.Fatalf("sign forged token: %v", err)
	}

	protected := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler with org_id=0")
	}))

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "no organization" {
		t.Fatalf("expected 'no organization' error, got %q", body["error"])
	}
}

func TestAnonymousToken(t *testing.T) {
	a, h, _ := setupAuthWithGoogle(t)

	// Mint an anonymous token.
	anonReq := httptest.NewRequest("POST", "/api/auth/anonymous", nil)
	anonW := httptest.NewRecorder()
	h.ServeHTTP(anonW, anonReq)
	if anonW.Code != http.StatusOK {
		t.Fatalf("mint anonymous token: expected 200, got %d: %s", anonW.Code, anonW.Body.String())
	}
	resp := parseResponse(t, anonW.Body.Bytes())
	token, _ := resp["token"].(string)
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if _, hasRefresh := resp["refresh_token"]; hasRefresh {
		t.Fatal("anonymous token must not include refresh_token")
	}

	// Build a protected handler that reports the context values set by Middleware.
	protected := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := auth.UserRoleFromContext(r.Context())
		userID, _ := auth.UserIDFromContext(r.Context())
		orgID, _ := auth.OrgIDFromContext(r.Context())
		email, _ := auth.UserEmailFromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"role":%q,"user_id":%d,"org_id":%d,"email":%q}`, role, userID, orgID, email)
	}))

	tests := []struct {
		name       string
		method     string
		token      string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "read endpoint accepts anonymous token",
			method:     "GET",
			token:      token,
			wantStatus: http.StatusOK,
			wantBody:   `"anonymous"`,
		},
		{
			name:       "POST rejected for anonymous",
			method:     "POST",
			token:      token,
			wantStatus: http.StatusForbidden,
			wantBody:   "write access not allowed",
		},
		{
			name:       "PATCH rejected for anonymous",
			method:     "PATCH",
			token:      token,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "DELETE rejected for anonymous",
			method:     "DELETE",
			token:      token,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no token rejected",
			method:     "GET",
			token:      "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "expired token rejected",
			method:     "GET",
			token:      makeExpiredAnonymousToken(t, a),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "tampered token rejected",
			method:     "GET",
			token:      token + "x",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/protected", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			w := httptest.NewRecorder()
			protected.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Fatalf("expected body to contain %q, got %q", tt.wantBody, w.Body.String())
			}
		})
	}
}

// makeExpiredAnonymousToken returns an anonymous token that has already expired.
func makeExpiredAnonymousToken(t *testing.T, a *auth.Auth) string {
	t.Helper()
	claims := auth.Claims{
		UserID: 0,
		Email:  "",
		Role:   "anonymous",
		OrgID:  0,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("test-secret-32-chars!!!"))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	return signed
}

func TestListUsers(t *testing.T) {
	_, handler, protected := setupAuth(t)

	for i, email := range []string{"u1@test.com"} {
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
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
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

func TestListUsersRequiresAdmin(t *testing.T) {
	_, handler, protected := setupAuth(t)

	// Register first user (admin)
	regReq := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"admin@test.com","password":"secret123"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	handler.ServeHTTP(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("register admin: expected 201, got %d: %s", regW.Code, regW.Body.String())
	}

	// Register second user (non-admin)
	regReq2 := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"user@test.com","password":"secret123"}`))
	regReq2.Header.Set("Content-Type", "application/json")
	regW2 := httptest.NewRecorder()
	handler.ServeHTTP(regW2, regReq2)
	if regW2.Code != http.StatusCreated {
		t.Fatalf("register user: expected 201, got %d: %s", regW2.Code, regW2.Body.String())
	}

	// Login as non-admin user
	loginReq := httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"email":"user@test.com","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", loginW.Code, loginW.Body.String())
	}
	resp := parseResponse(t, loginW.Body.Bytes())
	token, _ := resp["token"].(string)
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Try GET /api/auth/users as non-admin — should be 403
	req := httptest.NewRequest("GET", "/api/auth/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin user, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != "forbidden" {
		t.Fatalf("expected 'forbidden' error, got %q", body["error"])
	}
}

func TestListUsersForbiddenForNonAdmin(t *testing.T) {
	_, handler, protected := setupAuth(t)

	// Register first user (admin) and second user (non-admin)
	for _, email := range []string{"alpha@test.com", "beta@test.com"} {
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"`+email+`","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)
		if regW.Code != http.StatusCreated {
			t.Fatalf("register %s: expected 201, got %d: %s", email, regW.Code, regW.Body.String())
		}
	}

	// Login as non-admin (beta) — should get 403 and no user list
	loginReq := httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"email":"beta@test.com","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusOK {
		t.Fatalf("login beta: expected 200, got %d: %s", loginW.Code, loginW.Body.String())
	}
	resp := parseResponse(t, loginW.Body.Bytes())
	nonAdminToken, _ := resp["token"].(string)
	if nonAdminToken == "" {
		t.Fatal("expected non-empty token for beta")
	}

	// Non-admin should get 403
	req := httptest.NewRequest("GET", "/api/auth/users", nil)
	req.Header.Set("Authorization", "Bearer "+nonAdminToken)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("non-admin: expected 403, got %d: %s", w.Code, w.Body.String())
	}

	// Verify response body contains no user data
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, hasData := body["data"]; hasData {
		t.Fatal("non-admin response should not contain 'data' field with user list")
	}
	if body["error"] != "forbidden" {
		t.Fatalf("expected 'forbidden' error, got %v", body["error"])
	}

	// Login as admin (alpha) — should get 200 with full user list
	loginAdmin := httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"email":"alpha@test.com","password":"secret123"}`))
	loginAdmin.Header.Set("Content-Type", "application/json")
	loginAdminW := httptest.NewRecorder()
	handler.ServeHTTP(loginAdminW, loginAdmin)
	if loginAdminW.Code != http.StatusOK {
		t.Fatalf("login alpha: expected 200, got %d: %s", loginAdminW.Code, loginAdminW.Body.String())
	}
	adminResp := parseResponse(t, loginAdminW.Body.Bytes())
	adminToken, _ := adminResp["token"].(string)

	req2 := httptest.NewRequest("GET", "/api/auth/users", nil)
	req2.Header.Set("Authorization", "Bearer "+adminToken)
	w2 := httptest.NewRecorder()
	protected.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("admin: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var adminBody map[string][]map[string]any
	if err := json.NewDecoder(w2.Body).Decode(&adminBody); err != nil {
		t.Fatalf("decode admin response: %v", err)
	}
	users := adminBody["data"]
	if len(users) != 1 {
		t.Fatalf("expected 1 user in admin response, got %d", len(users))
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

type mockGoogleVerifier struct {
	user *auth.GoogleUser
	err  error
}

func (m *mockGoogleVerifier) Verify(ctx context.Context, code string) (*auth.GoogleUser, error) {
	return m.user, m.err
}

func setupAuthWithGoogle(t *testing.T) (*auth.Auth, http.Handler, http.Handler) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{
		DB:                 d,
		Logger:             logger,
		Secret:             "test-secret-32-chars!!!",
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
	})
	a.SetOTPStore(auth.NewMapOTPStore())
	a.SetRateLimitStore(auth.NewMapRateLimitStore())
	a.SetLoginLockoutStore(auth.NewMapLoginLockoutStore())

	k.Register(a)
	k.Register(d)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	a.SetGoogleVerifier(&mockGoogleVerifier{
		user: &auth.GoogleUser{GoogleID: "google-123", Email: "oauth@test.com", Name: "OAuth User", Avatar: ""},
	})

	return a, a.Handler(), a.ProtectedHandler()
}

func setupAuthWithPublicURL(t *testing.T, publicURL string) (*auth.Auth, http.Handler, http.Handler) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{
		DB:                 d,
		Logger:             logger,
		Secret:             "test-secret-32-chars!!!",
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
		PublicURL:          publicURL,
	})
	a.SetOTPStore(auth.NewMapOTPStore())
	a.SetRateLimitStore(auth.NewMapRateLimitStore())
	a.SetLoginLockoutStore(auth.NewMapLoginLockoutStore())

	k.Register(a)
	k.Register(d)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	a.SetGoogleVerifier(&mockGoogleVerifier{
		user: &auth.GoogleUser{GoogleID: "google-123", Email: "oauth@test.com", Name: "OAuth User", Avatar: ""},
	})

	return a, a.Handler(), a.ProtectedHandler()
}

// makeValidOAuthCallback creates a callback request with a valid oauth_state cookie.
func makeValidOAuthCallback(t *testing.T, code, state string) *http.Request {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/auth/oauth/google/callback?code="+code+"&state="+state, nil)
	req.AddCookie(&http.Cookie{
		Name:  "oauth_state",
		Value: auth.SignState(state, []byte("test-secret-32-chars!!!")),
	})
	return req
}

func TestGoogleCallbackCreatesUser(t *testing.T) {
	a, handler, _ := setupAuthWithGoogle(t)

	req := makeValidOAuthCallback(t, "testcode", "test-state")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d: %s", w.Code, w.Body.String())
	}

	loc := w.Header().Get("Location")
	if loc != "/admin/" {
		t.Fatalf("expected redirect to /admin/, got %q", loc)
	}

	cookies := w.Result().Cookies()
	var tokenCookie string
	for _, c := range cookies {
		if c.Name == "token" {
			tokenCookie = c.Value
			break
		}
	}
	if tokenCookie == "" {
		t.Fatal("expected token cookie in response")
	}

	claims, err := a.ValidateToken(tokenCookie)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if claims.Email != "oauth@test.com" {
		t.Fatalf("expected email 'oauth@test.com', got %q", claims.Email)
	}
}

func TestGoogleCallbackLinksExistingUser(t *testing.T) {
	a, handler, _ := setupAuthWithGoogle(t)

	regReq := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"oauth@test.com","password":"secret123"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	handler.ServeHTTP(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("register: %d", regW.Code)
	}

	req := makeValidOAuthCallback(t, "testcode", "test-state")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d: %s", w.Code, w.Body.String())
	}

	cookies := w.Result().Cookies()
	var tokenCookie string
	for _, c := range cookies {
		if c.Name == "token" {
			tokenCookie = c.Value
			break
		}
	}
	if tokenCookie == "" {
		t.Fatal("expected token cookie")
	}

	claims, err := a.ValidateToken(tokenCookie)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if claims.Email != "oauth@test.com" {
		t.Fatalf("expected email 'oauth@test.com', got %q", claims.Email)
	}
}

func TestOAuthStateCookieSet(t *testing.T) {
	_, handler, _ := setupAuthWithGoogle(t)

	req := httptest.NewRequest("GET", "/api/auth/oauth/google", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", w.Code)
	}

	cookies := w.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "oauth_state" {
			stateCookie = c
			break
		}
	}
	if stateCookie == nil {
		t.Fatal("expected oauth_state cookie to be set")
	}
	if stateCookie.Value == "" {
		t.Fatal("expected non-empty oauth_state cookie value")
	}
	if !stateCookie.HttpOnly {
		t.Error("expected oauth_state cookie to be HttpOnly")
	}
	if stateCookie.Path != "/" {
		t.Errorf("expected Path '/', got %q", stateCookie.Path)
	}
	if stateCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax, got %v", stateCookie.SameSite)
	}
	if stateCookie.MaxAge != 600 {
		t.Errorf("expected MaxAge=600, got %d", stateCookie.MaxAge)
	}
}

func TestOAuthStateMismatchRejected(t *testing.T) {
	_, handler, _ := setupAuthWithGoogle(t)

	// Callback without oauth_state cookie should return 400
	req := httptest.NewRequest("GET", "/api/auth/oauth/google/callback?code=testcode&state=abc123", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing state cookie, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["error"] != "invalid state" {
		t.Fatalf("expected 'invalid state' error, got %v", resp["error"])
	}

	// Verify no token cookie was set
	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "token" && c.Value != "" {
			t.Fatal("expected no token cookie when state validation fails")
		}
	}
}

func TestOAuthStateClearedOnSuccess(t *testing.T) {
	_, handler, _ := setupAuthWithGoogle(t)

	req := makeValidOAuthCallback(t, "testcode", "test-state")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect on success, got %d: %s", w.Code, w.Body.String())
	}

	// Verify oauth_state cookie is cleared (MaxAge <= 0 or expired in past)
	setCookies := w.Result().Header["Set-Cookie"]
	if len(setCookies) == 0 {
		t.Fatal("expected Set-Cookie headers")
	}
	var stateCookieCleared bool
	for _, sc := range setCookies {
		if strings.HasPrefix(sc, "oauth_state=") {
			// Should be cleared: MaxAge=0, MaxAge=-1, or Expires in the past
			stateCookieCleared = strings.Contains(sc, "Max-Age=0") ||
				strings.Contains(sc, "Max-Age=-1") ||
				strings.Contains(sc, "expires=")
		}
	}
	if !stateCookieCleared {
		t.Fatal("expected oauth_state cookie to be cleared (MaxAge<=0)")
	}
}

func TestOAuthCallbackNoStateCookie(t *testing.T) {
	_, handler, _ := setupAuthWithGoogle(t)

	t.Run("no_cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/oauth/google/callback?code=valid&state=somestate", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
		// Verify no token cookie was set
		for _, c := range w.Result().Cookies() {
			if c.Name == "token" && c.Value != "" {
				t.Fatal("expected no token cookie when state validation fails")
			}
		}
	})

	t.Run("wrong_state_in_cookie", func(t *testing.T) {
		// Sign a different state than what's in the query
		req := httptest.NewRequest("GET", "/api/auth/oauth/google/callback?code=valid&state=real-state", nil)
		req.AddCookie(&http.Cookie{
			Name:  "oauth_state",
			Value: auth.SignState("wrong-state", []byte("test-secret-32-chars!!!")),
		})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for mismatched state, got %d: %s", w.Code, w.Body.String())
		}
		for _, c := range w.Result().Cookies() {
			if c.Name == "token" && c.Value != "" {
				t.Fatal("expected no token cookie when state validation fails")
			}
		}
	})
}

func TestGoogleOAuthDisabledWhenNoConfig(t *testing.T) {
	_, handler, _ := setupAuth(t)

	req := httptest.NewRequest("GET", "/api/auth/oauth/google", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when google auth not configured, got %d", w.Code)
	}
}

func TestOAuthPublicURL(t *testing.T) {
	// Setup with PublicURL configured
	a, h, _ := setupAuthWithPublicURL(t, "https://bigbase.click")

	req := httptest.NewRequest("GET", "/api/auth/oauth/google", nil)
	req.Host = "evil.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", w.Code)
	}
	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("expected Location header")
	}
	// Parse and decode the redirect_uri query parameter
	parsed, _ := url.Parse(location)
	redirectURI := parsed.Query().Get("redirect_uri")
	if !strings.Contains(redirectURI, "https://bigbase.click") {
		t.Fatalf("expected redirect_uri to use PublicURL (bigbase.click), got: %s", redirectURI)
	}
	if strings.Contains(redirectURI, "evil.com") {
		t.Fatal("redirect_uri must not use Host header when PublicURL is set")
	}

	got := a.PublicURLOrDefault(req)
	if got != "https://bigbase.click" {
		t.Fatalf("expected PublicURLOrDefault to return PublicURL, got %q", got)
	}
}

func TestOAuthPublicURLOrDefaultIgnoresHostHeader(t *testing.T) {
	// CWE-601: When publicURL is configured, the Host header must be ignored entirely.
	a, h, _ := setupAuthWithPublicURL(t, "https://bigbase.click")

	req := httptest.NewRequest("GET", "/api/auth/oauth/google", nil)
	req.Host = "evil.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", w.Code)
	}
	location := w.Header().Get("Location")
	parsed, _ := url.Parse(location)
	redirectURI := parsed.Query().Get("redirect_uri")
	if !strings.Contains(redirectURI, "https://bigbase.click") {
		t.Fatalf("expected redirect_uri to use configured PublicURL, got: %s", redirectURI)
	}
	if strings.Contains(redirectURI, "evil.com") {
		t.Fatal("redirect_uri must not use attacker-controlled Host header")
	}

	got := a.PublicURLOrDefault(req)
	if got != "https://bigbase.click" {
		t.Fatalf("expected PublicURLOrDefault to return PublicURL, got %q", got)
	}
}

func TestHandleMe(t *testing.T) {
	_, handler, protected := setupAuth(t)

	regReq := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"me@test.com","password":"secret123"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	handler.ServeHTTP(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("register: %d", regW.Code)
	}

	loginReq := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"me@test.com","password":"secret123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginReq)
	resp := parseResponse(t, loginW.Body.Bytes())
	token, _ := resp["token"].(string)
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["email"] != "me@test.com" {
		t.Fatalf("expected email 'me@test.com', got '%v'", body["email"])
	}
	if _, ok := body["id"]; !ok {
		t.Fatal("expected id in response")
	}
}

func TestHandleMeUnauthenticated(t *testing.T) {
	_, _, protected := setupAuth(t)

	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestOrganization(t *testing.T) {
	t.Run("orgs_table_migration", func(t *testing.T) {
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

		// Verify orgs table exists with correct schema
		var nameCol string
		err := d.QueryRowContext(context.Background(),
			`SELECT name FROM sqlite_master WHERE type='table' AND name='orgs'`,
		).Scan(&nameCol)
		if err != nil {
			t.Fatalf("orgs table not found: %v", err)
		}

		rows, err := d.QueryContext(context.Background(), "PRAGMA table_info('orgs')")
		if err != nil {
			t.Fatalf("pragma orgs: %v", err)
		}
		defer func() { _ = rows.Close() }()

		columns := map[string]bool{}
		for rows.Next() {
			var cid int
			var name, coltype string
			var notnull, pk int
			var defaultVal *string
			if err := rows.Scan(&cid, &name, &coltype, &notnull, &defaultVal, &pk); err != nil {
				t.Fatalf("scan column: %v", err)
			}
			columns[name] = true
		}

		for _, col := range []string{"id", "name", "slug", "owner_id", "created_at", "updated_at"} {
			if !columns[col] {
				t.Errorf("expected column %q in orgs table, but not found", col)
			}
		}

		// Verify users.default_org_id column exists
		rows2, err := d.QueryContext(context.Background(), "PRAGMA table_info('users')")
		if err != nil {
			t.Fatalf("pragma users: %v", err)
		}
		defer func() { _ = rows2.Close() }()

		hasDefaultOrgID := false
		for rows2.Next() {
			var cid int
			var name, coltype string
			var notnull, pk int
			var defaultVal *string
			if err := rows2.Scan(&cid, &name, &coltype, &notnull, &defaultVal, &pk); err != nil {
				t.Fatalf("scan column: %v", err)
			}
			if name == "default_org_id" {
				hasDefaultOrgID = true
			}
		}
		if !hasDefaultOrgID {
			t.Error("expected default_org_id column in users table")
		}
	})

	t.Run("personal_org_created_on_register", func(t *testing.T) {
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

		handler := a.Handler()

		// Register a user
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"alice@test.com","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)

		if regW.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", regW.Code, regW.Body.String())
		}

		// Verify a personal org was created for the user
		var orgID int64
		var orgName, orgSlug string
		var orgOwnerID int64
		err := d.QueryRowContext(context.Background(),
			"SELECT id, name, slug, owner_id FROM orgs WHERE slug = ?",
			"alice@test.com",
		).Scan(&orgID, &orgName, &orgSlug, &orgOwnerID)
		if err != nil {
			t.Fatalf("personal org not created: %v", err)
		}

		if orgName != "alice@test.com" {
			t.Errorf("expected org name 'alice@test.com', got %q", orgName)
		}

		// Verify user's default_org_id is set
		var userID int64
		var defaultOrgID sql.NullInt64
		err = d.QueryRowContext(context.Background(),
			"SELECT id, default_org_id FROM users WHERE email = ?",
			"alice@test.com",
		).Scan(&userID, &defaultOrgID)
		if err != nil {
			t.Fatalf("user not found: %v", err)
		}

		if !defaultOrgID.Valid || defaultOrgID.Int64 != orgID {
			t.Errorf("expected default_org_id=%d, got %v", orgID, defaultOrgID)
		}

		if orgOwnerID != userID {
			t.Errorf("expected org owner_id=%d, got %d", userID, orgOwnerID)
		}
	})

	t.Run("personal_org_created_on_google_oauth", func(t *testing.T) {
		logger := testLogger{}
		k := kernel.New(logger)

		d := db.New(db.Options{Path: ":memory:", Logger: logger})
		a := auth.New(auth.Options{
			DB:                 d,
			Logger:             logger,
			Secret:             "test-secret-32-chars!!!",
			GoogleClientID:     "test-client-id",
			GoogleClientSecret: "test-client-secret",
		})

		k.Register(a)
		k.Register(d)

		if err := k.Start(); err != nil {
			t.Fatalf("kernel start: %v", err)
		}
		t.Cleanup(func() { _ = k.Stop() })

		a.SetGoogleVerifier(&mockGoogleVerifier{
			user: &auth.GoogleUser{GoogleID: "google-456", Email: "oauth@test.com", Name: "OAuth User", Avatar: ""},
		})

		handler := a.Handler()

		// Simulate Google OAuth callback with valid state cookie
		state := "test-state"
		signed := auth.SignState(state, []byte("test-secret-32-chars!!!"))
		req := httptest.NewRequest("GET", "/api/auth/oauth/google/callback?code=testcode&state="+state, nil)
		req.AddCookie(&http.Cookie{Name: "oauth_state", Value: signed})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusFound {
			t.Fatalf("expected 302 redirect, got %d: %s", w.Code, w.Body.String())
		}

		// Verify a personal org was created for the OAuth user
		var orgID int64
		var orgName string
		err := d.QueryRowContext(context.Background(),
			"SELECT id, name FROM orgs WHERE slug = ?",
			"oauth@test.com",
		).Scan(&orgID, &orgName)
		if err != nil {
			t.Fatalf("personal org not created for OAuth user: %v", err)
		}

		if orgName != "oauth@test.com" {
			t.Errorf("expected org name 'oauth@test.com', got %q", orgName)
		}

		// Verify user's default_org_id is set
		var defaultOrgID sql.NullInt64
		err = d.QueryRowContext(context.Background(),
			"SELECT default_org_id FROM users WHERE email = ?",
			"oauth@test.com",
		).Scan(&defaultOrgID)
		if err != nil {
			t.Fatalf("user not found: %v", err)
		}

		if !defaultOrgID.Valid || defaultOrgID.Int64 != orgID {
			t.Errorf("expected default_org_id=%d, got %v", orgID, defaultOrgID)
		}
	})

	t.Run("org_store", func(t *testing.T) {
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

		ctx := context.Background()

		// Test CreateOrg
		org, err := a.CreateOrg(ctx, "Test Org", "test-org", 1)
		if err != nil {
			t.Fatalf("CreateOrg: %v", err)
		}

		if org.ID == 0 {
			t.Error("expected non-zero org ID")
		}
		if org.Name != "Test Org" {
			t.Errorf("expected name 'Test Org', got %q", org.Name)
		}
		if org.Slug != "test-org" {
			t.Errorf("expected slug 'test-org', got %q", org.Slug)
		}
		if org.OwnerID != 1 {
			t.Errorf("expected owner_id 1, got %d", org.OwnerID)
		}
		if org.CreatedAt == "" {
			t.Error("expected non-empty created_at")
		}
		if org.UpdatedAt == "" {
			t.Error("expected non-empty updated_at")
		}

		// Test slug uniqueness
		_, err = a.CreateOrg(ctx, "Duplicate", "test-org", 2)
		if err == nil {
			t.Error("expected error for duplicate slug")
		}

		// Test GetOrgByID
		fetched, err := a.GetOrgByID(ctx, org.ID)
		if err != nil {
			t.Fatalf("GetOrgByID: %v", err)
		}
		if fetched == nil {
			t.Fatal("expected org, got nil")
		}
		if fetched.Name != org.Name {
			t.Errorf("expected name %q, got %q", org.Name, fetched.Name)
		}

		// Test GetOrgByID not found
		notFound, err := a.GetOrgByID(ctx, 99999)
		if err != nil {
			t.Fatalf("GetOrgByID: %v", err)
		}
		if notFound != nil {
			t.Error("expected nil for non-existent org")
		}

		// Test ListOrgsByOwner
		_, _ = a.CreateOrg(ctx, "Org 2", "org-2", 1)
		_, _ = a.CreateOrg(ctx, "Org 3", "org-3", 1)

		orgs, err := a.ListOrgsByOwner(ctx, 1)
		if err != nil {
			t.Fatalf("ListOrgsByOwner: %v", err)
		}
		if len(orgs) != 3 {
			t.Errorf("expected 3 orgs, got %d", len(orgs))
		}

		// List orgs for owner with no orgs
		empty, err := a.ListOrgsByOwner(ctx, 99999)
		if err != nil {
			t.Fatalf("ListOrgsByOwner: %v", err)
		}
		if len(empty) != 0 {
			t.Errorf("expected 0 orgs, got %d", len(empty))
		}

		// Test UpdateOrg
		updated, err := a.UpdateOrg(ctx, org.ID, "Updated Org", "updated-org")
		if err != nil {
			t.Fatalf("UpdateOrg: %v", err)
		}
		if updated.Name != "Updated Org" {
			t.Errorf("expected name 'Updated Org', got %q", updated.Name)
		}
		if updated.Slug != "updated-org" {
			t.Errorf("expected slug 'updated-org', got %q", updated.Slug)
		}

		// Update to existing slug should fail
		_, err = a.UpdateOrg(ctx, org.ID, "Collision", "org-2")
		if err == nil {
			t.Error("expected error updating to existing slug")
		}

		// Test DeleteOrg
		if err := a.DeleteOrg(ctx, org.ID); err != nil {
			t.Fatalf("DeleteOrg: %v", err)
		}

		// Verify deleted
		deleted, err := a.GetOrgByID(ctx, org.ID)
		if err != nil {
			t.Fatalf("GetOrgByID after delete: %v", err)
		}
		if deleted != nil {
			t.Error("expected nil after delete")
		}

		// Delete non-existent org should not error
		if err := a.DeleteOrg(ctx, 99999); err != nil {
			t.Errorf("DeleteOrg non-existent: %v", err)
		}
	})

	t.Run("org_handlers", func(t *testing.T) {
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

		handler := a.Handler()
		protected := a.ProtectedHandler()

		// Register and login
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"org@test.com","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)
		if regW.Code != http.StatusCreated {
			t.Fatalf("register: expected 201, got %d: %s", regW.Code, regW.Body.String())
		}

		loginReq := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"email":"org@test.com","password":"secret123"}`))
		loginReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()
		handler.ServeHTTP(loginW, loginReq)
		resp := parseResponse(t, loginW.Body.Bytes())
		token, _ := resp["token"].(string)
		if token == "" {
			t.Fatal("expected non-empty token")
		}

		// POST /api/orgs — create a new org
		createReq := httptest.NewRequest("POST", "/api/orgs",
			strings.NewReader(`{"name":"My New Org","slug":"my-new-org"}`))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+token)
		createW := httptest.NewRecorder()
		protected.ServeHTTP(createW, createReq)

		if createW.Code != http.StatusCreated {
			t.Fatalf("create org: expected 201, got %d: %s", createW.Code, createW.Body.String())
		}

		var createResp map[string]any
		if err := json.NewDecoder(createW.Body).Decode(&createResp); err != nil {
			t.Fatalf("decode create: %v", err)
		}
		orgData, ok := createResp["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data in response, got %v", createResp)
		}
		if orgData["name"] != "My New Org" {
			t.Errorf("expected name 'My New Org', got %v", orgData["name"])
		}

		// POST /api/orgs — duplicate slug should fail
		dupReq := httptest.NewRequest("POST", "/api/orgs",
			strings.NewReader(`{"name":"Duplicate","slug":"my-new-org"}`))
		dupReq.Header.Set("Content-Type", "application/json")
		dupReq.Header.Set("Authorization", "Bearer "+token)
		dupW := httptest.NewRecorder()
		protected.ServeHTTP(dupW, dupReq)
		if dupW.Code != http.StatusConflict {
			t.Errorf("duplicate slug: expected 409, got %d: %s", dupW.Code, dupW.Body.String())
		}

		// POST with invalid slug (contains spaces)
		badSlugReq := httptest.NewRequest("POST", "/api/orgs",
			strings.NewReader(`{"name":"Bad Slug","slug":"my bad slug"}`))
		badSlugReq.Header.Set("Content-Type", "application/json")
		badSlugReq.Header.Set("Authorization", "Bearer "+token)
		badSlugW := httptest.NewRecorder()
		protected.ServeHTTP(badSlugW, badSlugReq)
		if badSlugW.Code != http.StatusBadRequest {
			t.Errorf("bad slug: expected 400, got %d: %s", badSlugW.Code, badSlugW.Body.String())
		}

		// POST with invalid slug (uppercase)
		upperSlugReq := httptest.NewRequest("POST", "/api/orgs",
			strings.NewReader(`{"name":"Upper Slug","slug":"UPPER-SLUG"}`))
		upperSlugReq.Header.Set("Content-Type", "application/json")
		upperSlugReq.Header.Set("Authorization", "Bearer "+token)
		upperSlugW := httptest.NewRecorder()
		protected.ServeHTTP(upperSlugW, upperSlugReq)
		if upperSlugW.Code != http.StatusBadRequest {
			t.Errorf("uppercase slug: expected 400, got %d: %s", upperSlugW.Code, upperSlugW.Body.String())
		}

		// GET /api/orgs — list orgs
		listReq := httptest.NewRequest("GET", "/api/orgs", nil)
		listReq.Header.Set("Authorization", "Bearer "+token)
		listW := httptest.NewRecorder()
		protected.ServeHTTP(listW, listReq)

		if listW.Code != http.StatusOK {
			t.Fatalf("list orgs: expected 200, got %d: %s", listW.Code, listW.Body.String())
		}

		var listResp map[string]any
		if err := json.NewDecoder(listW.Body).Decode(&listResp); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		data, ok := listResp["data"].([]any)
		if !ok {
			t.Fatalf("expected data array, got %v", listResp)
		}
		// Should include the personal org + the new org
		if len(data) < 2 {
			t.Errorf("expected at least 2 orgs (personal + created), got %d", len(data))
		}

		// Get the new org's ID from the list
		var newOrgID float64
		for _, item := range data {
			org := item.(map[string]any)
			if org["slug"] == "my-new-org" {
				newOrgID = org["id"].(float64)
			}
		}
		if newOrgID == 0 {
			t.Fatal("could not find created org in list")
		}

		// GET /api/orgs/{id}
		getReq := httptest.NewRequest("GET", "/api/orgs/"+fmt.Sprintf("%.0f", newOrgID), nil)
		getReq.Header.Set("Authorization", "Bearer "+token)
		getW := httptest.NewRecorder()
		protected.ServeHTTP(getW, getReq)

		if getW.Code != http.StatusOK {
			t.Fatalf("get org: expected 200, got %d: %s", getW.Code, getW.Body.String())
		}

		var getResp map[string]any
		if err := json.NewDecoder(getW.Body).Decode(&getResp); err != nil {
			t.Fatalf("decode get: %v", err)
		}
		gotOrg := getResp["data"].(map[string]any)
		if gotOrg["name"] != "My New Org" {
			t.Errorf("expected 'My New Org', got %v", gotOrg["name"])
		}

		// PATCH /api/orgs/{id} — update org
		patchReq := httptest.NewRequest("PATCH", "/api/orgs/"+fmt.Sprintf("%.0f", newOrgID),
			strings.NewReader(`{"name":"Updated Org","slug":"updated-org"}`))
		patchReq.Header.Set("Content-Type", "application/json")
		patchReq.Header.Set("Authorization", "Bearer "+token)
		patchW := httptest.NewRecorder()
		protected.ServeHTTP(patchW, patchReq)

		if patchW.Code != http.StatusOK {
			t.Fatalf("patch org: expected 200, got %d: %s", patchW.Code, patchW.Body.String())
		}

		var patchResp map[string]any
		if err := json.NewDecoder(patchW.Body).Decode(&patchResp); err != nil {
			t.Fatalf("decode patch: %v", err)
		}
		patched := patchResp["data"].(map[string]any)
		if patched["name"] != "Updated Org" {
			t.Errorf("expected 'Updated Org', got %v", patched["name"])
		}

		// PATCH with empty body — should reject
		emptyPatch := httptest.NewRequest("PATCH", "/api/orgs/"+fmt.Sprintf("%.0f", newOrgID),
			strings.NewReader(`{}`))
		emptyPatch.Header.Set("Content-Type", "application/json")
		emptyPatch.Header.Set("Authorization", "Bearer "+token)
		emptyPatchW := httptest.NewRecorder()
		protected.ServeHTTP(emptyPatchW, emptyPatch)
		if emptyPatchW.Code != http.StatusBadRequest {
			t.Errorf("empty patch body: expected 400, got %d: %s", emptyPatchW.Code, emptyPatchW.Body.String())
		}

		// PATCH with invalid json — should reject
		badPatch := httptest.NewRequest("PATCH", "/api/orgs/"+fmt.Sprintf("%.0f", newOrgID),
			strings.NewReader(`not json`))
		badPatch.Header.Set("Content-Type", "application/json")
		badPatch.Header.Set("Authorization", "Bearer "+token)
		badPatchW := httptest.NewRecorder()
		protected.ServeHTTP(badPatchW, badPatch)
		if badPatchW.Code != http.StatusBadRequest {
			t.Errorf("bad json patch: expected 400, got %d", badPatchW.Code)
		}

		// PATCH with invalid slug
		badSlugPatch := httptest.NewRequest("PATCH", "/api/orgs/"+fmt.Sprintf("%.0f", newOrgID),
			strings.NewReader(`{"slug":"bad slug"}`))
		badSlugPatch.Header.Set("Content-Type", "application/json")
		badSlugPatch.Header.Set("Authorization", "Bearer "+token)
		badSlugPatchW := httptest.NewRecorder()
		protected.ServeHTTP(badSlugPatchW, badSlugPatch)
		if badSlugPatchW.Code != http.StatusBadRequest {
			t.Errorf("bad slug patch: expected 400, got %d", badSlugPatchW.Code)
		}

		// PATCH — non-owner should get 403
		// Register another user
		reg2Req := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"other@test.com","password":"secret123"}`))
		reg2Req.Header.Set("Content-Type", "application/json")
		reg2W := httptest.NewRecorder()
		handler.ServeHTTP(reg2W, reg2Req)

		login2Req := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"email":"other@test.com","password":"secret123"}`))
		login2Req.Header.Set("Content-Type", "application/json")
		login2W := httptest.NewRecorder()
		handler.ServeHTTP(login2W, login2Req)
		resp2 := parseResponse(t, login2W.Body.Bytes())
		token2, _ := resp2["token"].(string)

		nonOwnerPatch := httptest.NewRequest("PATCH", "/api/orgs/"+fmt.Sprintf("%.0f", newOrgID),
			strings.NewReader(`{"name":"Hacked"}`))
		nonOwnerPatch.Header.Set("Content-Type", "application/json")
		nonOwnerPatch.Header.Set("Authorization", "Bearer "+token2)
		nonOwnerW := httptest.NewRecorder()
		protected.ServeHTTP(nonOwnerW, nonOwnerPatch)
		if nonOwnerW.Code != http.StatusForbidden {
			t.Errorf("non-owner patch: expected 403, got %d: %s", nonOwnerW.Code, nonOwnerW.Body.String())
		}

		// DELETE /api/orgs/{id} — non-owner should get 403
		nonOwnerDel := httptest.NewRequest("DELETE", "/api/orgs/"+fmt.Sprintf("%.0f", newOrgID), nil)
		nonOwnerDel.Header.Set("Authorization", "Bearer "+token2)
		nonOwnerDelW := httptest.NewRecorder()
		protected.ServeHTTP(nonOwnerDelW, nonOwnerDel)
		if nonOwnerDelW.Code != http.StatusForbidden {
			t.Errorf("non-owner delete: expected 403, got %d", nonOwnerDelW.Code)
		}

		// DELETE /api/orgs/{id} — owner can delete
		delReq := httptest.NewRequest("DELETE", "/api/orgs/"+fmt.Sprintf("%.0f", newOrgID), nil)
		delReq.Header.Set("Authorization", "Bearer "+token)
		delW := httptest.NewRecorder()
		protected.ServeHTTP(delW, delReq)

		if delW.Code != http.StatusOK {
			t.Fatalf("delete org: expected 200, got %d: %s", delW.Code, delW.Body.String())
		}

		// Verify org is gone
		getAfterDel := httptest.NewRequest("GET", "/api/orgs/"+fmt.Sprintf("%.0f", newOrgID), nil)
		getAfterDel.Header.Set("Authorization", "Bearer "+token)
		getAfterDelW := httptest.NewRecorder()
		protected.ServeHTTP(getAfterDelW, getAfterDel)
		if getAfterDelW.Code != http.StatusNotFound {
			t.Errorf("get after delete: expected 404, got %d", getAfterDelW.Code)
		}
	})

	t.Run("default_org_backfill", func(t *testing.T) {
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

		ctx := context.Background()

		// Manually insert a user with NULL default_org_id (simulating pre-migration state)
		_, err := d.ExecContext(ctx,
			`INSERT INTO users (email, password_hash, role, created_at) VALUES (?, ?, 'user', ?)`,
			"legacy@test.com", "hash", "2024-01-01T00:00:00Z")
		if err != nil {
			t.Fatalf("insert legacy user: %v", err)
		}

		// Verify default_org_id is NULL
		var defaultOrgID sql.NullInt64
		err = d.QueryRowContext(ctx, "SELECT default_org_id FROM users WHERE email = ?", "legacy@test.com").Scan(&defaultOrgID)
		if err != nil {
			t.Fatalf("query legacy user: %v", err)
		}
		if defaultOrgID.Valid {
			t.Error("expected NULL default_org_id for legacy user before backfill")
		}

		// Run backfill
		a.BackfillDefaultOrgs()

		// Verify backfill created a personal org and set default_org_id
		err = d.QueryRowContext(ctx, "SELECT default_org_id FROM users WHERE email = ?", "legacy@test.com").Scan(&defaultOrgID)
		if err != nil {
			t.Fatalf("query legacy user after backfill: %v", err)
		}
		if !defaultOrgID.Valid {
			t.Fatal("expected default_org_id to be set after backfill")
		}

		// Verify the org exists
		var orgName string
		err = d.QueryRowContext(ctx, "SELECT name FROM orgs WHERE id = ?", defaultOrgID.Int64).Scan(&orgName)
		if err != nil {
			t.Fatalf("query backfilled org: %v", err)
		}
		if orgName != "legacy@test.com" {
			t.Errorf("expected org name 'legacy@test.com', got %q", orgName)
		}
	})
}

func TestStateSignRoundTrip(t *testing.T) {
	secret := []byte("test-secret-32-chars!!!")
	state := "abc123def456"

	signed := auth.SignState(state, secret)
	if signed == "" {
		t.Fatal("expected non-empty signed state")
	}

	// verifyState should succeed with matching state and secret
	if !auth.VerifyState(signed, state, secret) {
		t.Fatal("expected VerifyState to return true for valid signed state")
	}

	// verifyState should fail with wrong state
	if auth.VerifyState(signed, "wrong-state", secret) {
		t.Fatal("expected VerifyState to return false for wrong state")
	}

	// verifyState should fail with wrong secret
	wrongSecret := []byte("wrong-secret!!!!!!")
	if auth.VerifyState(signed, state, wrongSecret) {
		t.Fatal("expected VerifyState to return false for wrong secret")
	}

	// verifyState should fail with tampered signature
	parts := strings.Split(signed, ".")
	if len(parts) == 2 {
		tampered := parts[0] + ".tampered"
		if auth.VerifyState(tampered, state, secret) {
			t.Fatal("expected VerifyState to return false for tampered signature")
		}
	}

	// verifyState should fail with malformed input (no dot)
	if auth.VerifyState("no-dot-here", state, secret) {
		t.Fatal("expected VerifyState to return false for malformed input")
	}

	// verifyState should fail with empty state
	if auth.VerifyState("", state, secret) {
		t.Fatal("expected VerifyState to return false for empty input")
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

func TestListUsersOrgScoping(t *testing.T) {
	_, handler, protected, d := setupAuthWithDB(t)

	// Register 2 admins to represent two distinct orgs
	// Admin 1 (gets Org 1, default role is admin because it's first user)
	req1 := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"admin1@org1.com","password":"password123"}`))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("register admin1: expected 201, got %d: %s", w1.Code, w1.Body.String())
	}

	// Admin 2 (gets Org 2, default role is user, we will update it to admin)
	req2 := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"admin2@org2.com","password":"password123"}`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("register admin2: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	// Update admin2's role to admin in the DB so they are authorized to list users
	_, err := d.Exec("UPDATE users SET role = 'admin' WHERE email = ?", "admin2@org2.com")
	if err != nil {
		t.Fatalf("failed to update admin2 role: %v", err)
	}

	// Log in as Admin 1
	login1Body := `{"email":"admin1@org1.com","password":"password123"}`
	login1Req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(login1Body))
	login1Req.Header.Set("Content-Type", "application/json")
	login1W := httptest.NewRecorder()
	handler.ServeHTTP(login1W, login1Req)
	resp1 := parseResponse(t, login1W.Body.Bytes())
	token1, _ := resp1["token"].(string)

	// Log in as Admin 2
	login2Body := `{"email":"admin2@org2.com","password":"password123"}`
	login2Req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(login2Body))
	login2Req.Header.Set("Content-Type", "application/json")
	login2W := httptest.NewRecorder()
	handler.ServeHTTP(login2W, login2Req)
	resp2 := parseResponse(t, login2W.Body.Bytes())
	token2, _ := resp2["token"].(string)

	// Admin 1 queries users list. Should only see admin1@org1.com (since admin2 is in Org 2).
	listReq1 := httptest.NewRequest("GET", "/api/auth/users", nil)
	listReq1.Header.Set("Authorization", "Bearer "+token1)
	listW1 := httptest.NewRecorder()
	protected.ServeHTTP(listW1, listReq1)

	if listW1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listW1.Code, listW1.Body.String())
	}

	var body1 map[string][]map[string]any
	if err := json.NewDecoder(listW1.Body).Decode(&body1); err != nil {
		t.Fatalf("decode: %v", err)
	}
	users1 := body1["data"]
	if len(users1) != 1 {
		t.Fatalf("admin1: expected 1 user, got %d: %v", len(users1), users1)
	}
	if users1[0]["email"] != "admin1@org1.com" {
		t.Errorf("admin1: expected to see admin1@org1.com, got %v", users1[0]["email"])
	}

	// Admin 2 queries users list. Should only see admin2@org2.com.
	listReq2 := httptest.NewRequest("GET", "/api/auth/users", nil)
	listReq2.Header.Set("Authorization", "Bearer "+token2)
	listW2 := httptest.NewRecorder()
	protected.ServeHTTP(listW2, listReq2)

	if listW2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listW2.Code, listW2.Body.String())
	}

	var body2 map[string][]map[string]any
	if err := json.NewDecoder(listW2.Body).Decode(&body2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	users2 := body2["data"]
	if len(users2) != 1 {
		t.Fatalf("admin2: expected 1 user, got %d: %v", len(users2), users2)
	}
	if users2[0]["email"] != "admin2@org2.com" {
		t.Errorf("admin2: expected to see admin2@org2.com, got %v", users2[0]["email"])
	}
}

func setupAuthWithDB(t *testing.T) (*auth.Auth, http.Handler, http.Handler, *db.DB) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	a.SetOTPStore(auth.NewMapOTPStore())
	a.SetRateLimitStore(auth.NewMapRateLimitStore())
	a.SetLoginLockoutStore(auth.NewMapLoginLockoutStore())

	k.Register(a)
	k.Register(d)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	return a, a.Handler(), a.ProtectedHandler(), d
}
