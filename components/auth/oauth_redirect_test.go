package auth_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func setupOAuthWithOpts(t *testing.T, opts auth.Options) (*auth.Auth, http.Handler) {
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
	k.Register(d)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return a, a.Handler()
}

func TestGoogleCallbackRedirectConfigurable(t *testing.T) {
	// Default PostLoginRedirect = "/admin/"
	a, h := setupOAuthWithOpts(t, auth.Options{})
	a.SetGoogleVerifier(&mockGoogleVerifier{user: &auth.GoogleUser{GoogleID: "g-1", Email: "test@example.com"}})

	rawState := "test-state-abc123"
	signedState := auth.SignState(rawState, []byte("test-secret-32-chars!!!"))
	req := httptest.NewRequest("GET", "/api/auth/oauth/google/callback?code=test-code&state="+rawState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: signedState})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/admin/" {
		t.Errorf("expected /admin/, got %q", rec.Header().Get("Location"))
	}
	if rec.Header().Get("Set-Cookie") == "" {
		t.Error("expected Set-Cookie with token")
	}
}

func TestGoogleCallbackSPATokenDelivery(t *testing.T) {
	a, h := setupOAuthWithOpts(t, auth.Options{
		SPAOriginAllowlist: []string{"https://bolao.example.com"},
	})
	a.SetGoogleVerifier(&mockGoogleVerifier{user: &auth.GoogleUser{GoogleID: "g-1", Email: "test@example.com"}})

	rawState := "test-state-spa"
	spaRedirect := "https://bolao.example.com/dashboard"
	signedState := auth.SignSPAState(rawState, spaRedirect, []byte("test-secret-32-chars!!!"))

	req := httptest.NewRequest("GET", "/api/auth/oauth/google/callback?code=test-code&state="+rawState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: signedState})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://bolao.example.com/dashboard#token=") {
		t.Errorf("expected SPA redirect with #token=, got %q", loc)
	}
	// No HttpOnly token cookie should be set (only oauth_state clearance cookie).
	setCookie := rec.Header().Get("Set-Cookie")
	if setCookie != "" && !strings.Contains(setCookie, "oauth_state") {
		t.Errorf("expected only oauth_state clear cookie, got %q", setCookie)
	}
	// Token in fragment should be URL-encoded.
	fragment := loc[strings.Index(loc, "#token=")+7:]
	decoded, err := url.QueryUnescape(fragment)
	if err != nil || decoded == "" {
		t.Errorf("expected valid URL-encoded JWT in fragment, got %q", fragment)
	}
}

func TestGoogleCallbackSPARejectedOrigin(t *testing.T) {
	a, h := setupOAuthWithOpts(t, auth.Options{
		SPAOriginAllowlist: []string{"https://trusted.example.com"},
	})
	a.SetGoogleVerifier(&mockGoogleVerifier{user: &auth.GoogleUser{GoogleID: "g-1", Email: "test@example.com"}})

	rawState := "test-state-rejected"
	// SPA redirect to an origin NOT in the allowlist.
	spaRedirect := "https://evil.com/steal-token"
	signedState := auth.SignSPAState(rawState, spaRedirect, []byte("test-secret-32-chars!!!"))

	req := httptest.NewRequest("GET", "/api/auth/oauth/google/callback?code=test-code&state="+rawState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: signedState})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	// Should fall back to default /admin/ redirect with cookie.
	if rec.Header().Get("Location") != "/admin/" {
		t.Errorf("expected fallback to /admin/, got %q", rec.Header().Get("Location"))
	}
	if rec.Header().Get("Set-Cookie") == "" {
		t.Error("expected Set-Cookie with token for fallback")
	}
}

func TestGoogleCallbackDefaultBehavior(t *testing.T) {
	// No SPAOriginAllowlist configured — should behave as before.
	a, h := setupOAuthWithOpts(t, auth.Options{})
	a.SetGoogleVerifier(&mockGoogleVerifier{user: &auth.GoogleUser{GoogleID: "g-1", Email: "test@example.com"}})

	rawState := "test-state-default"
	signedState := auth.SignState(rawState, []byte("test-secret-32-chars!!!"))

	req := httptest.NewRequest("GET", "/api/auth/oauth/google/callback?code=test-code&state="+rawState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: signedState})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/admin/" {
		t.Errorf("expected /admin/, got %q", rec.Header().Get("Location"))
	}
	if rec.Header().Get("Set-Cookie") == "" {
		t.Error("expected Set-Cookie with token")
	}
}
