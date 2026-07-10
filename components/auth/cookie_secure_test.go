package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCookieSecureWithHTTPSPublicURL(t *testing.T) {
	// BUG-160110: behind TLS proxy r.TLS is nil; https PublicURL must force Secure.
	_, handler, _ := setupAuthWithPublicURL(t, "https://bigbase.click")

	reg := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"secure@test.com","password":"secret123"}`))
	reg.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, reg)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}

	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "token" {
			found = true
			if !c.Secure {
				t.Fatal("token cookie must be Secure when PublicURL is https")
			}
		}
	}
	if !found {
		t.Fatal("expected token cookie")
	}
}

func TestCookieSecureWithForwardedProto(t *testing.T) {
	_, handler, _ := setupAuth(t)

	reg := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"fwd@test.com","password":"secret123"}`))
	reg.Header.Set("Content-Type", "application/json")
	reg.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, reg)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d", w.Code)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "token" && !c.Secure {
			t.Fatal("token cookie must be Secure when X-Forwarded-Proto=https")
		}
	}
}
