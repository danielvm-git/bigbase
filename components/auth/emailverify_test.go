package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

// capturedEmail holds emails sent via the mock messaging sender.
type capturedEmail struct {
	To      string
	Subject string
	Body    string
}

// mockEmailSender captures sent emails instead of delivering them.
type mockEmailSender struct {
	sent []capturedEmail
}

func (m *mockEmailSender) SendEmail(to, subject, body string) {
	m.sent = append(m.sent, capturedEmail{To: to, Subject: subject, Body: body})
}

func setupAuthWithEmail(t *testing.T) (*auth.Auth, *mockEmailSender, http.Handler, http.Handler) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	sender := &mockEmailSender{}
	a := auth.New(auth.Options{
		DB:          d,
		Logger:      logger,
		Secret:      "test-secret-32-chars!!!",
		EmailSender: sender,
	})

	k.Register(a)
	k.Register(d)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	return a, sender, a.Handler(), a.ProtectedHandler()
}

func TestEmailVerify(t *testing.T) {
	t.Run("register_sends_verification_email", func(t *testing.T) {
		_, sender, handler, _ := setupAuthWithEmail(t)

		req := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"verify@test.com","password":"secret123"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		if len(sender.sent) == 0 {
			t.Fatal("expected verification email to be sent")
		}

		if sender.sent[0].To != "verify@test.com" {
			t.Errorf("expected email to 'verify@test.com', got %q", sender.sent[0].To)
		}
	})

	t.Run("verify_email_valid_token", func(t *testing.T) {
		a, sender, handler, _ := setupAuthWithEmail(t)

		// Register to trigger verification email
		req := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"v2@test.com","password":"secret123"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("register: %d %s", w.Code, w.Body.String())
		}

		// Extract token sent in the email
		if len(sender.sent) == 0 {
			t.Fatal("no email sent")
		}
		token := a.ExtractVerifyToken(sender.sent[0].Body)
		if token == "" {
			t.Fatalf("could not extract token from email body: %q", sender.sent[0].Body)
		}

		// Verify the email
		verifyReq := httptest.NewRequest("GET", "/api/auth/verify-email?token="+token, nil)
		verifyW := httptest.NewRecorder()
		handler.ServeHTTP(verifyW, verifyReq)
		if verifyW.Code != http.StatusOK {
			t.Fatalf("verify: expected 200, got %d: %s", verifyW.Code, verifyW.Body.String())
		}
	})

	t.Run("verify_email_invalid_token", func(t *testing.T) {
		_, _, handler, _ := setupAuthWithEmail(t)

		verifyReq := httptest.NewRequest("GET", "/api/auth/verify-email?token=badtoken123", nil)
		verifyW := httptest.NewRecorder()
		handler.ServeHTTP(verifyW, verifyReq)
		if verifyW.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid token, got %d", verifyW.Code)
		}
	})

	t.Run("verify_email_missing_token", func(t *testing.T) {
		_, _, handler, _ := setupAuthWithEmail(t)

		verifyReq := httptest.NewRequest("GET", "/api/auth/verify-email", nil)
		verifyW := httptest.NewRecorder()
		handler.ServeHTTP(verifyW, verifyReq)
		if verifyW.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing token, got %d", verifyW.Code)
		}
	})

	t.Run("unverified_user_blocked_on_protected_routes", func(t *testing.T) {
		_, _, handler, protected := setupAuthWithEmail(t)

		// Register (email not verified)
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"unverified@test.com","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)
		if regW.Code != http.StatusCreated {
			t.Fatalf("register: %d", regW.Code)
		}

		resp := parseResponse(t, regW.Body.Bytes())
		token, _ := resp["token"].(string)

		// Try accessing a protected route without verifying email
		protReq := httptest.NewRequest("GET", "/api/auth/me", nil)
		protReq.Header.Set("Authorization", "Bearer "+token)
		protW := httptest.NewRecorder()
		protected.ServeHTTP(protW, protReq)
		if protW.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for unverified user, got %d: %s", protW.Code, protW.Body.String())
		}
	})

	t.Run("verified_user_allowed_on_protected_routes", func(t *testing.T) {
		a, sender, handler, protected := setupAuthWithEmail(t)

		// Register
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"verified@test.com","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)
		if regW.Code != http.StatusCreated {
			t.Fatalf("register: %d", regW.Code)
		}

		resp := parseResponse(t, regW.Body.Bytes())
		token, _ := resp["token"].(string)

		// Verify email
		token6 := a.ExtractVerifyToken(sender.sent[0].Body)
		verifyReq := httptest.NewRequest("GET", "/api/auth/verify-email?token="+token6, nil)
		verifyW := httptest.NewRecorder()
		handler.ServeHTTP(verifyW, verifyReq)
		if verifyW.Code != http.StatusOK {
			t.Fatalf("verify: %d %s", verifyW.Code, verifyW.Body.String())
		}

		// Now access protected route — should be allowed
		protReq := httptest.NewRequest("GET", "/api/auth/me", nil)
		protReq.Header.Set("Authorization", "Bearer "+token)
		protW := httptest.NewRecorder()
		protected.ServeHTTP(protW, protReq)
		if protW.Code != http.StatusOK {
			t.Fatalf("expected 200 for verified user, got %d: %s", protW.Code, protW.Body.String())
		}
	})

	t.Run("no_email_sender_skips_verification", func(t *testing.T) {
		// When no EmailSender is configured, registration should succeed without verification gate
		_, handler, protected := setupAuth(t)

		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"nosender@test.com","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)
		if regW.Code != http.StatusCreated {
			t.Fatalf("register: %d %s", regW.Code, regW.Body.String())
		}

		resp := parseResponse(t, regW.Body.Bytes())
		token, _ := resp["token"].(string)

		protReq := httptest.NewRequest("GET", "/api/auth/me", nil)
		protReq.Header.Set("Authorization", "Bearer "+token)
		protW := httptest.NewRecorder()
		protected.ServeHTTP(protW, protReq)
		if protW.Code != http.StatusOK {
			t.Fatalf("expected 200 when no email sender, got %d: %s", protW.Code, protW.Body.String())
		}
	})
}
