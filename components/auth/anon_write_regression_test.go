package auth

// Regression test (e81s06) for the anonymous-token write block: anonymous JWTs
// may read (GET/HEAD/OPTIONS) but every mutating method must be rejected with
// 403. Guards components/auth/middleware.go against a silent regression that
// would let anonymous callers write. The valid-anonymous path touches neither
// the DB nor the logger, so a minimal Auth is sufficient.

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAnonymousWriteBlocked(t *testing.T) {
	secret := []byte("test-secret-32-chars!!!")
	a := &Auth{secret: secret}

	tok, err := createJWT(0, "anon@example.test", "anonymous", 0, secret, time.Hour)
	if err != nil {
		t.Fatalf("mint anonymous token: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := a.Middleware(next)

	call := func(method string) int {
		req := httptest.NewRequest(method, "/api/resource", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)
		return rec.Code
	}

	// Read methods are allowed.
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		if code := call(m); code != http.StatusOK {
			t.Errorf("anonymous %s: got %d, want 200 (read must be allowed)", m, code)
		}
	}

	// Every mutating method must be blocked with 403.
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		if code := call(m); code != http.StatusForbidden {
			t.Errorf("anonymous %s: got %d, want 403 (write must be blocked)", m, code)
		}
	}
}
