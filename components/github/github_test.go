package github_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/github"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func TestVerifyWebhookSignatureValid(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"ref":"refs/heads/main"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !github.VerifyWebhookSignatureForTest(secret, body, sig) {
		t.Fatal("expected valid signature")
	}
}

func TestVerifyWebhookSignatureInvalid(t *testing.T) {
	if github.VerifyWebhookSignatureForTest("secret", []byte("x"), "sha256=deadbeef") {
		t.Fatal("expected invalid signature")
	}
}

// setupGitHub creates an in-memory DB and returns a github component handler.
// opts are passed through to github.New() — the caller controls whether flags are set.
func setupGitHub(t *testing.T, opts github.Options) http.Handler {
	t.Helper()
	logger := testLogger{}
	opts.DB = db.New(db.Options{Path: ":memory:", Logger: logger})
	opts.Logger = logger
	ctx := &kernel.Context{}
	if err := opts.DB.(*db.DB).Start(ctx); err != nil {
		t.Fatalf("db start: %v", err)
	}
	t.Cleanup(func() { _ = opts.DB.(*db.DB).Stop(ctx) })
	return github.New(opts).Handler()
}

// Regression test for BUG-2026-06-02T000000: GitHub App CLI flags not registered.
// Verifies that github.New() correctly propagates AppID/AppSlug/PrivateKeyPath
// options so that configured() returns true and /api/github/install redirects.
func TestGitHubFlagsConfigured(t *testing.T) {
	handler := setupGitHub(t, github.Options{
		AppID:          "12345",
		AppSlug:        "test-app",
		PrivateKeyPath: "/tmp/key.pem",
		WebhookSecret:  "secret123",
	})

	t.Run("status reports configured=true", func(t *testing.T) {
		w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/github/status", nil)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		configured, ok := resp["configured"].(bool)
		if !ok {
			t.Fatal("response missing 'configured' field")
		}
		if !configured {
			t.Fatal("expected configured=true when GitHub flags are set")
		}
	})

	t.Run("install redirects not 503", func(t *testing.T) {
		w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/github/install", nil)
		handler.ServeHTTP(w, r)
		if w.Code == http.StatusServiceUnavailable {
			t.Fatal("install returned 503 - flags not wired")
		}
		if w.Code != http.StatusFound {
			t.Fatalf("expected 302 redirect, got %d", w.Code)
		}
	})
}

// Regression test: without flags, configured() must still return false
// and install must still return 503 (backward compatibility).
func TestGitHubFlagsUnconfigured(t *testing.T) {
	handler := setupGitHub(t, github.Options{})

	t.Run("status reports configured=false", func(t *testing.T) {
		w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/github/status", nil)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		configured, ok := resp["configured"].(bool)
		if !ok {
			t.Fatal("response missing 'configured' field")
		}
		if configured {
			t.Fatal("expected configured=false when no GitHub flags set")
		}
	})

	t.Run("install returns 503", func(t *testing.T) {
		w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/github/install", nil)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 when unconfigured, got %d", w.Code)
		}
	})
}

func setupGitHubPublic(t *testing.T, opts github.Options) http.Handler {
	t.Helper()
	logger := testLogger{}
	opts.DB = db.New(db.Options{Path: ":memory:", Logger: logger})
	opts.Logger = logger
	ctx := &kernel.Context{}
	if err := opts.DB.(*db.DB).Start(ctx); err != nil {
		t.Fatalf("db start: %v", err)
	}
	t.Cleanup(func() { _ = opts.DB.(*db.DB).Stop(ctx) })
	return github.New(opts).PublicHandler()
}

// BUG-2026-06-03T120000: callback must work without session after GitHub redirect.
func TestGitHubCallbackPublicNoAuth(t *testing.T) {
	handler := setupGitHubPublic(t, github.Options{
		AppID:          "12345",
		AppSlug:        "test-app",
		PrivateKeyPath: "/tmp/key.pem",
	})

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/github/callback?installation_id=42", nil)
	handler.ServeHTTP(w, r)
	if w.Code == http.StatusUnauthorized {
		t.Fatal("callback must not require auth")
	}
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d body=%s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != "/admin/#/deploy/new" {
		t.Fatalf("unexpected redirect: %q", loc)
	}
}

func TestGitHubWebhookPublicNotUnauthorized(t *testing.T) {
	handler := setupGitHubPublic(t, github.Options{WebhookSecret: "secret"})

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/github/webhook", nil)
	handler.ServeHTTP(w, r)
	if strings.Contains(w.Body.String(), "authorization required") {
		t.Fatalf("webhook must not use auth middleware, got %s", w.Body.String())
	}
}
