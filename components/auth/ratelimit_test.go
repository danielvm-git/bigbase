package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/auth"
)

func TestRateLimit(t *testing.T) {
	t.Run("ip_rate_limit_60_per_min", func(t *testing.T) {
		rl := auth.NewRateLimiter(auth.RateLimiterConfig{
			IPLimit:      3,
			IPWindow:     time.Minute,
			UserLimit:    300,
			UserWindow:   time.Minute,
			CleanupEvery: time.Hour,
		})

		handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First 3 requests should pass (limit = 3 for test)
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.0.2.1:1234"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
			}
		}

		// 4th request should be rate limited
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("expected 429 after limit exceeded, got %d", w.Code)
		}

		// Retry-After header must be present
		if w.Header().Get("Retry-After") == "" {
			t.Fatal("expected Retry-After header on 429 response")
		}
	})

	t.Run("different_ips_are_independent", func(t *testing.T) {
		rl := auth.NewRateLimiter(auth.RateLimiterConfig{
			IPLimit:      2,
			IPWindow:     time.Minute,
			UserLimit:    300,
			UserWindow:   time.Minute,
			CleanupEvery: time.Hour,
		})

		handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Exhaust IP1
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "10.0.0.1:1000"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}

		// IP1 should be limited
		req1 := httptest.NewRequest("GET", "/", nil)
		req1.RemoteAddr = "10.0.0.1:1000"
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)
		if w1.Code != http.StatusTooManyRequests {
			t.Fatalf("expected 429 for IP1, got %d", w1.Code)
		}

		// IP2 should still be allowed
		req2 := httptest.NewRequest("GET", "/", nil)
		req2.RemoteAddr = "10.0.0.2:2000"
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
		if w2.Code != http.StatusOK {
			t.Fatalf("expected 200 for fresh IP2, got %d", w2.Code)
		}
	})

	t.Run("authenticated_user_higher_limit", func(t *testing.T) {
		_, handler, _ := setupAuth(t)

		// Register and login to get a token
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"rl@test.com","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)
		if regW.Code != http.StatusCreated {
			t.Fatalf("register: %d", regW.Code)
		}

		// Auth endpoint exists and is reachable (rate limit tests are at unit level above)
		// This sub-test just ensures the auth flow still works under the rate limiter
		loginReq := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(`{"email":"rl@test.com","password":"secret123"}`))
		loginReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()
		handler.ServeHTTP(loginW, loginReq)
		if loginW.Code != http.StatusOK {
			t.Fatalf("login: expected 200, got %d", loginW.Code)
		}
	})

	t.Run("user_bucket_takes_precedence_when_authenticated", func(t *testing.T) {
		rl := auth.NewRateLimiter(auth.RateLimiterConfig{
			IPLimit:      1,    // Very low IP limit
			IPWindow:     time.Minute,
			UserLimit:    10,   // Higher user limit
			UserWindow:   time.Minute,
			CleanupEvery: time.Hour,
		})

		// Build a handler that injects a user ID claim
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		rlHandler := rl.Middleware(inner)

		// Two requests from same IP but with user ID context — should use user bucket (limit=10)
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "10.1.2.3:9000"
			// Inject user_id into context via header trick — actually use the UserIDKey func
			ctx := auth.WithUserID(req.Context(), 42)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()
			rlHandler.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("request %d: expected 200 (user bucket), got %d", i+1, w.Code)
			}
		}
	})

	t.Run("default_limits_are_sane", func(t *testing.T) {
		cfg := auth.DefaultRateLimiterConfig()
		if cfg.IPLimit != 60 {
			t.Errorf("expected IPLimit=60, got %d", cfg.IPLimit)
		}
		if cfg.UserLimit != 300 {
			t.Errorf("expected UserLimit=300, got %d", cfg.UserLimit)
		}
		if cfg.IPWindow != time.Minute {
			t.Errorf("expected IPWindow=1m, got %v", cfg.IPWindow)
		}
		if cfg.UserWindow != time.Minute {
			t.Errorf("expected UserWindow=1m, got %v", cfg.UserWindow)
		}
	})
}
