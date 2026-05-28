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

type MetricsCollector struct {
	mu        sync.RWMutex
	startedAt time.Time
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
		logger:  logger,
		metrics: &MetricsCollector{startedAt: time.Now()},
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
	return mux
}

func (m *Monitoring) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

var _ kernel.Component = (*Monitoring)(nil)
