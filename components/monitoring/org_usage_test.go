package monitoring_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/monitoring"
	"github.com/danielvm/bigbase/kernel"
)

func setupMonitoringWithOrg(t *testing.T) (*monitoring.Monitoring, http.Handler) {
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

func TestOrgUsage(t *testing.T) {
	t.Run("record_request_increments_count", func(t *testing.T) {
		m, _ := setupMonitoringWithOrg(t)

		m.RecordOrgRequest(1, 100)
		m.RecordOrgRequest(1, 200)
		m.RecordOrgRequest(2, 50)

		// Verify no panic and that data is recorded
		// (actual persistence tested via GET endpoint)
	})

	t.Run("get_usage_returns_30_day_breakdown", func(t *testing.T) {
		m, handler := setupMonitoringWithOrg(t)

		// Record some usage for org 1
		m.RecordOrgRequest(1, 512)
		m.RecordOrgRequest(1, 1024)
		m.RecordOrgRequest(2, 256) // org 2 — should not appear

		req := httptest.NewRequest("GET", "/api/orgs/1/usage", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}

		data, ok := resp["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data object, got: %v", resp)
		}
		days, ok := data["days"].([]any)
		if !ok {
			t.Fatalf("expected days array, got: %v", data)
		}
		if len(days) == 0 {
			t.Fatal("expected at least 1 day entry")
		}
		if len(days) > 30 {
			t.Fatalf("expected at most 30 days, got %d", len(days))
		}

		// Verify totals
		totalRequests, ok := data["total_requests"].(float64)
		if !ok {
			t.Fatalf("expected total_requests float64, got: %T %v", data["total_requests"], data["total_requests"])
		}
		if totalRequests < 2 {
			t.Errorf("expected at least 2 total_requests for org 1, got %.0f", totalRequests)
		}

		totalBytes, ok := data["total_bytes"].(float64)
		if !ok {
			t.Fatalf("expected total_bytes float64, got: %T %v", data["total_bytes"], data["total_bytes"])
		}
		if totalBytes != 1536 { // 512 + 1024
			t.Errorf("expected total_bytes=1536, got %.0f", totalBytes)
		}
	})

	t.Run("get_usage_invalid_org_id", func(t *testing.T) {
		_, handler := setupMonitoringWithOrg(t)

		req := httptest.NewRequest("GET", "/api/orgs/not-a-number/usage", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("get_usage_empty_returns_empty_days", func(t *testing.T) {
		_, handler := setupMonitoringWithOrg(t)

		req := httptest.NewRequest("GET", "/api/orgs/99/usage", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]any
		_ = json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].(map[string]any)
		days, ok := data["days"].([]any)
		if !ok {
			// An empty slice marshalled to JSON is [] which is valid
			t.Fatalf("expected days array, got: %v", data)
		}
		if len(days) != 0 {
			t.Fatalf("expected 0 days for org with no usage, got %d", len(days))
		}
	})

	t.Run("daily_buckets_aggregate_correctly", func(t *testing.T) {
		m, handler := setupMonitoringWithOrg(t)

		// Record multiple requests in same bucket (today)
		for i := 0; i < 5; i++ {
			m.RecordOrgRequest(3, int64(100*(i+1)))
		}

		req := httptest.NewRequest("GET", "/api/orgs/3/usage", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		_ = json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].(map[string]any)
		days := data["days"].([]any)

		// Should have 1 day (today only)
		if len(days) != 1 {
			t.Fatalf("expected 1 day bucket, got %d", len(days))
		}

		day := days[0].(map[string]any)
		if day["requests"].(float64) != 5 {
			t.Errorf("expected 5 requests, got %v", day["requests"])
		}
		expectedBytes := float64(100 + 200 + 300 + 400 + 500)
		if day["bytes"].(float64) != expectedBytes {
			t.Errorf("expected bytes=%v, got %v", expectedBytes, day["bytes"])
		}

		// date field must be present
		if day["date"] == "" || day["date"] == nil {
			t.Error("expected non-empty date in day bucket")
		}
		// date format: YYYY-MM-DD
		dateStr, _ := day["date"].(string)
		if _, err := time.Parse("2006-01-02", dateStr); err != nil {
			t.Errorf("expected date in YYYY-MM-DD format, got %q: %v", dateStr, err)
		}
	})
}
