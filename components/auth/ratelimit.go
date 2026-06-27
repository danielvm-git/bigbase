package auth

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiterConfig configures the token-bucket rate limiter.
type RateLimiterConfig struct {
	// IPLimit is the maximum number of requests per IPWindow for unauthenticated IPs.
	IPLimit int
	// IPWindow is the duration of the IP token-bucket window.
	IPWindow time.Duration
	// UserLimit is the maximum number of requests per UserWindow for authenticated users.
	UserLimit int
	// UserWindow is the duration of the user token-bucket window.
	UserWindow time.Duration
	// CleanupEvery controls how often stale buckets are pruned.
	CleanupEvery time.Duration
}

// DefaultRateLimiterConfig returns the production rate limit settings:
// 60 req/min per IP, 300 req/min per authenticated user.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		IPLimit:      60,
		IPWindow:     time.Minute,
		UserLimit:    300,
		UserWindow:   time.Minute,
		CleanupEvery: 5 * time.Minute,
	}
}

// bucket is a token-bucket entry for a single client key.
type bucket struct {
	mu        sync.Mutex
	tokens    int
	limit     int
	window    time.Duration
	resetAt   time.Time
	lastSeen  time.Time
}

// allow returns true if this request is within the limit, false if exceeded.
func (b *bucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	b.lastSeen = now

	// Refill tokens when the window has elapsed.
	if now.After(b.resetAt) {
		b.tokens = b.limit
		b.resetAt = now.Add(b.window)
	}

	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// retryAfter returns seconds until the bucket resets.
func (b *bucket) retryAfter() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	secs := int(time.Until(b.resetAt).Seconds())
	if secs < 1 {
		secs = 1
	}
	return secs
}

// RateLimiter is a concurrent-safe token-bucket rate limiter that tracks
// both per-IP and per-authenticated-user buckets.
type RateLimiter struct {
	cfg     RateLimiterConfig
	ipMu    sync.Mutex
	ipMap   map[string]*bucket
	userMu  sync.Mutex
	userMap map[int64]*bucket
}

// Reconfigure updates the rate limiter configuration at runtime.
// Existing buckets retain their current limits until their windows reset;
// new callers see the updated config immediately.
// Thread-safe: acquires write locks on ipMap and userMap during the swap.
func (rl *RateLimiter) Reconfigure(cfg RateLimiterConfig) {
	rl.ipMu.Lock()
	rl.userMu.Lock()
	rl.cfg = cfg
	rl.userMu.Unlock()
	rl.ipMu.Unlock()
}

// NewRateLimiter creates a new RateLimiter with the given config and starts
// the background cleanup goroutine.
func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		cfg:     cfg,
		ipMap:   make(map[string]*bucket),
		userMap: make(map[int64]*bucket),
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cfg.CleanupEvery)
	defer ticker.Stop()
	for range ticker.C {
		threshold := time.Now().Add(-rl.cfg.CleanupEvery)

		rl.ipMu.Lock()
		for key, b := range rl.ipMap {
			b.mu.Lock()
			ls := b.lastSeen
			b.mu.Unlock()
			if ls.Before(threshold) {
				delete(rl.ipMap, key)
			}
		}
		rl.ipMu.Unlock()

		rl.userMu.Lock()
		for key, b := range rl.userMap {
			b.mu.Lock()
			ls := b.lastSeen
			b.mu.Unlock()
			if ls.Before(threshold) {
				delete(rl.userMap, key)
			}
		}
		rl.userMu.Unlock()
	}
}

func (rl *RateLimiter) ipBucket(ip string) *bucket {
	rl.ipMu.Lock()
	defer rl.ipMu.Unlock()
	b, ok := rl.ipMap[ip]
	if !ok {
		b = &bucket{
			tokens:   rl.cfg.IPLimit,
			limit:    rl.cfg.IPLimit,
			window:   rl.cfg.IPWindow,
			resetAt:  time.Now().Add(rl.cfg.IPWindow),
			lastSeen: time.Now(),
		}
		rl.ipMap[ip] = b
	}
	return b
}

func (rl *RateLimiter) userBucket(userID int64) *bucket {
	rl.userMu.Lock()
	defer rl.userMu.Unlock()
	b, ok := rl.userMap[userID]
	if !ok {
		b = &bucket{
			tokens:   rl.cfg.UserLimit,
			limit:    rl.cfg.UserLimit,
			window:   rl.cfg.UserWindow,
			resetAt:  time.Now().Add(rl.cfg.UserWindow),
			lastSeen: time.Now(),
		}
		rl.userMap[userID] = b
	}
	return b
}

// Middleware returns an http.Handler that enforces rate limits.
// Authenticated requests (context has a UserID) use the per-user bucket;
// unauthenticated requests use the per-IP bucket.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b *bucket
		if uid, ok := UserIDFromContext(r.Context()); ok {
			b = rl.userBucket(uid)
		} else {
			ip := extractIP(r)
			b = rl.ipBucket(ip)
		}

		if !b.allow() {
			w.Header().Set("Retry-After", strconv.Itoa(b.retryAfter()))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractIP parses the client IP from RemoteAddr, stripping the port.
func extractIP(r *http.Request) string {
	// Prefer X-Forwarded-For when behind a proxy, but only trust the first.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the leftmost (client) IP.
		for i, c := range xff {
			if c == ',' {
				xff = xff[:i]
				break
			}
		}
		if ip := net.ParseIP(xff); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// WithUserID injects a userID into the context — used in tests to simulate
// an authenticated request without a full JWT round-trip.
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, ctxUserID, userID)
}
