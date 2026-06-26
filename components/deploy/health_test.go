package deploy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// fakeHTTPDoer is a scripted HTTP client for testing probeHealth.
type fakeHTTPDoer struct {
	responses []fakeResponse
	callCount int
}

type fakeResponse struct {
	status int
	body   string
	err    error
}

func (f *fakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	if f.callCount >= len(f.responses) {
		return nil, fmt.Errorf("unexpected call #%d (only %d scripted)", f.callCount, len(f.responses))
	}
	resp := f.responses[f.callCount]
	f.callCount++
	if resp.err != nil {
		return nil, resp.err
	}
	return &http.Response{
		StatusCode: resp.status,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
	}, nil
}

func TestProbeHealth(t *testing.T) {
	t.Run("passes on first attempt", func(t *testing.T) {
		clock := &FakeClock{now: time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)}
		doer := &fakeHTTPDoer{
			responses: []fakeResponse{{status: 200, body: "ok"}},
		}
		cfg := ManifestHealthCheck{}.WithDefaults()

		result := probeHealth(context.Background(), doer, "http://localhost:9999", cfg, clock)

		if !result.OK {
			t.Fatal("expected OK=true")
		}
		if result.Attempts != 1 {
			t.Errorf("attempts = %d, want 1", result.Attempts)
		}
		if len(result.Probes) != 1 {
			t.Errorf("probes = %d, want 1", len(result.Probes))
		}
		// No sleep should have occurred
		if len(clock.Sleeps) != 0 {
			t.Errorf("sleeps = %d, want 0 (no retry)", len(clock.Sleeps))
		}
	})

	t.Run("passes on third attempt after two failures", func(t *testing.T) {
		clock := &FakeClock{now: time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)}
		doer := &fakeHTTPDoer{
			responses: []fakeResponse{
				{status: 503, body: "unavailable"},
				{status: 503, body: "still unavailable"},
				{status: 200, body: "ok"},
			},
		}
		cfg := ManifestHealthCheck{
			MaxRetries:      5,
			IntervalSeconds: 2,
		}.WithDefaults()

		result := probeHealth(context.Background(), doer, "http://localhost:9999", cfg, clock)

		if !result.OK {
			t.Fatal("expected OK=true after retries")
		}
		if result.Attempts != 3 {
			t.Errorf("attempts = %d, want 3", result.Attempts)
		}
		// Two sleeps: after attempt 0 and attempt 1
		if len(clock.Sleeps) != 2 {
			t.Errorf("sleeps = %d, want 2", len(clock.Sleeps))
		}
		if clock.Sleeps[0] != 2*time.Second {
			t.Errorf("sleep[0] = %v, want 2s", clock.Sleeps[0])
		}
		if clock.Sleeps[1] != 2*time.Second {
			t.Errorf("sleep[1] = %v, want 2s", clock.Sleeps[1])
		}
		if result.FirstFailureReason == "" {
			t.Error("expected first_failure_reason to be set")
		}
	})

	t.Run("exhausts retries on persistent failure", func(t *testing.T) {
		clock := &FakeClock{now: time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)}
		doer := &fakeHTTPDoer{
			responses: []fakeResponse{
				{status: 503, body: "down"},
				{status: 503, body: "down"},
				{status: 503, body: "down"},
			},
		}
		cfg := ManifestHealthCheck{
			MaxRetries:      3,
			IntervalSeconds: 1,
		}.WithDefaults()

		result := probeHealth(context.Background(), doer, "http://localhost:9999", cfg, clock)

		if result.OK {
			t.Fatal("expected OK=false")
		}
		if result.Attempts != 3 {
			t.Errorf("attempts = %d, want 3", result.Attempts)
		}
		// Two sleeps: after attempt 0 and attempt 1 (no sleep after last attempt)
		if len(clock.Sleeps) != 2 {
			t.Errorf("sleeps = %d, want 2", len(clock.Sleeps))
		}
		if result.FirstFailureReason == "" {
			t.Error("expected first_failure_reason")
		}
		// AvgResponseTimeMS may be 0 with fakeHTTPDoer (instant response);
		// just verify it's non-negative and the probe count matches.
		if result.AvgResponseTimeMS < 0 {
			t.Errorf("avg_response_time_ms = %d, want >= 0", result.AvgResponseTimeMS)
		}
		if len(result.Probes) != 3 {
			t.Errorf("probes = %d, want 3", len(result.Probes))
		}
	})

	t.Run("respects expected_body_contains", func(t *testing.T) {
		clock := &FakeClock{now: time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)}
		doer := &fakeHTTPDoer{
			responses: []fakeResponse{
				{status: 200, body: `{"status": "error"}`},
				{status: 200, body: `{"status": "ok"}`},
			},
		}
		cfg := ManifestHealthCheck{
			ExpectedBodyContains: "ok",
			MaxRetries:           3,
			IntervalSeconds:      1,
		}.WithDefaults()

		result := probeHealth(context.Background(), doer, "http://localhost:9999", cfg, clock)

		if !result.OK {
			t.Fatal("expected OK=true after body match on retry")
		}
		if result.Attempts != 2 {
			t.Errorf("attempts = %d, want 2", result.Attempts)
		}
	})

	t.Run("body mismatch fails even on 200", func(t *testing.T) {
		clock := &FakeClock{now: time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)}
		doer := &fakeHTTPDoer{
			responses: []fakeResponse{
				{status: 200, body: `{"status": "error"}`},
				{status: 200, body: `{"status": "error"}`},
				{status: 200, body: `{"status": "error"}`},
			},
		}
		cfg := ManifestHealthCheck{
			ExpectedBodyContains: "ok",
			MaxRetries:           3,
			IntervalSeconds:      1,
		}.WithDefaults()

		result := probeHealth(context.Background(), doer, "http://localhost:9999", cfg, clock)

		if result.OK {
			t.Fatal("expected OK=false on body mismatch")
		}
		if result.Attempts != 3 {
			t.Errorf("attempts = %d, want 3", result.Attempts)
		}
		if !strings.Contains(result.FirstFailureReason, "body missing") {
			t.Errorf("first_failure_reason = %q, want body missing", result.FirstFailureReason)
		}
	})

	t.Run("request error handled gracefully", func(t *testing.T) {
		clock := &FakeClock{now: time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)}
		doer := &fakeHTTPDoer{
			responses: []fakeResponse{
				{err: errors.New("connection refused")},
				{status: 200, body: "ok"},
			},
		}
		cfg := ManifestHealthCheck{
			MaxRetries:      3,
			IntervalSeconds: 1,
		}.WithDefaults()

		result := probeHealth(context.Background(), doer, "http://localhost:9999", cfg, clock)

		if !result.OK {
			t.Fatal("expected OK=true after error retry")
		}
		if result.Attempts != 2 {
			t.Errorf("attempts = %d, want 2", result.Attempts)
		}
	})

	t.Run("empty ExpectedBodyContains means no body assertion", func(t *testing.T) {
		clock := &FakeClock{now: time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)}
		doer := &fakeHTTPDoer{
			responses: []fakeResponse{{status: 200, body: "anything"}},
		}
		cfg := ManifestHealthCheck{
			ExpectedBodyContains: "",
		}.WithDefaults()

		result := probeHealth(context.Background(), doer, "http://localhost:9999", cfg, clock)

		if !result.OK {
			t.Fatal("expected OK=true when no body assertion")
		}
	})
}
