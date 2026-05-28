package monitoring

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}
func (noopLogger) Debug(msg string, args ...any) {}

type SystemMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      float64 `json:"memory_mb"`
	Goroutines    int     `json:"goroutines"`
	UptimeSeconds float64 `json:"uptime_seconds"`
}

type EndpointMetrics struct {
	Count       int64             `json:"count"`
	StatusCount map[int]int64     `json:"status_count"`
	Latencies   []float64         `json:"-"`
	AvgLatency  float64           `json:"avg_latency_ms"`
	P99Latency  float64           `json:"p99_latency_ms"`
}

type RequestMetrics struct {
	Total        int64                       `json:"total"`
	ByEndpoint   map[string]*EndpointMetrics `json:"by_endpoint"`
	ByStatus     map[int]int64               `json:"by_status"`
	AvgLatencyMs float64                     `json:"avg_latency_ms"`
}

type MetricsCollector struct {
	mu        sync.RWMutex
	startedAt time.Time
	requests  map[string]*EndpointMetrics
}

type Monitoring struct {
	logger   Logger
	metrics  *MetricsCollector
}

type Options struct {
	Logger Logger
}

func New(opts Options) *Monitoring {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	return &Monitoring{
		logger: logger,
		metrics: &MetricsCollector{
			startedAt: time.Now(),
			requests:  make(map[string]*EndpointMetrics),
		},
	}
}

func (m *Monitoring) Name() string                  { return "monitoring" }
func (m *Monitoring) Version() string               { return version }
func (m *Monitoring) Dependencies() []string         { return nil }
func (m *Monitoring) ConfigSchema() json.RawMessage { return nil }
func (m *Monitoring) Hooks() []kernel.HookDef        { return nil }
func (m *Monitoring) Init(ctx *kernel.Context, config json.RawMessage) error { return nil }
func (m *Monitoring) Start(ctx *kernel.Context) error { return nil }
func (m *Monitoring) Stop(ctx *kernel.Context) error  { return nil }

func (m *Monitoring) SystemMetrics() SystemMetrics {
	m.metrics.mu.RLock()
	uptime := time.Since(m.metrics.startedAt).Seconds()
	m.metrics.mu.RUnlock()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return SystemMetrics{
		Goroutines:    runtime.NumGoroutine(),
		UptimeSeconds: uptime,
		MemoryMB:      float64(mem.Alloc) / 1024 / 1024,
	}
}

func (m *Monitoring) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/monitoring/health", m.handleHealth)
	mux.HandleFunc("/api/monitoring/metrics", m.handleMetrics)
	return mux
}

func (m *Monitoring) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (m *Monitoring) handleMetrics(w http.ResponseWriter, r *http.Request) {
	sys := m.SystemMetrics()

	m.metrics.mu.RLock()
	total := int64(0)
	byStatus := make(map[int]int64)
	var allLatencies []float64

	for _, ep := range m.metrics.requests {
		total += ep.Count
		for code, count := range ep.StatusCount {
			byStatus[code] += count
		}
		allLatencies = append(allLatencies, ep.Latencies...)
	}

	endpoints := make(map[string]map[string]any)
	for path, ep := range m.metrics.requests {
		endpoints[path] = map[string]any{
			"count":         ep.Count,
			"status_count":  ep.StatusCount,
			"avg_latency_ms": ep.AvgLatency,
		}
	}
	m.metrics.mu.RUnlock()

	var avgLatency float64
	if len(allLatencies) > 0 {
		var sum float64
		for _, l := range allLatencies {
			sum += l
		}
		avgLatency = sum / float64(len(allLatencies))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"system": map[string]any{
			"cpu_percent":    sys.CPUPercent,
			"memory_mb":      sys.MemoryMB,
			"goroutines":     sys.Goroutines,
			"uptime_seconds": sys.UptimeSeconds,
		},
		"requests": map[string]any{
			"total":          total,
			"by_endpoint":    endpoints,
			"by_status":      byStatus,
			"avg_latency_ms": avgLatency,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

var _ kernel.Component = (*Monitoring)(nil)
