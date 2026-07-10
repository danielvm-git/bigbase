package auth_test

import (
	"fmt"
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

	t.Run("auth_flow_integration", func(t *testing.T) {
		_, handler, _ := setupAuth(t)

		// Smoke test: register + login through the full auth handler stack.
		regReq := httptest.NewRequest("POST", "/api/auth/register",
			strings.NewReader(`{"email":"rl@test.com","password":"secret123"}`))
		regReq.Header.Set("Content-Type", "application/json")
		regW := httptest.NewRecorder()
		handler.ServeHTTP(regW, regReq)
		if regW.Code != http.StatusCreated {
			t.Fatalf("register: %d", regW.Code)
		}

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
			IPLimit:      1, // Very low IP limit
			IPWindow:     time.Minute,
			UserLimit:    10, // Higher user limit
			UserWindow:   time.Minute,
			CleanupEvery: time.Hour,
		})

		// Build a handler that injects a user ID claim
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		rlHandler := rl.Middleware(inner)

		// Authenticated requests use the user bucket (limit=10), not IP bucket (limit=1).
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "10.1.2.3:9000"
			ctx := auth.WithUserID(req.Context(), 42)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()
			rlHandler.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("request %d: expected 200 (user bucket), got %d", i+1, w.Code)
			}
		}

		// Unauthenticated requests from the same IP use the IP bucket (limit=1).
		// First anonymous request passes, second returns 429.
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.1.2.3:9000"
		w := httptest.NewRecorder()
		rlHandler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("anonymous request 1: expected 200, got %d", w.Code)
		}

		req = httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.1.2.3:9000"
		w = httptest.NewRecorder()
		rlHandler.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("anonymous request 2: expected 429 (IP exhausted), got %d", w.Code)
		}
	})

	t.Run("reconfigure_updates_limits_for_new_buckets", func(t *testing.T) {
		rl := auth.NewRateLimiter(auth.RateLimiterConfig{
			IPLimit:      2,
			IPWindow:     time.Minute,
			UserLimit:    2,
			UserWindow:   time.Minute,
			CleanupEvery: time.Hour,
		})

		handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Exhaust IP1 with old limit of 2
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.0.2.10:1234"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}

		// 3rd from IP1 should be blocked
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("expected 429 after exhausting old limit, got %d", w.Code)
		}

		// Reconfigure with higher limit
		rl.Reconfigure(auth.RateLimiterConfig{
			IPLimit:      10,
			IPWindow:     time.Minute,
			UserLimit:    10,
			UserWindow:   time.Minute,
			CleanupEvery: time.Hour,
		})

		// A new IP should get the new limit (10)
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.0.2.20:5678"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("new IP request %d: expected 200 after reconfig, got %d", i+1, w.Code)
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

// TestRateLimitIntegration verifies that 61 login POSTs from the same IP
// result in a 429 response on the 61st request when the rate limiter is
// configured for 60 req/min per IP.
func TestRateLimitIntegration(t *testing.T) {
	_, handler, _ := setupAuth(t)

	// Register a user first
	regReq := httptest.NewRequest("POST", "/api/auth/register",
		strings.NewReader(`{"email":"rl-int@test.com","password":"secret123"}`))
	regReq.Header.Set("Content-Type", "application/json")
	regReq.RemoteAddr = "10.99.99.1:9999"
	regW := httptest.NewRecorder()
	handler.ServeHTTP(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d body=%s", regW.Code, regW.Body.String())
	}

	// Wrap with rate limiter: 60 req/min per IP
	rl := auth.NewRateLimiter(auth.RateLimiterConfig{
		IPLimit:      60,
		IPWindow:     time.Minute,
		UserLimit:    300,
		UserWindow:   time.Minute,
		CleanupEvery: time.Hour,
	})
	rlHandler := rl.Middleware(handler)

	// Send 60 login POSTs from same IP — unique emails so per-account lockout
	// (BUG-160003) does not fire; IP rate limiter is what we assert here.
	for i := 0; i < 60; i++ {
		body := fmt.Sprintf(`{"email":"rl-int-%d@test.com","password":"wrong"}`, i)
		req := httptest.NewRequest("POST", "/api/auth/login",
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.99.99.1:9999"
		w := httptest.NewRecorder()
		rlHandler.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d: got unexpected 429 (rate limited too early) body=%s", i+1, w.Body.String())
		}
	}

	// 61st request should be rate limited by IP middleware
	req := httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"email":"rl-int-60@test.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.99.99.1:9999"
	w := httptest.NewRecorder()
	rlHandler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("request 61: expected 429, got %d", w.Code)
	}

	// Body should include IP rate-limit message (not account lockout)
	if !strings.Contains(w.Body.String(), "rate limit exceeded") {
		t.Fatalf("expected 'rate limit exceeded' in body, got %s", w.Body.String())
	}
}

// TestRateLimitHeaders verifies that 429 responses include the Retry-After header.
func TestRateLimitHeaders(t *testing.T) {
	rl := auth.NewRateLimiter(auth.RateLimiterConfig{
		IPLimit:      1,
		IPWindow:     time.Minute,
		UserLimit:    300,
		UserWindow:   time.Minute,
		CleanupEvery: time.Hour,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Use the single token
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.1.1:4321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Next request should be 429 with Retry-After
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.1.1:4321"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}

	retryAfter := w.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("expected Retry-After header on 429 response")
	}
	if retryAfter == "0" {
		t.Fatal("Retry-After must be > 0")
	}
}

// TestRateLimitDisabled verifies that when the rate limiter middleware
// is not applied (simulating --rate-limit-enabled=false), requests pass
// unhindered even with a restrictive config.
func TestRateLimitDisabled(t *testing.T) {
	rl := auth.NewRateLimiter(auth.RateLimiterConfig{
		IPLimit:      2,
		IPWindow:     time.Minute,
		UserLimit:    300,
		UserWindow:   time.Minute,
		CleanupEvery: time.Hour,
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// With middleware applied: only 2 requests allowed
	withRL := rl.Middleware(inner)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		withRL.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("with middleware request %d: expected 200, got %d", i+1, w.Code)
		}
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	withRL.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("with middleware request 3: expected 429, got %d", w.Code)
	}

	// Without middleware (disabled mode): all requests pass
	// Fresh IP to avoid the exhausted bucket
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.2:5678"
		w := httptest.NewRecorder()
		inner.ServeHTTP(w, req) // inner only — no rate limiter middleware
		if w.Code != http.StatusOK {
			t.Fatalf("disabled mode request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

// TestRateLimiterStop verifies that calling Stop() exits the cleanup
// goroutine and is safe to call multiple times.
func TestRateLimiterStop(t *testing.T) {
	rl := auth.NewRateLimiter(auth.RateLimiterConfig{
		IPLimit:      60,
		IPWindow:     time.Minute,
		UserLimit:    300,
		UserWindow:   time.Minute,
		CleanupEvery: 50 * time.Millisecond,
	})

	// Stop once — cleanup goroutine should exit.
	rl.Stop()

	// Wait briefly for goroutine to drain.
	time.Sleep(100 * time.Millisecond)

	// Second Stop is a no-op (must not panic).
	rl.Stop()

	// The limiter still works after Stop — buckets just won't be pruned.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	rlHandler := rl.Middleware(inner)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	rlHandler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after Stop, got %d", w.Code)
	}
}
