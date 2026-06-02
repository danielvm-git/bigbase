# Slice 14: Monitoring — "See Metrics"

**type:** epic  
**status:** done  
**verify:** Dashboard shows CPU, requests, error rates

## Purpose

Metrics collection, structured logging, and uptime monitoring. Grafana-like dashboard built into the admin UI.

## Scope

- Request metrics (count, latency, status codes per endpoint)
- System metrics (CPU, memory, disk, goroutines)
- Structured JSON logs with search
- Uptime monitoring and health checks
- Real-time metrics dashboard in admin UI
- Alert thresholds (configurable)
- Log retention policy

## Design Decisions

- Metrics stored in-memory with periodic flush to SQLite
- Prometheus exposition format for `/metrics` endpoint
- Logs stored in SQLite with full-text search
- 15-second metric collection interval
- 30-day log retention (configurable)
- Dashboard auto-refresh every 5 seconds

## Metrics Endpoint

```jsonc
// GET /metrics
{
  "requests": {
    "total": 15234,
    "by_endpoint": {
      "/api/auth/login": 1234,
      "/api/collections/posts": 567
    },
    "by_status": {
      "2xx": 14000,
      "4xx": 1000,
      "5xx": 234
    },
    "avg_latency_ms": 12.5,
    "p99_latency_ms": 150
  },
  "system": {
    "cpu_percent": 23.4,
    "memory_mb": 45.2,
    "disk_mb": 128,
    "goroutines": 42,
    "uptime_seconds": 86400
  }
}
```

## Implementation Plan

### components/monitoring/monitoring.go

```go
type Monitoring struct {
    db       *db.DB
    logger   Logger
    metrics  *MetricsCollector
    alerts   *AlertManager
}

type MetricsCollector struct {
    mu          sync.RWMutex
    requests    map[string]*EndpointMetrics
    system      *SystemMetrics
    startedAt   time.Time
}

type EndpointMetrics struct {
    Count       int64
    StatusCount map[int]int64
    Latencies   []float64
    LastSeen    time.Time
}

type Alert struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    Metric    string `json:"metric"`
    Threshold float64 `json:"threshold"`
    Operator  string `json:"operator"` // gt, lt, eq
    Enabled   bool   `json:"enabled"`
}
```

### API Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/monitoring/metrics` | Yes | Get current metrics |
| GET | `/api/monitoring/logs` | Yes | Search logs |
| GET | `/api/monitoring/logs/:id` | Yes | Get log details |
| GET | `/api/monitoring/alerts` | Yes | List alerts |
| POST | `/api/monitoring/alerts` | Yes | Create alert |
| GET | `/api/monitoring/health` | No | Basic health check |

## Verify

```bash
# Health check
curl http://localhost:9999/api/monitoring/health

# Get metrics (authenticated)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:9999/api/monitoring/metrics

# Search logs
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:9999/api/monitoring/logs?q=error&since=24h"

# Admin UI dashboard shows real-time charts
open http://localhost:9999/admin/#/monitoring
```
