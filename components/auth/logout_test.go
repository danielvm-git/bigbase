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

func TestLogout(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{
		DB:     d,
		Logger: logger,
		Secret: "test-secret-32-chars!!!",
	})
	k.Register(d)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	defer k.Stop()

	h := a.Handler()

	// Send POST /api/auth/logout with an existing token cookie.
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "some-old-token"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Set-Cookie should expire the token cookie.
	setCookie := rec.Header().Get("Set-Cookie")
	if setCookie == "" {
		t.Fatal("expected Set-Cookie header")
	}
	if !strings.Contains(setCookie, "token=;") {
		t.Errorf("expected token cookie to be cleared, got %q", setCookie)
	}
	if !strings.Contains(setCookie, "Max-Age=0") {
		t.Errorf("expected Max-Age=0, got %q", setCookie)
	}

	// Body should be valid JSON.
	body := rec.Body.String()
	if !strings.Contains(body, `"ok"`) {
		t.Errorf("expected JSON with ok:true, got %q", body)
	}
}

func TestLogoutWithoutCookie(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{
		DB:     d,
		Logger: logger,
		Secret: "test-secret-32-chars!!!",
	})
	k.Register(d)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	defer k.Stop()

	h := a.Handler()

	// Logout without any cookie — still returns 200 (idempotent).
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// Still clears the cookie even if not present.
	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "token=;") {
		t.Errorf("expected cookie to be cleared even without existing cookie, got %q", setCookie)
	}
}
