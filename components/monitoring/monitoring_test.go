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
