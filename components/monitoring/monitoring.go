package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/danielvm/bigbase/components/internal/eventrecorder"
	"github.com/danielvm/bigbase/components/internal/llm"
	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type Alert struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Metric          string  `json:"metric"`
	Threshold       float64 `json:"threshold"`
	Operator        string  `json:"operator"`
	Enabled         bool    `json:"enabled"`
	DurationSeconds int64   `json:"duration_seconds"`
}

type LogEntry struct {
	ID        string `json:"id"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	OrgID     int64  `json:"org_id"`
	CreatedAt string `json:"created_at"`
}

type SystemMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      float64 `json:"memory_mb"`
	Goroutines    int     `json:"goroutines"`
	UptimeSeconds float64 `json:"uptime_seconds"`
}

type EndpointMetrics struct {
	Count       int64         `json:"count"`
	StatusCount map[int]int64 `json:"status_count"`
	Latencies   []float64     `json:"-"`
	AvgLatency  float64       `json:"avg_latency_ms"`
	P99Latency  float64       `json:"p99_latency_ms"`
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
	host      HostMetrics
}

type Monitoring struct {
	db      DBer
	logger  kernel.Logger
	metrics *MetricsCollector
	stream  *eventStream

	cpuMu        sync.Mutex
	cpuSampleAt  time.Time
	cpuTotalNano int64

	stopHost     chan struct{}
	stopAlerts   chan struct{}
	lifecycleMu  sync.Mutex
	workers      sync.WaitGroup
	kctx         *kernel.Context // set in Start; used by alert checker to emit events
	recorder     *eventrecorder.Recorder
	llm          *llm.Client
	llmModelName string

	// siteStatus feeds the site_up / site_http_status metrics (Issue #178).
	// Optional: when nil, site metrics are treated as unknown.
	siteStatus SiteStatusProvider
	// alertNotifier delivers alert.triggered events to a human/channel
	// (e.g. SMTP email). Optional: when nil, alerts only hit the dashboard +
	// investigation pipeline. (Issue #178, gap #2.)
	alertNotifier AlertNotifier
}

var _ kernel.Component = (*Monitoring)(nil)

type Options struct {
	DB     DBer
	Logger kernel.Logger
	LLM    llm.Config
}

func New(opts Options) *Monitoring {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	llmCfg, modelName := llmConfigFromOptions(opts)
	return &Monitoring{
		db:     opts.DB,
		logger: logger,
		metrics: &MetricsCollector{
			startedAt: time.Now(),
			requests:  make(map[string]*EndpointMetrics),
		},
		llm:          llm.New(llmCfg),
		llmModelName: modelName,
	}
}

func llmConfigFromOptions(opts Options) (llm.Config, string) {
	cfg := opts.LLM
	if cfg.APIKey == "" && cfg.BaseURL == "" && cfg.Model == "" {
		cfg = llm.ConfigFromEnv()
	}
	model := cfg.Model
	if model == "" {
		model = "deepseek-chat"
	}
	return cfg, model
}

const hostCollectInterval = 15 * time.Second

func (m *Monitoring) Name() string           { return "monitoring" }
func (m *Monitoring) Version() string        { return version }
func (m *Monitoring) Dependencies() []string { return []string{"db"} }

func (m *Monitoring) Init(ctx *kernel.Context, config json.RawMessage) error { return nil }
func (m *Monitoring) Start(ctx *kernel.Context) error {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	m.stopWorkersLocked()
	m.stream = newEventStream()
	if m.db != nil {
		if err := m.initObservability(); err != nil {
			return err
		}
	}
	if ctx != nil && ctx.Kernel != nil {
		m.wireObservabilityHooks(ctx.Kernel.EventBus())
	}
	if m.db != nil {
		if err := m.db.Migrate(`CREATE TABLE IF NOT EXISTS monitoring_logs (
			id TEXT PRIMARY KEY,
			level TEXT NOT NULL DEFAULT 'info',
			message TEXT NOT NULL,
			org_id INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`); err != nil {
			return err
		}
		// Add org_id column for tenant isolation on DBs created before it (BUG-143).
		if _, err := m.db.Exec(
			"ALTER TABLE monitoring_logs ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0"); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				m.logger.Warn("add org_id column to logs", "error", err)
			}
		}
		if err := m.db.Migrate(`CREATE TABLE IF NOT EXISTS monitoring_alerts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			metric TEXT NOT NULL,
			threshold REAL NOT NULL DEFAULT 0,
			operator TEXT NOT NULL DEFAULT 'gt',
			enabled INTEGER NOT NULL DEFAULT 1,
			duration_seconds INTEGER NOT NULL DEFAULT 300
		)`); err != nil {
			return err
		}
		// Ensure duration_seconds column exists for DBs created before this migration.
		if _, err := m.db.Exec(
			"ALTER TABLE monitoring_alerts ADD COLUMN duration_seconds INTEGER NOT NULL DEFAULT 300"); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				m.logger.Warn("add duration_seconds column", "error", err)
			}
		}
		// Add org_id column for tenant isolation (BUG-143).
		if _, err := m.db.Exec(
			"ALTER TABLE monitoring_alerts ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0"); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				m.logger.Warn("add org_id column to alerts", "error", err)
			}
		}
		if err := m.migrateOrgUsageTable(); err != nil {
			return err
		}
	}

	m.kctx = ctx
	m.stopHost = make(chan struct{})
	m.stopAlerts = make(chan struct{})
	m.workers.Add(2)
	go func() {
		defer m.workers.Done()
		m.runHostCollector()
	}()
	go func() {
		defer m.workers.Done()
		m.runAlertChecker()
	}()
	return nil
}

func (m *Monitoring) Stop(ctx *kernel.Context) error {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	m.stopWorkersLocked()
	return nil
}

func (m *Monitoring) stopWorkersLocked() {
	if m.stopHost != nil {
		close(m.stopHost)
	}
	if m.stopAlerts != nil {
		close(m.stopAlerts)
	}
	m.workers.Wait()
	m.stopHost = nil
	m.stopAlerts = nil
}

func (m *Monitoring) runHostCollector() {
	m.storeHostMetrics(collectHostMetricsNonBlocking(m.logger))
	ticker := time.NewTicker(hostCollectInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.storeHostMetrics(collectHostMetrics(m.logger))
		case <-m.stopHost:
			return
		}
	}
}

func (m *Monitoring) storeHostMetrics(h HostMetrics) {
	m.metrics.mu.Lock()
	m.metrics.host = h
	m.metrics.mu.Unlock()
}

func (m *Monitoring) LatestHostMetrics() HostMetrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()
	return m.metrics.host
}

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

func (m *Monitoring) processCPUTime() time.Duration {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		m.logger.Warn("read process CPU time failed", "error", err)
		return 0
	}
	user := time.Duration(ru.Utime.Sec)*time.Second + time.Duration(ru.Utime.Usec)*time.Microsecond
	sys := time.Duration(ru.Stime.Sec)*time.Second + time.Duration(ru.Stime.Usec)*time.Microsecond
	return user + sys
}

func (m *Monitoring) sampleCPUPercent() float64 {
	total := m.processCPUTime()
	now := time.Now()
	totalNano := total.Nanoseconds()

	m.cpuMu.Lock()
	defer m.cpuMu.Unlock()

	if m.cpuSampleAt.IsZero() {
		m.cpuSampleAt = now
		m.cpuTotalNano = totalNano
		return 0
	}

	elapsed := now.Sub(m.cpuSampleAt).Seconds()
	if elapsed <= 0 {
		return 0
	}

	delta := float64(totalNano - m.cpuTotalNano)
	m.cpuSampleAt = now
	m.cpuTotalNano = totalNano

	cpus := runtime.NumCPU()
	if cpus < 1 {
		cpus = 1
	}
	pct := (delta / 1e9) / elapsed * 100 / float64(cpus)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct
}

func (m *Monitoring) SystemMetrics() SystemMetrics {
	m.metrics.mu.RLock()
	uptime := time.Since(m.metrics.startedAt).Seconds()
	m.metrics.mu.RUnlock()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return SystemMetrics{
		CPUPercent:    m.sampleCPUPercent(),
		Goroutines:    runtime.NumGoroutine(),
		UptimeSeconds: uptime,
		MemoryMB:      float64(mem.Alloc) / 1024 / 1024,
	}
}

func (m *Monitoring) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/monitoring/health", m.handleHealth)
	mux.HandleFunc("/api/monitoring/metrics", m.handleMetrics)
	mux.HandleFunc("/api/monitoring/metrics/stream", m.handleMetricsStream)
	mux.HandleFunc("/api/monitoring/metrics/prometheus", m.handlePrometheusMetrics)
	mux.HandleFunc("/api/monitoring/logs", m.handleLogs)
	mux.HandleFunc("/api/monitoring/logs/", m.handleLogByID)
	mux.HandleFunc("/api/monitoring/alerts", m.handleAlerts)
	mux.HandleFunc("GET /api/orgs/{id}/usage", m.handleOrgUsage)
	mux.HandleFunc("/api/monitoring/events", m.handleSSEEvents)
	mux.HandleFunc("/api/monitoring/incidents", m.handleIncidents)
	mux.HandleFunc("/api/monitoring/incidents/", m.handleIncidentInvestigation)
	mux.HandleFunc("/api/monitoring/alerts/", m.handleAlertByID)
	mux.HandleFunc("/api/monitoring/processes", m.handleProcesses)
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
			"count":          ep.Count,
			"status_count":   ep.StatusCount,
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

	kernel.WriteJSON(w, http.StatusOK, map[string]any{
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
		"host": m.LatestHostMetrics(),
	})
}

// prometheusSnapshot holds a point-in-time view of metrics for Prometheus export.
type prometheusSnapshot struct {
	byStatus      map[int]int64
	avgLatency    float64
	p99Latency    float64
	endpointStats map[string]endpointMetric
	sys           SystemMetrics
}

type endpointMetric struct {
	count        int64
	avgLatencyMs float64
}

// collectPrometheusSnapshot locks metrics, aggregates data, and returns a snapshot.
func (m *Monitoring) collectPrometheusSnapshot() prometheusSnapshot {
	s := prometheusSnapshot{
		byStatus:      make(map[int]int64),
		endpointStats: make(map[string]endpointMetric),
		sys:           m.SystemMetrics(),
	}

	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	var allLatencies []float64
	for path, ep := range m.metrics.requests {
		for code, count := range ep.StatusCount {
			s.byStatus[code] += count
		}
		allLatencies = append(allLatencies, ep.Latencies...)
		s.endpointStats[path] = endpointMetric{
			count:        ep.Count,
			avgLatencyMs: ep.AvgLatency,
		}
	}

	if len(allLatencies) > 0 {
		var sum float64
		for _, l := range allLatencies {
			sum += l
		}
		s.avgLatency = sum / float64(len(allLatencies))

		sorted := make([]float64, len(allLatencies))
		copy(sorted, allLatencies)
		sort.Float64s(sorted)
		idx := int(float64(len(sorted)) * 0.99)
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		s.p99Latency = sorted[idx]
	}

	return s
}

func (m *Monitoring) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)

	s := m.collectPrometheusSnapshot()

	writePromCounter(w, "bigbase_requests_total", "Total HTTP requests processed", s.byStatus)
	writePromGauge(w, "bigbase_request_duration_ms", "Request duration in milliseconds", s.avgLatency)
	writePromGauge(w, "bigbase_request_duration_p99_ms", "Request P99 latency in milliseconds", s.p99Latency)
	writePromGauge(w, "bigbase_goroutines", "Number of goroutines", float64(s.sys.Goroutines))
	writePromGauge(w, "bigbase_memory_mb", "Memory usage in MB", s.sys.MemoryMB)
	writePromGauge(w, "bigbase_cpu_percent", "CPU usage percent", s.sys.CPUPercent)

	for path, ep := range s.endpointStats {
		writePromLinef(w, "bigbase_endpoint_requests_total{path=\"%s\"} %d\n", path, ep.count)
		if ep.avgLatencyMs > 0 {
			writePromLinef(w, "bigbase_endpoint_duration_ms{path=\"%s\"} %.2f\n", path, ep.avgLatencyMs)
		}
	}
}

// writePromLine is a helper that writes a line to the response, discarding errors.
func writePromLine(w http.ResponseWriter, s string) {
	_, _ = w.Write([]byte(s))
}

// writePromLinef is a helper that formats and writes a line, discarding errors.
func writePromLinef(w http.ResponseWriter, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

// writePromCounter writes a Prometheus counter metric with optional labels.
func writePromCounter(w http.ResponseWriter, name, help string, values map[int]int64) {
	writePromLine(w, fmt.Sprintf("# HELP %s %s\n", name, help))
	writePromLine(w, fmt.Sprintf("# TYPE %s counter\n", name))
	for labelValue, count := range values {
		writePromLinef(w, "%s{status=\"%d\"} %d\n", name, labelValue, count)
	}
	writePromLine(w, "\n")
}

// writePromGauge writes a single-value Prometheus gauge metric.
func writePromGauge(w http.ResponseWriter, name, help string, value float64) {
	writePromLine(w, fmt.Sprintf("# HELP %s %s\n", name, help))
	writePromLine(w, fmt.Sprintf("# TYPE %s gauge\n", name))
	writePromLinef(w, "%s %.2f\n", name, value)
	writePromLine(w, "\n")
}

func (m *Monitoring) handleLogs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		m.handleLogCreate(w, r)
	case "GET":
		m.handleLogSearch(w, r)
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (m *Monitoring) handleLogCreate(w http.ResponseWriter, r *http.Request) {
	if m.db == nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "db not configured"})
		return
	}
	// Require org_id for tenant isolation (BUG-143).
	orgID, ok := kernel.OrgIDFromContext(r.Context())
	if !ok {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "org_id required"})
		return
	}
	var entry struct {
		Level   string `json:"level"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if entry.Level == "" {
		entry.Level = "info"
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	_, err := m.db.ExecContext(r.Context(),
		"INSERT INTO monitoring_logs (id, level, message, org_id, created_at) VALUES (?, ?, ?, ?, datetime('now'))",
		id, entry.Level, entry.Message, orgID)
	if err != nil {
		m.logger.Error("insert log", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (m *Monitoring) handleLogSearch(w http.ResponseWriter, r *http.Request) {
	if m.db == nil {
		kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": []LogEntry{}, "next_cursor": "", "has_more": false})
		return
	}
	// Require org_id for tenant isolation (BUG-143).
	orgID, ok := kernel.OrgIDFromContext(r.Context())
	if !ok {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "org_id required"})
		return
	}

	// limit: default 100, clamp to [1, 500].
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 500 {
		limit = 500
	}
	cursor := r.URL.Query().Get("cursor")
	q := r.URL.Query().Get("q")

	// Keyset pagination on the monotonic id: WHERE org_id = ? [AND message LIKE ?]
	// [AND id < cursor] ORDER BY id DESC. Fetch limit+1 to detect has_more.
	query := "SELECT id, level, message, org_id, created_at FROM monitoring_logs WHERE org_id = ?"
	args := []any{orgID}
	if q != "" {
		query += " AND message LIKE ?"
		args = append(args, "%"+q+"%")
	}
	if cursor != "" {
		query += " AND id < ?"
		args = append(args, cursor)
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit+1)

	rows, err := m.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		m.logger.Error("search logs", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	logs := make([]LogEntry, 0, limit)
	for rows.Next() {
		var l LogEntry
		if err := rows.Scan(&l.ID, &l.Level, &l.Message, &l.OrgID, &l.CreatedAt); err != nil {
			m.logger.Error("scan log", "error", err)
			continue
		}
		logs = append(logs, l)
	}

	hasMore := len(logs) > limit
	nextCursor := ""
	if hasMore {
		logs = logs[:limit]
		nextCursor = logs[len(logs)-1].ID
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{
		"data":        logs,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

func (m *Monitoring) handleLogByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/monitoring/logs/")
	if id == "" || strings.Contains(id, "/") {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if m.db == nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	// Require org_id and scope the lookup to it (BUG-143): a log from another
	// org must be indistinguishable from a nonexistent one (404).
	orgID, ok := kernel.OrgIDFromContext(r.Context())
	if !ok {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "org_id required"})
		return
	}
	var l LogEntry
	err := m.db.QueryRowContext(r.Context(),
		"SELECT id, level, message, org_id, created_at FROM monitoring_logs WHERE id = ? AND org_id = ?", id, orgID).
		Scan(&l.ID, &l.Level, &l.Message, &l.OrgID, &l.CreatedAt)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "log not found"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, l)
}

func (m *Monitoring) handleAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		m.handleAlertList(w, r)
	case "POST":
		m.handleAlertCreate(w, r)
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (m *Monitoring) handleAlertCreate(w http.ResponseWriter, r *http.Request) {
	if m.db == nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "db not configured"})
		return
	}
	// Require org_id for tenant isolation (BUG-143).
	orgID, ok := kernel.OrgIDFromContext(r.Context())
	if !ok {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "org_id required"})
		return
	}
	var alert struct {
		Name            string  `json:"name"`
		Metric          string  `json:"metric"`
		Threshold       float64 `json:"threshold"`
		Operator        string  `json:"operator"`
		Enabled         bool    `json:"enabled"`
		DurationSeconds int64   `json:"duration_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if alert.Name == "" || alert.Metric == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name and metric are required"})
		return
	}
	if alert.DurationSeconds <= 0 {
		alert.DurationSeconds = 300 // default 5 minutes
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	enabled := 0
	if alert.Enabled {
		enabled = 1
	}
	_, err := m.db.ExecContext(r.Context(),
		"INSERT INTO monitoring_alerts (id, name, metric, threshold, operator, enabled, duration_seconds, org_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, alert.Name, alert.Metric, alert.Threshold, alert.Operator, enabled, alert.DurationSeconds, orgID)
	if err != nil {
		m.logger.Error("insert alert", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (m *Monitoring) handleAlertList(w http.ResponseWriter, r *http.Request) {
	if m.db == nil {
		kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": []Alert{}})
		return
	}
	// Require org_id for tenant isolation (BUG-143).
	orgID, ok := kernel.OrgIDFromContext(r.Context())
	if !ok {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "org_id required"})
		return
	}
	rows, err := m.db.QueryContext(r.Context(),
		"SELECT id, name, metric, threshold, operator, enabled, duration_seconds FROM monitoring_alerts WHERE org_id = ? ORDER BY name", orgID)
	if err != nil {
		m.logger.Error("list alerts", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	alerts := make([]Alert, 0)
	for rows.Next() {
		var a Alert
		var enabled int
		if err := rows.Scan(&a.ID, &a.Name, &a.Metric, &a.Threshold, &a.Operator, &enabled, &a.DurationSeconds); err != nil {
			m.logger.Error("scan alert", "error", err)
			continue
		}
		a.Enabled = enabled == 1
		alerts = append(alerts, a)
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": alerts})
}

// handleMetricsStream, handleAlertByID, handleProcesses are implemented in separate files.
