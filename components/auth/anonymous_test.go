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

func setupPopupAuth(t *testing.T, opts auth.Options) (*auth.Auth, http.Handler) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	opts.DB = d
	opts.Logger = logger
	if opts.Secret == "" {
		opts.Secret = "test-secret-32-chars!!!"
	}
	if opts.GoogleClientID == "" {
		opts.GoogleClientID = "test-client-id"
	}
	if opts.PublicURL == "" {
		opts.PublicURL = "https://test.example.com"
	}
	a := auth.New(opts)
	a.SetOTPStore(auth.NewMapOTPStore())
	a.SetRateLimitStore(auth.NewMapRateLimitStore())
	a.SetLoginLockoutStore(auth.NewMapLoginLockoutStore())
	k.Register(d)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return a, a.Handler()
}

// TestPopupCallbackNoRedirect_FailsClosed verifies that popup callback
// without a redirect query param returns an error page (not postMessage).
// This prevents JWT leakage via postMessage("*").
func TestPopupCallbackNoRedirect_FailsClosed(t *testing.T) {
	a, h := setupPopupAuth(t, auth.Options{
		SPAOriginAllowlist: []string{"https://bolao.example.com"},
	})
	a.SetGoogleVerifier(&mockGoogleVerifier{
		user: &auth.GoogleUser{GoogleID: "g-1", Email: "test@example.com"},
	})

	rawState := "test-state-no-redirect"
	signedState := auth.SignState(rawState, []byte("test-secret-32-chars!!!"))

	req := httptest.NewRequest("GET", "/api/auth/oauth/google/popup?code=test-code&state="+rawState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: signedState})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Should NOT return postMessage HTML with wildcard origin
	body := rec.Body.String()
	if strings.Contains(body, `postMessage`) && strings.Contains(body, `"*"`) {
		t.Error("FAIL: popup callback returned postMessage with wildcard origin — JWT would leak to any origin")
	}

	// Should return error or redirect, not postMessage HTML
	if rec.Code == http.StatusOK && strings.Contains(body, `postMessage`) {
		t.Errorf("FAIL: popup callback returned 200 with postMessage HTML (body: %s)", body)
	}
}

// TestPopupCallbackNonAllowlistedRedirect_FailsClosed verifies that popup callback
// with a non-allowlisted redirect returns an error (not postMessage to that origin).
func TestPopupCallbackNonAllowlistedRedirect_FailsClosed(t *testing.T) {
	a, h := setupPopupAuth(t, auth.Options{
		SPAOriginAllowlist: []string{"https://trusted.example.com"},
	})
	a.SetGoogleVerifier(&mockGoogleVerifier{
		user: &auth.GoogleUser{GoogleID: "g-1", Email: "test@example.com"},
	})

	rawState := "test-state-evil"
	spaRedirect := "https://evil.com/steal-token"
	signedState := auth.SignSPAState(rawState, spaRedirect, []byte("test-secret-32-chars!!!"))

	req := httptest.NewRequest("GET", "/api/auth/oauth/google/popup?code=test-code&state="+rawState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: signedState})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Should NOT return postMessage HTML with evil.com origin
	body := rec.Body.String()
	if strings.Contains(body, `postMessage`) && strings.Contains(body, "evil.com") {
		t.Error("FAIL: popup callback returned postMessage to evil.com — JWT would leak to attacker")
	}

	// Should return error or redirect, not postMessage HTML
	if rec.Code == http.StatusOK && strings.Contains(body, `postMessage`) {
		t.Errorf("FAIL: popup callback returned 200 with postMessage HTML (body: %s)", body)
	}
}

// TestPopupCallbackValidRedirect_UsesExactOrigin verifies that popup callback
// with a valid allowlisted redirect uses the exact origin (not wildcard).
func TestPopupCallbackValidRedirect_UsesExactOrigin(t *testing.T) {
	a, h := setupPopupAuth(t, auth.Options{
		SPAOriginAllowlist: []string{"https://bolao.example.com"},
	})
	a.SetGoogleVerifier(&mockGoogleVerifier{
		user: &auth.GoogleUser{GoogleID: "g-1", Email: "test@example.com"},
	})

	rawState := "test-state-valid"
	spaRedirect := "https://bolao.example.com/dashboard"
	signedState := auth.SignSPAState(rawState, spaRedirect, []byte("test-secret-32-chars!!!"))

	req := httptest.NewRequest("GET", "/api/auth/oauth/google/popup?code=test-code&state="+rawState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: signedState})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Should use exact origin, NOT wildcard
	if strings.Contains(body, `"*"`) {
		t.Error("FAIL: popup callback used wildcard origin — must use exact origin from allowlist")
	}

	// Should use the exact origin from the allowlist
	if !strings.Contains(body, "https://bolao.example.com") {
		t.Errorf("FAIL: popup callback did not use allowlisted origin (body: %s)", body)
	}

	// Should contain the JWT token
	if !strings.Contains(body, `bigbase-auth:oauth-complete`) {
		t.Errorf("FAIL: popup callback missing auth message (body: %s)", body)
	}
}

// TestPopupCallbackSetsCOOPHeader verifies that popup callback responses
// include Cross-Origin-Opener-Policy: same-origin header as defense in depth.
func TestPopupCallbackSetsCOOPHeader(t *testing.T) {
	a, h := setupPopupAuth(t, auth.Options{
		SPAOriginAllowlist: []string{"https://bolao.example.com"},
	})
	a.SetGoogleVerifier(&mockGoogleVerifier{
		user: &auth.GoogleUser{GoogleID: "g-1", Email: "test@example.com"},
	})

	rawState := "test-state-coop"
	spaRedirect := "https://bolao.example.com/dashboard"
	signedState := auth.SignSPAState(rawState, spaRedirect, []byte("test-secret-32-chars!!!"))

	req := httptest.NewRequest("GET", "/api/auth/oauth/google/popup?code=test-code&state="+rawState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: signedState})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Check for Cross-Origin-Opener-Policy header
	coop := rec.Header().Get("Cross-Origin-Opener-Policy")
	if coop == "" {
		t.Error("FAIL: popup callback missing Cross-Origin-Opener-Policy header")
	} else if coop != "same-origin" {
		t.Errorf("FAIL: expected Cross-Origin-Opener-Policy: same-origin, got %q", coop)
	}
}
