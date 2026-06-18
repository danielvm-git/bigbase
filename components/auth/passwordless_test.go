package auth_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

type captureSender struct {
	lastTo    string
	lastSubj  string
	lastBody  string
}

func (c *captureSender) SendEmail(to, subject, body string) {
	c.lastTo = to
	c.lastSubj = subject
	c.lastBody = body
}

func setupWithEmail(t *testing.T) (*auth.Auth, http.Handler, *captureSender) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	sender := &captureSender{}
	a := auth.New(auth.Options{
		DB:          d,
		Logger:      logger,
		Secret:      "test-secret-32-chars!!!",
		EmailSender: sender,
	})
	k.Register(d)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return a, a.Handler(), sender
}

func TestOTPEndToEnd(t *testing.T) {
	_, h, sender := setupWithEmail(t)

	// Send OTP.
	sendBody := `{"email":"test@example.com"}`
	req := httptest.NewRequest("POST", "/api/auth/otp/send", strings.NewReader(sendBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if sender.lastTo != "test@example.com" {
		t.Errorf("expected email to test@example.com, got %q", sender.lastTo)
	}
	// Extract code from email body (format: "Your code is: 123456").
	code := ""
	fmt.Sscanf(sender.lastBody, "Your code is: %s", &code)
	if len(code) != 6 {
		t.Fatalf("could not extract 6-digit code from email body: %q", sender.lastBody)
	}

	// Verify OTP.
	verifyBody := fmt.Sprintf(`{"email":"test@example.com","code":"%s"}`, code)
	req2 := httptest.NewRequest("POST", "/api/auth/otp/verify", strings.NewReader(verifyBody))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rec2.Body).Decode(&resp)
	if resp["token"] == nil {
		t.Error("expected JWT token in response")
	}
	if rec2.Header().Get("Set-Cookie") == "" {
		t.Error("expected Set-Cookie header")
	}
}

func TestOTPRateLimit(t *testing.T) {
	_, h, _ := setupWithEmail(t)

	sendBody := `{"email":"spam@example.com"}`
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/auth/otp/send", strings.NewReader(sendBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("iter %d: expected 200, got %d", i, rec.Code)
		}
	}

	// 4th request should be rate-limited.
	req := httptest.NewRequest("POST", "/api/auth/otp/send", strings.NewReader(sendBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}
}

func TestMagicLinkEndToEnd(t *testing.T) {
	_, h, sender := setupWithEmail(t)

	// Send magic link.
	sendBody := `{"email":"magic@example.com"}`
	req := httptest.NewRequest("POST", "/api/auth/magic-link/send", strings.NewReader(sendBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if sender.lastTo != "magic@example.com" {
		t.Errorf("expected magic@example.com, got %q", sender.lastTo)
	}

	// Extract token from URL. Body contains: "Click here to sign in: http://...?token=xxx"
	token := ""
	idx := strings.Index(sender.lastBody, "token=")
	if idx == -1 {
		t.Fatalf("could not find token in email body: %q", sender.lastBody)
	}
	parts := strings.SplitN(sender.lastBody[idx+6:], "&", 2)
	token = strings.TrimSpace(parts[0])

	// Verify magic link.
	req2 := httptest.NewRequest("GET", "/api/auth/magic-link/verify?token="+token, nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec2.Code)
	}
	if rec2.Header().Get("Location") != "/admin/" {
		t.Errorf("expected /admin/, got %q", rec2.Header().Get("Location"))
	}
	if rec2.Header().Get("Set-Cookie") == "" {
		t.Error("expected Set-Cookie header")
	}

	// Re-use same token → should fail.
	req3 := httptest.NewRequest("GET", "/api/auth/magic-link/verify?token="+token, nil)
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for reused token, got %d", rec3.Code)
	}
}
