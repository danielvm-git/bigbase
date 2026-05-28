package monitoring_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/monitoring"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupMonitoring(t *testing.T) (*monitoring.Monitoring, http.Handler) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)
	m := monitoring.New(monitoring.Options{Logger: logger})
	k.Register(m)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return m, m.Handler()
}

func TestMonitoringImplementsComponent(t *testing.T) {
	var _ kernel.Component = (*monitoring.Monitoring)(nil)
}

func TestMonitoringName(t *testing.T) {
	m := &monitoring.Monitoring{}
	if got := m.Name(); got != "monitoring" {
		t.Fatalf("expected Name()='monitoring', got '%s'", got)
	}
}

func TestMonitoringHealthEndpoint(t *testing.T) {
	_, handler := setupMonitoring(t)
	req := httptest.NewRequest("GET", "/api/monitoring/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status 'ok', got '%s'", body["status"])
	}
}

func TestMonitoringMetricsEndpoint(t *testing.T) {
	_, handler := setupMonitoring(t)
	req := httptest.NewRequest("GET", "/api/monitoring/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	sys, ok := body["system"].(map[string]any)
	if !ok {
		t.Fatal("expected 'system' key in metrics response")
	}
	if _, ok := sys["goroutines"]; !ok {
		t.Fatal("expected 'goroutines' in system metrics")
	}
	if _, ok := sys["uptime_seconds"]; !ok {
		t.Fatal("expected 'uptime_seconds' in system metrics")
	}
	if _, ok := sys["memory_mb"]; !ok {
		t.Fatal("expected 'memory_mb' in system metrics")
	}

	reqs, ok := body["requests"].(map[string]any)
	if !ok {
		t.Fatal("expected 'requests' key in metrics response")
	}
	total, ok := reqs["total"].(float64)
	if !ok || total < 0 {
		t.Fatalf("expected non-negative requests total, got %v", reqs["total"])
	}
}

func TestMonitoringSystemMetrics(t *testing.T) {
	m, _ := setupMonitoring(t)
	metrics := m.SystemMetrics()

	if metrics.Goroutines <= 0 {
		t.Fatalf("expected positive goroutines, got %d", metrics.Goroutines)
	}
	if metrics.UptimeSeconds < 0 {
		t.Fatalf("expected non-negative uptime, got %f", metrics.UptimeSeconds)
	}
	if metrics.MemoryMB <= 0 {
		t.Fatalf("expected positive memory, got %f", metrics.MemoryMB)
	}
	if metrics.CPUPercent < 0 {
		t.Fatalf("expected non-negative cpu, got %f", metrics.CPUPercent)
	}
}
