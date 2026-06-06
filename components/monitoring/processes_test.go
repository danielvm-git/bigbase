package monitoring_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestProcessesEndpoint verifies GET /api/monitoring/processes returns a "processes" key
// with at least the current BigBase process listed.
func TestProcessesEndpoint(t *testing.T) {
	_, handler := setupMonitoring(t)

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{"GET returns process list", http.MethodGet, http.StatusOK},
		{"POST not allowed", http.MethodPost, http.StatusMethodNotAllowed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/monitoring/processes", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}

	t.Run("response contains processes key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/monitoring/processes", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		var body map[string]json.RawMessage
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		raw, ok := body["processes"]
		if !ok {
			t.Fatalf("expected 'processes' key in response, got: %v", body)
		}

		var processes []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &processes); err != nil {
			t.Fatalf("unmarshal processes: %v", err)
		}
		if len(processes) == 0 {
			t.Fatal("expected at least one process in the list")
		}

		proc := processes[0]
		t.Run("has pid field", func(t *testing.T) {
			if _, ok := proc["pid"]; !ok {
				t.Fatal("expected pid field")
			}
			var pid int
			if err := json.Unmarshal(proc["pid"], &pid); err != nil {
				t.Fatalf("unmarshal pid: %v", err)
			}
			if pid != os.Getpid() {
				t.Fatalf("expected pid %d, got %d", os.Getpid(), pid)
			}
		})

		t.Run("has name field", func(t *testing.T) {
			if _, ok := proc["name"]; !ok {
				t.Fatal("expected name field")
			}
		})

		t.Run("has cpu_pct field", func(t *testing.T) {
			if _, ok := proc["cpu_pct"]; !ok {
				t.Fatal("expected cpu_pct field")
			}
		})

		t.Run("has mem_mb field", func(t *testing.T) {
			if _, ok := proc["mem_mb"]; !ok {
				t.Fatal("expected mem_mb field")
			}
		})

		t.Run("has status field", func(t *testing.T) {
			if _, ok := proc["status"]; !ok {
				t.Fatal("expected status field")
			}
		})
	})
}
