package monitoring

import (
	"testing"
)

// TestMetricValue verifies that metricValue maps metric names to the correct
// source field from HostMetrics or SystemMetrics. It uses a fixed struct
// literal for reproducibility — no real OS calls.
func TestMetricValue(t *testing.T) {
	host := HostMetrics{
		CPUPercent:     45.2,
		MemUsedBytes:   8_589_934_592,
		MemTotalBytes:  16_000_000_000,
		DiskUsedBytes:  50_000_000_000,
		DiskTotalBytes: 100_000_000_000,
	}
	sys := SystemMetrics{
		CPUPercent: 12.3,
		Goroutines: 42,
		MemoryMB:   256.0,
	}

	tests := []struct {
		metric   string
		expected float64
	}{
		{"cpu_percent", 45.2},
		{"mem_used_bytes", float64(host.MemUsedBytes)},
		{"mem_percent", float64(host.MemUsedBytes) / float64(host.MemTotalBytes) * 100},
		{"disk_used_bytes", float64(host.DiskUsedBytes)},
		{"disk_percent", float64(host.DiskUsedBytes) / float64(host.DiskTotalBytes) * 100},
		{"goroutines", float64(sys.Goroutines)},
		{"process_cpu_percent", sys.CPUPercent},
		{"process_mem_mb", sys.MemoryMB},
		{"unknown_metric", 0},
	}

	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			got := metricValue(tt.metric, host, sys)
			if got != tt.expected {
				t.Fatalf("metricValue(%q) = %f, want %f", tt.metric, got, tt.expected)
			}
		})
	}
}

func TestMetricValueEdgeCases(t *testing.T) {
	zeroHost := HostMetrics{}
	zeroSys := SystemMetrics{}

	t.Run("mem_percent with zero total returns 0", func(t *testing.T) {
		if got := metricValue("mem_percent", zeroHost, zeroSys); got != 0 {
			t.Fatalf("expected 0, got %f", got)
		}
	})

	t.Run("disk_percent with zero total returns 0", func(t *testing.T) {
		if got := metricValue("disk_percent", zeroHost, zeroSys); got != 0 {
			t.Fatalf("expected 0, got %f", got)
		}
	})
}

func TestEvalOperator(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		operator  string
		threshold float64
		expected  bool
	}{
		{"gt true", 100, "gt", 90, true},
		{"gt false (equal)", 90, "gt", 90, false},
		{"gt false", 80, "gt", 90, false},
		{"lt true", 50, "lt", 100, true},
		{"lt false (equal)", 100, "lt", 100, false},
		{"lt false", 150, "lt", 100, false},
		{"eq true", 42.0, "eq", 42.0, true},
		{"eq false", 43.0, "eq", 42.0, false},
		{"gte true (greater)", 100, "gte", 90, true},
		{"gte true (equal)", 90, "gte", 90, true},
		{"gte false", 80, "gte", 90, false},
		{"lte true (less)", 80, "lte", 90, true},
		{"lte true (equal)", 90, "lte", 90, true},
		{"lte false", 100, "lte", 90, false},
		{"unknown operator", 100, "ne", 90, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalOperator(tt.value, tt.operator, tt.threshold)
			if got != tt.expected {
				t.Fatalf("evalOperator(%f, %q, %f) = %v, want %v",
					tt.value, tt.operator, tt.threshold, got, tt.expected)
			}
		})
	}
}

func TestEmitAlertTriggeredNilContext(t *testing.T) {
	m := &Monitoring{kctx: nil, logger: &noopLogger{}}
	m.emitAlertTriggered("alert-1", "test alert", "cpu_percent", 95.0, 90.0, "gt", "incident-1")
}
