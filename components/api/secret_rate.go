package api

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/danielvm/bigbase/components/auth"
)

// secret_rate.go implements the e89s04 mutation rate-limit contract: at most
// 30 mutating secret requests per minute per authenticated actor and Project,
// rejected before persistence. It is a dedicated token bucket keyed on
// (actor, project) rather than the auth component's per-IP/per-user limiter,
// because the contract is per-actor-and-project.

// secretBucket is a single (actor, project) token bucket.
type secretBucket struct {
	tokens  int
	resetAt time.Time
}

// rateLimiterPruneThreshold bounds the bucket map size; once reached, expired
// buckets are swept on the next allow call.
const rateLimiterPruneThreshold = 2048

// secretRateLimiter is a concurrent-safe token bucket keyed on actor|project.
type secretRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*secretBucket
}

// newSecretRateLimiter builds a limiter granting limit tokens per window.
func newSecretRateLimiter(limit int, window time.Duration) *secretRateLimiter {
	return &secretRateLimiter{
		limit:   limit,
		window:  window,
		buckets: make(map[string]*secretBucket),
	}
}

// allow consumes one token from the (actor, project) bucket. It returns
// whether the request is within budget and, when denied, the Retry-After
// seconds until the bucket resets. When the bucket map grows past
// rateLimiterPruneThreshold, expired buckets are swept inline so memory stays
// bounded even under adversarial actor/project churn (CWE-400).
func (rl *secretRateLimiter) allow(actor, projectID string) (bool, int) {
	key := actor + "|" + projectID
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if len(rl.buckets) >= rateLimiterPruneThreshold {
		for k, b := range rl.buckets {
			if now.After(b.resetAt) {
				delete(rl.buckets, k)
			}
		}
	}

	b, ok := rl.buckets[key]
	if !ok {
		b = &secretBucket{tokens: rl.limit, resetAt: now.Add(rl.window)}
		rl.buckets[key] = b
	}
	if now.After(b.resetAt) {
		b.tokens = rl.limit
		b.resetAt = now.Add(rl.window)
	}
	if b.tokens <= 0 {
		retry := int(time.Until(b.resetAt).Seconds())
		if retry < 1 {
			retry = 1
		}
		return false, retry
	}
	b.tokens--
	return true, 0
}

// secretActorKey derives the authenticated actor identity used in the rate
// limit key. Authenticated users key on their user id; org API keys key on
// the organization id. It never includes caller-supplied values.
func secretActorKey(ctx context.Context) string {
	if userID, ok := auth.UserIDFromContext(ctx); ok && userID > 0 {
		return "user:" + strconv.FormatInt(userID, 10)
	}
	if orgID, ok := auth.OrgIDFromContext(ctx); ok && orgID > 0 {
		return "orgkey:" + strconv.FormatInt(orgID, 10)
	}
	return "anonymous"
}
