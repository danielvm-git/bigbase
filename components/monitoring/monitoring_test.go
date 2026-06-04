package monitoring_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
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
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	m := monitoring.New(monitoring.Options{DB: d, Logger: logger})
	k.Register(d)
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

func TestMonitoringMiddlewareRecordsMetrics(t *testing.T) {
	m, _ := setupMonitoring(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	})

	wrapped := m.Middleware(next)
	mux := http.NewServeMux()
	mux.Handle("/api/test", wrapped)
	mux.Handle("/api/monitoring/metrics", m.Handler())

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from test handler, got %d", w.Code)
	}

	metricsReq := httptest.NewRequest("GET", "/api/monitoring/metrics", nil)
	mw := httptest.NewRecorder()
	mux.ServeHTTP(mw, metricsReq)

	var body map[string]any
	if err := json.NewDecoder(mw.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	reqs, _ := body["requests"].(map[string]any)
	total, _ := reqs["total"].(float64)
	if total != 1 {
		t.Fatalf("expected 1 request, got %v", total)
	}

	byEndpoint, _ := reqs["by_endpoint"].(map[string]any)
	if _, ok := byEndpoint["/api/test"]; !ok {
		t.Fatalf("expected /api/test in by_endpoint, got %v", byEndpoint)
	}

	byStatus, _ := reqs["by_status"].(map[string]any)
	status200, ok := byStatus["200"].(float64)
	if !ok || status200 != 1 {
		t.Fatalf("expected 1 request with status 200, got %v", byStatus)
	}
}

func setupMonitoringWithDB(t *testing.T) (*monitoring.Monitoring, http.Handler, *db.DB) {
	t.Helper()
	logger := testLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	m := monitoring.New(monitoring.Options{DB: d, Logger: logger})
	k.Register(d)
	k.Register(m)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return m, m.Handler(), d
}

func TestMonitoringLogWriteAndSearch(t *testing.T) {
	_, handler, _ := setupMonitoringWithDB(t)

	entry := map[string]string{
		"level":   "info",
		"message": "test log entry",
	}
	body, _ := json.Marshal(entry)
	req := httptest.NewRequest("POST", "/api/monitoring/logs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	searchReq := httptest.NewRequest("GET", "/api/monitoring/logs", nil)
	sw := httptest.NewRecorder()
	handler.ServeHTTP(sw, searchReq)
	if sw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", sw.Code, sw.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(sw.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	logs, ok := result["data"].([]any)
	if !ok || len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %v", result)
	}
}

func TestMonitoringLogByID(t *testing.T) {
	_, handler, _ := setupMonitoringWithDB(t)

	entry := map[string]string{"level": "error", "message": "find me by id"}
	body, _ := json.Marshal(entry)
	req := httptest.NewRequest("POST", "/api/monitoring/logs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var created map[string]string
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	logID := created["id"]

	getReq := httptest.NewRequest("GET", "/api/monitoring/logs/"+logID, nil)
	gw := httptest.NewRecorder()
	handler.ServeHTTP(gw, getReq)
	if gw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", gw.Code, gw.Body.String())
	}
	var got map[string]any
	if err := json.NewDecoder(gw.Body).Decode(&got); err != nil {
		t.Fatalf("decode got: %v", err)
	}
	if got["id"] != logID {
		t.Fatalf("expected id '%s', got '%s'", logID, got["id"])
	}
}

func TestMonitoringLogSearchByQuery(t *testing.T) {
	_, handler, _ := setupMonitoringWithDB(t)

	for _, msg := range []string{"hello world", "error occurred", "hello again"} {
		entry := map[string]string{"message": msg}
		body, _ := json.Marshal(entry)
		req := httptest.NewRequest("POST", "/api/monitoring/logs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed: %d", w.Code)
		}
	}

	searchReq := httptest.NewRequest("GET", "/api/monitoring/logs?q=hello", nil)
	sw := httptest.NewRecorder()
	handler.ServeHTTP(sw, searchReq)
	if sw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", sw.Code)
	}
	var result map[string]any
	if err := json.NewDecoder(sw.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	logs, _ := result["data"].([]any)
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs matching 'hello', got %d", len(logs))
	}
}

func TestMonitoringAlertCreateMissingFields(t *testing.T) {
	_, handler, _ := setupMonitoringWithDB(t)

	body, _ := json.Marshal(map[string]string{"name": ""})
	req := httptest.NewRequest("POST", "/api/monitoring/alerts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing fields, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMonitoringAlertCreateAndList(t *testing.T) {
	_, handler, _ := setupMonitoringWithDB(t)

	alert := map[string]any{
		"name":      "high error rate",
		"metric":    "error_rate",
		"threshold": 5.0,
		"operator":  "gt",
		"enabled":   true,
	}
	body, _ := json.Marshal(alert)
	req := httptest.NewRequest("POST", "/api/monitoring/alerts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	listReq := httptest.NewRequest("GET", "/api/monitoring/alerts", nil)
	lw := httptest.NewRecorder()
	handler.ServeHTTP(lw, listReq)
	if lw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", lw.Code, lw.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(lw.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	alerts, ok := result["data"].([]any)
	if !ok || len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %v", result)
	}
}

func TestMonitoringDependenciesIncludeDB(t *testing.T) {
	m := &monitoring.Monitoring{}
	deps := m.Dependencies()
	found := false
	for _, d := range deps {
		if d == "db" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected monitoring to depend on db")
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

func TestSystemMetricsCPUPercentAfterSecondSample(t *testing.T) {
	m, _ := setupMonitoring(t)
	_ = m.SystemMetrics()
	time.Sleep(15 * time.Millisecond)
	for i := 0; i < 5000; i++ {
		_ = i * i
	}
	metrics := m.SystemMetrics()
	if metrics.CPUPercent <= 0 || metrics.CPUPercent > 100 {
		t.Fatalf("expected cpu percent in (0,100], got %f", metrics.CPUPercent)
	}
}
