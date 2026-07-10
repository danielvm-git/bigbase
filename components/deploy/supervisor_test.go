package deploy

import (
	"testing"
	"time"
)

// nextBackoff implements full jitter: a value in [0, min(cap, base*factor^attempt)].
// The jitter fraction is injected (frac in [0,1]) so the function is pure and
// deterministic; production supplies rand.Float64(). Defaults: base 1s, factor 2,
// cap 60s. See ADR 0004.
func TestNextBackoff(t *testing.T) {
	cases := []struct {
		attempt int
		frac    float64
		want    time.Duration
	}{
		{attempt: 0, frac: 0, want: 0},                 // floor is always 0
		{attempt: 0, frac: 1, want: 1 * time.Second},   // min(60s, 1s*2^0) = 1s
		{attempt: 1, frac: 1, want: 2 * time.Second},   // 1s*2^1 = 2s
		{attempt: 3, frac: 1, want: 8 * time.Second},   // 1s*2^3 = 8s
		{attempt: 2, frac: 0.5, want: 2 * time.Second}, // 0.5 * 4s = 2s
		{attempt: 6, frac: 1, want: 60 * time.Second},  // 1s*2^6 = 64s, capped to 60s
		{attempt: 40, frac: 1, want: 60 * time.Second}, // no overflow at large attempt
	}
	for _, tc := range cases {
		if got := nextBackoff(tc.attempt, tc.frac); got != tc.want {
			t.Errorf("nextBackoff(%d, %v) = %v, want %v", tc.attempt, tc.frac, got, tc.want)
		}
	}
}

// isCrashLooping trips when >= burst (5) restarts fall within the trailing
// window (60s) ending at now. Mirrors systemd StartLimitBurst/IntervalSec.
func TestIsCrashLooping(t *testing.T) {
	base := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	now := base.Add(60 * time.Second)
	at := func(secs ...int) []time.Time {
		h := make([]time.Time, 0, len(secs))
		for _, s := range secs {
			h = append(h, base.Add(time.Duration(s)*time.Second))
		}
		return h
	}
	cases := []struct {
		name    string
		history []time.Time
		want    bool
	}{
		{"empty history", nil, false},
		{"four within window", at(5, 10, 20, 30), false},                     // 4 < burst
		{"five within window", at(5, 10, 20, 30, 40), true},                  // 5 == burst
		{"five but spread beyond window", at(-120, -110, 10, 20, 30), false}, // only 3 within 60s
	}
	for _, tc := range cases {
		if got := isCrashLooping(tc.history, now); got != tc.want {
			t.Errorf("isCrashLooping(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}
