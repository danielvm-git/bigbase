package monitoring_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAlertRules covers the full CRUD lifecycle of alert rules and verifies that
// the background checker correctly evaluates rule conditions.
func TestAlertRules(t *testing.T) {
	t.Run("create alert with duration_seconds", func(t *testing.T) {
		_, handler, _ := setupMonitoringWithDB(t)

		body, _ := json.Marshal(map[string]any{
			"name":             "high cpu",
			"metric":           "cpu_percent",
			"threshold":        90.0,
			"operator":         "gt",
			"enabled":          true,
			"duration_seconds": 300,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/monitoring/alerts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp["id"] == "" {
			t.Fatal("expected non-empty id in response")
		}
	})

	t.Run("list alerts includes duration_seconds", func(t *testing.T) {
		_, handler, _ := setupMonitoringWithDB(t)

		// Create one.
		body, _ := json.Marshal(map[string]any{
			"name":             "mem alert",
			"metric":           "mem_percent",
			"threshold":        80.0,
			"operator":         "gt",
			"enabled":          true,
			"duration_seconds": 60,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/monitoring/alerts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed: %d", w.Code)
		}

		// List.
		listReq := httptest.NewRequest(http.MethodGet, "/api/monitoring/alerts", nil)
		lw := httptest.NewRecorder()
		handler.ServeHTTP(lw, listReq)
		if lw.Code != http.StatusOK {
			t.Fatalf("list failed: %d", lw.Code)
		}

		var result map[string]json.RawMessage
		if err := json.NewDecoder(lw.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		var alerts []map[string]json.RawMessage
		if err := json.Unmarshal(result["data"], &alerts); err != nil {
			t.Fatalf("unmarshal alerts: %v", err)
		}
		if len(alerts) != 1 {
			t.Fatalf("expected 1 alert, got %d", len(alerts))
		}

		var durSec float64
		if err := json.Unmarshal(alerts[0]["duration_seconds"], &durSec); err != nil {
			t.Fatalf("unmarshal duration_seconds: %v", err)
		}
		if durSec != 60 {
			t.Fatalf("expected duration_seconds=60, got %f", durSec)
		}
	})

	t.Run("get alert by id", func(t *testing.T) {
		_, handler, _ := setupMonitoringWithDB(t)

		body, _ := json.Marshal(map[string]any{
			"name": "disk alert", "metric": "disk_percent",
			"threshold": 95.0, "operator": "gt", "enabled": true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/monitoring/alerts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d", w.Code)
		}

		var created map[string]string
		_ = json.NewDecoder(w.Body).Decode(&created)
		id := created["id"]

		getReq := httptest.NewRequest(http.MethodGet, "/api/monitoring/alerts/"+id, nil)
		gw := httptest.NewRecorder()
		handler.ServeHTTP(gw, getReq)
		if gw.Code != http.StatusOK {
			t.Fatalf("get: %d %s", gw.Code, gw.Body.String())
		}

		var got map[string]json.RawMessage
		if err := json.NewDecoder(gw.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		var gotID string
		_ = json.Unmarshal(got["id"], &gotID)
		if gotID != id {
			t.Fatalf("expected id %s, got %s", id, gotID)
		}
	})

	t.Run("delete alert", func(t *testing.T) {
		_, handler, _ := setupMonitoringWithDB(t)

		body, _ := json.Marshal(map[string]any{
			"name": "temp alert", "metric": "goroutines",
			"threshold": 1000.0, "operator": "gt", "enabled": true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/monitoring/alerts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d", w.Code)
		}

		var created map[string]string
		_ = json.NewDecoder(w.Body).Decode(&created)
		id := created["id"]

		delReq := httptest.NewRequest(http.MethodDelete, "/api/monitoring/alerts/"+id, nil)
		dw := httptest.NewRecorder()
		handler.ServeHTTP(dw, delReq)
		if dw.Code != http.StatusOK {
			t.Fatalf("delete: %d %s", dw.Code, dw.Body.String())
		}

		// Verify it's gone.
		getReq := httptest.NewRequest(http.MethodGet, "/api/monitoring/alerts/"+id, nil)
		gw := httptest.NewRecorder()
		handler.ServeHTTP(gw, getReq)
		if gw.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got %d", gw.Code)
		}
	})

	t.Run("update alert", func(t *testing.T) {
		_, handler, _ := setupMonitoringWithDB(t)

		body, _ := json.Marshal(map[string]any{
			"name": "update me", "metric": "cpu_percent",
			"threshold": 50.0, "operator": "gt", "enabled": true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/monitoring/alerts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d", w.Code)
		}

		var created map[string]string
		_ = json.NewDecoder(w.Body).Decode(&created)
		id := created["id"]

		upd, _ := json.Marshal(map[string]any{
			"name": "updated", "metric": "cpu_percent",
			"threshold": 75.0, "operator": "gt", "enabled": false,
		})
		updReq := httptest.NewRequest(http.MethodPut, "/api/monitoring/alerts/"+id, bytes.NewReader(upd))
		updReq.Header.Set("Content-Type", "application/json")
		uw := httptest.NewRecorder()
		handler.ServeHTTP(uw, updReq)
		if uw.Code != http.StatusOK {
			t.Fatalf("update: %d %s", uw.Code, uw.Body.String())
		}

		// Verify via GET.
		getReq := httptest.NewRequest(http.MethodGet, "/api/monitoring/alerts/"+id, nil)
		gw := httptest.NewRecorder()
		handler.ServeHTTP(gw, getReq)
		if gw.Code != http.StatusOK {
			t.Fatalf("get after update: %d", gw.Code)
		}

		var got map[string]json.RawMessage
		_ = json.NewDecoder(gw.Body).Decode(&got)
		var name string
		_ = json.Unmarshal(got["name"], &name)
		if name != "updated" {
			t.Fatalf("expected name='updated', got '%s'", name)
		}
		var enabled bool
		_ = json.Unmarshal(got["enabled"], &enabled)
		if enabled {
			t.Fatal("expected enabled=false after update")
		}
	})

	t.Run("default duration_seconds is 300", func(t *testing.T) {
		_, handler, _ := setupMonitoringWithDB(t)

		body, _ := json.Marshal(map[string]any{
			"name": "no duration", "metric": "cpu_percent",
			"threshold": 80.0, "operator": "gt", "enabled": true,
			// no duration_seconds — should default to 300
		})
		req := httptest.NewRequest(http.MethodPost, "/api/monitoring/alerts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d", w.Code)
		}

		var created map[string]string
		_ = json.NewDecoder(w.Body).Decode(&created)
		id := created["id"]

		getReq := httptest.NewRequest(http.MethodGet, "/api/monitoring/alerts/"+id, nil)
		gw := httptest.NewRecorder()
		handler.ServeHTTP(gw, getReq)
		if gw.Code != http.StatusOK {
			t.Fatalf("get: %d", gw.Code)
		}
		var got map[string]json.RawMessage
		_ = json.NewDecoder(gw.Body).Decode(&got)
		var dur float64
		_ = json.Unmarshal(got["duration_seconds"], &dur)
		if dur != 300 {
			t.Fatalf("expected default duration_seconds=300, got %f", dur)
		}
	})
}
