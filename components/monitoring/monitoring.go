package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

type DBer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Migrate(migration string) error
}

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

type LogEntry struct {
	ID        string `json:"id"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

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
	db      DBer
	logger  Logger
	metrics *MetricsCollector
}

type Options struct {
	DB     DBer
	Logger Logger
}

func New(opts Options) *Monitoring {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	return &Monitoring{
		db:     opts.DB,
		logger: logger,
		metrics: &MetricsCollector{
			startedAt: time.Now(),
			requests:  make(map[string]*EndpointMetrics),
		},
	}
}

func (m *Monitoring) Name() string                  { return "monitoring" }
func (m *Monitoring) Version() string               { return version }
func (m *Monitoring) Dependencies() []string         { return []string{"db"} }
func (m *Monitoring) ConfigSchema() json.RawMessage { return nil }
func (m *Monitoring) Hooks() []kernel.HookDef        { return nil }
func (m *Monitoring) Init(ctx *kernel.Context, config json.RawMessage) error { return nil }
func (m *Monitoring) Start(ctx *kernel.Context) error {
	if m.db == nil {
		return nil
	}
	return m.db.Migrate(`CREATE TABLE IF NOT EXISTS monitoring_logs (
		id TEXT PRIMARY KEY,
		level TEXT NOT NULL DEFAULT 'info',
		message TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
}
func (m *Monitoring) Stop(ctx *kernel.Context) error { return nil }

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (m *Monitoring) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)
		latency := time.Since(start).Seconds() * 1000

		path := r.URL.Path
		m.metrics.mu.Lock()
		ep, ok := m.metrics.requests[path]
		if !ok {
			ep = &EndpointMetrics{
				StatusCount: make(map[int]int64),
				Latencies:   make([]float64, 0, 100),
			}
			m.metrics.requests[path] = ep
		}
		ep.Count++
		ep.StatusCount[rec.statusCode]++
		ep.Latencies = append(ep.Latencies, latency)
		if len(ep.Latencies) > 1000 {
			ep.Latencies = ep.Latencies[len(ep.Latencies)-500:]
		}
		{
			var sum float64
			for _, l := range ep.Latencies {
				sum += l
			}
			ep.AvgLatency = sum / float64(len(ep.Latencies))
		}
		m.metrics.mu.Unlock()
	})
}

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
	mux.HandleFunc("/api/monitoring/logs", m.handleLogs)
	mux.HandleFunc("/api/monitoring/logs/", m.handleLogByID)
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

func (m *Monitoring) handleLogs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		m.handleLogCreate(w, r)
	case "GET":
		m.handleLogSearch(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (m *Monitoring) handleLogCreate(w http.ResponseWriter, r *http.Request) {
	if m.db == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db not configured"})
		return
	}
	var entry struct {
		Level   string `json:"level"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if entry.Level == "" {
		entry.Level = "info"
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	_, err := m.db.ExecContext(r.Context(),
		"INSERT INTO monitoring_logs (id, level, message, created_at) VALUES (?, ?, ?, datetime('now'))",
		id, entry.Level, entry.Message)
	if err != nil {
		m.logger.Error("insert log", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (m *Monitoring) handleLogSearch(w http.ResponseWriter, r *http.Request) {
	if m.db == nil {
		writeJSON(w, http.StatusOK, map[string]any{"data": []LogEntry{}})
		return
	}
	q := r.URL.Query().Get("q")
	var rows *sql.Rows
	var err error
	if q != "" {
		rows, err = m.db.QueryContext(r.Context(),
			"SELECT id, level, message, created_at FROM monitoring_logs WHERE message LIKE ? ORDER BY created_at DESC LIMIT 100",
			"%"+q+"%")
	} else {
		rows, err = m.db.QueryContext(r.Context(),
			"SELECT id, level, message, created_at FROM monitoring_logs ORDER BY created_at DESC LIMIT 100")
	}
	if err != nil {
		m.logger.Error("search logs", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer rows.Close()

	logs := make([]LogEntry, 0)
	for rows.Next() {
		var l LogEntry
		if err := rows.Scan(&l.ID, &l.Level, &l.Message, &l.CreatedAt); err != nil {
			m.logger.Error("scan log", "error", err)
			continue
		}
		logs = append(logs, l)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": logs})
}

func (m *Monitoring) handleLogByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/monitoring/logs/")
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if m.db == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	var l LogEntry
	err := m.db.QueryRowContext(r.Context(),
		"SELECT id, level, message, created_at FROM monitoring_logs WHERE id = ?", id).
		Scan(&l.ID, &l.Level, &l.Message, &l.CreatedAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "log not found"})
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

var _ kernel.Component = (*Monitoring)(nil)
