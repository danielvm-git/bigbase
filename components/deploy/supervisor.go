package deploy

import (
	"math"
	"time"
)

// Restart policy constants — the in-process equivalent of Docker
// `restart: unless-stopped` (ADR 0004). Named constants, not magic numbers (G25);
// the Supervisor exposes them as flags.
const (
	backoffBase   = 1 * time.Second // first-attempt ceiling
	backoffFactor = 2.0             // exponential growth per attempt
	backoffCap    = 60 * time.Second

	crashLoopBurst  = 5                // restarts within window that trip "failed"
	crashLoopWindow = 60 * time.Second // trailing window for the burst count
)

// nextBackoff returns the delay before restart `attempt` (0-indexed) using full
// jitter: a value in [0, min(cap, base*factor^attempt)]. The jitter fraction
// `frac` (in [0,1]) is injected by the caller so this function stays pure and
// deterministic; production passes rand.Float64().
//
// Full jitter is load-bearing, not cosmetic: resume re-spawns the whole fleet at
// once, so un-jittered backoff would crash-and-retry in lockstep (thundering herd
// on the VPS). Jitter makes that failure mode unreachable by construction.
func nextBackoff(attempt int, frac float64) time.Duration {
	ceiling := float64(backoffBase) * math.Pow(backoffFactor, float64(attempt))
	if ceiling > float64(backoffCap) {
		ceiling = float64(backoffCap)
	}
	return time.Duration(frac * ceiling)
}

// isCrashLooping reports whether at least crashLoopBurst restarts fall within the
// trailing crashLoopWindow ending at now. Mirrors systemd StartLimitBurst /
// StartLimitIntervalSec (consistent with the systemd Isolator). Named predicate (G28):
// on trip the Supervisor marks the deployment failed, de-registers its host, and
// emits deploy.crash_looped.
func isCrashLooping(history []time.Time, now time.Time) bool {
	cutoff := now.Add(-crashLoopWindow)
	count := 0
	for _, t := range history {
		if !t.Before(cutoff) && !t.After(now) {
			count++
		}
	}
	return count >= crashLoopBurst
}
