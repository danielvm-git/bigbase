package deploy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// httpDoer is the seam for probeHealth HTTP requests. *http.Client satisfies it.
type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// ProbeAttempt records a single health probe result.
type ProbeAttempt struct {
	Status     int    `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	Err        string `json:"err,omitempty"`
}

// HealthSummary is the persisted summary of a health check probe sequence.
type HealthSummary struct {
	ProbeCount         int    `json:"probe_count"`
	AvgResponseTimeMs  int64  `json:"avg_response_time_ms,omitempty"`
	FirstFailureReason string `json:"first_failure_reason,omitempty"`
}

// HealthResult captures the outcome of a health check probe sequence.
type HealthResult struct {
	OK                 bool           `json:"ok"`
	Attempts           int            `json:"attempts"`
	AvgResponseTimeMS  int64          `json:"avg_response_time_ms"`
	FirstFailureReason string         `json:"first_failure_reason,omitempty"`
	Probes             []ProbeAttempt `json:"probes,omitempty"`
}

// probeHealth runs the health check retry loop against baseURL+cfg.Path.
// It respects cfg.TimeoutSeconds per-request, sleeps cfg.IntervalSeconds
// between retries (via clk.Sleep), and stops after cfg.MaxRetries failures.
// A single successful (status match + optional body match) response returns
// immediately with OK=true.
func probeHealth(ctx context.Context, doer httpDoer, baseURL string, cfg ManifestHealthCheck, clk Clock) HealthResult {
	result := HealthResult{}
	url := baseURL + cfg.Path

	for attempt := 0; attempt < cfg.MaxRetries; attempt++ {
		probe := ProbeAttempt{}
		start := time.Now()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			probe.Err = err.Error()
			probe.DurationMS = time.Since(start).Milliseconds()
			result.Probes = append(result.Probes, probe)
			result.Attempts++
			if result.FirstFailureReason == "" {
				result.FirstFailureReason = fmt.Sprintf("request creation failed: %v", err)
			}
			if attempt < cfg.MaxRetries-1 {
				clk.Sleep(time.Duration(cfg.IntervalSeconds) * time.Second)
			}
			continue
		}

		resp, doErr := doer.Do(req)
		if doErr != nil {
			probe.Err = doErr.Error()
			probe.DurationMS = time.Since(start).Milliseconds()
			result.Probes = append(result.Probes, probe)
			result.Attempts++
			if result.FirstFailureReason == "" {
				result.FirstFailureReason = fmt.Sprintf("request failed: %v", doErr)
			}
			if attempt < cfg.MaxRetries-1 {
				clk.Sleep(time.Duration(cfg.IntervalSeconds) * time.Second)
			}
			continue
		}

		probe.Status = resp.StatusCode
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		probe.DurationMS = time.Since(start).Milliseconds()
		result.Probes = append(result.Probes, probe)
		result.Attempts++

		// Check status match
		statusOK := resp.StatusCode == cfg.ExpectedStatus

		// Check body match (if configured)
		bodyOK := true
		if cfg.ExpectedBodyContains != "" {
			bodyOK = strings.Contains(string(bodyBytes), cfg.ExpectedBodyContains)
		}

		if statusOK && bodyOK {
			result.OK = true
			// Compute average response time across all attempts
			var totalMS int64
			for _, p := range result.Probes {
				totalMS += p.DurationMS
			}
			if len(result.Probes) > 0 {
				result.AvgResponseTimeMS = totalMS / int64(len(result.Probes))
			}
			return result
		}

		// Record failure reason on first failing probe
		if result.FirstFailureReason == "" {
			switch {
			case !statusOK:
				result.FirstFailureReason = fmt.Sprintf("expected status %d, got %d", cfg.ExpectedStatus, resp.StatusCode)
			case !bodyOK:
				result.FirstFailureReason = fmt.Sprintf("body missing %q", cfg.ExpectedBodyContains)
			}
		}

		if attempt < cfg.MaxRetries-1 {
			clk.Sleep(time.Duration(cfg.IntervalSeconds) * time.Second)
		}
	}

	// All attempts exhausted — compute average response time for the failure summary
	var totalMS int64
	for _, p := range result.Probes {
		totalMS += p.DurationMS
	}
	if len(result.Probes) > 0 {
		result.AvgResponseTimeMS = totalMS / int64(len(result.Probes))
	}

	return result
}

