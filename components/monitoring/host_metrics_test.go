package monitoring_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/monitoring"
)

// TestHostMetrics verifies that the /api/monitoring/metrics endpoint includes a "host" key
// containing CPU%, memory, disk, and network statistics collected from the OS.
func TestHostMetrics(t *testing.T) {
	_, handler := setupMonitoring(t)

	// Allow collector one tick to populate host metrics (it runs on Start).
	time.Sleep(50 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/api/monitoring/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	raw, ok := body["host"]
	if !ok {
		t.Fatalf("expected 'host' key in metrics response, got keys: %v", keys(body))
	}

	var host monitoring.HostMetrics
	if err := json.Unmarshal(raw, &host); err != nil {
		t.Fatalf("unmarshal host metrics: %v", err)
	}

	t.Run("cpu_percent in [0,100]", func(t *testing.T) {
		if host.CPUPercent < 0 || host.CPUPercent > 100 {
			t.Fatalf("cpu_percent out of range: %f", host.CPUPercent)
		}
	})

	t.Run("memory fields non-negative", func(t *testing.T) {
		if host.MemUsedBytes < 0 {
			t.Fatalf("mem_used_bytes negative: %d", host.MemUsedBytes)
		}
		if host.MemTotalBytes <= 0 {
			t.Fatalf("mem_total_bytes must be positive, got: %d", host.MemTotalBytes)
		}
	})

	t.Run("disk fields non-negative", func(t *testing.T) {
		if host.DiskUsedBytes < 0 {
			t.Fatalf("disk_used_bytes negative: %d", host.DiskUsedBytes)
		}
		if host.DiskTotalBytes <= 0 {
			t.Fatalf("disk_total_bytes must be positive, got: %d", host.DiskTotalBytes)
		}
	})

	t.Run("network bytes non-negative", func(t *testing.T) {
		if host.NetBytesIn < 0 {
			t.Fatalf("net_bytes_in negative: %d", host.NetBytesIn)
		}
		if host.NetBytesOut < 0 {
			t.Fatalf("net_bytes_out negative: %d", host.NetBytesOut)
		}
	})
}

func keys(m map[string]json.RawMessage) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
