package monitoring

import (
	"context"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const alertCheckInterval = 30 * time.Second

// breachKey identifies an alert that is currently breaching its threshold.
// We track breach start times to support duration_seconds logic.
type breachKey string

// runAlertChecker checks all enabled alert rules every alertCheckInterval.
// When a rule is breached for >= duration_seconds, it emits an "alert.triggered" event.
func (m *Monitoring) runAlertChecker() {
	// Map from alert ID to the time at which the breach first started.
	breachStart := make(map[breachKey]time.Time)

	ticker := time.NewTicker(alertCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAlertRules(breachStart)
		case <-m.stopAlerts:
			return
		}
	}
}

// checkAlertRules evaluates every enabled alert rule against current host metrics.
func (m *Monitoring) checkAlertRules(breachStart map[breachKey]time.Time) {
	if m.db == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := m.db.QueryContext(ctx,
		"SELECT id, name, metric, threshold, operator, duration_seconds FROM monitoring_alerts WHERE enabled = 1")
	if err != nil {
		m.logger.Error("alert checker query", "error", err)
		return
	}
	defer func() { _ = rows.Close() }()

	host := m.LatestHostMetrics()
	sys := m.SystemMetrics()
	now := time.Now()

	var activeIDs []breachKey

	for rows.Next() {
		var id, name, metric, operator string
		var threshold float64
		var durationSec int64

		if err := rows.Scan(&id, &name, &metric, &threshold, &operator, &durationSec); err != nil {
			m.logger.Error("alert checker scan", "error", err)
			continue
		}

		value, known := m.metricValueWithSite(metric, host, sys)
		if !known {
			// Fail closed on unknown/typo'd metric names: log loudly and skip
			// the rule so it can neither fire forever (lt) nor never fire (gt).
			// (Issue #178, gap #3.)
			m.logger.Error("alert rule references unknown metric; skipping",
				"alert_id", id, "name", name, "metric", metric)
			delete(breachStart, breachKey(id))
			continue
		}
		breached := evalOperator(value, operator, threshold)
		key := breachKey(id)

		if !breached {
			delete(breachStart, key)
			m.resolveIncidents(ctx, id)
			continue
		}

		activeIDs = append(activeIDs, key)
		start, seen := breachStart[key]
		if !seen {
			breachStart[key] = now
			start = now
		}

		elapsed := now.Sub(start)
		if elapsed >= time.Duration(durationSec)*time.Second {
			incidentID, alreadyTriggered := m.openOrGetIncident(ctx, id, metric, value, threshold, operator)
			if incidentID != "" && !alreadyTriggered {
				m.emitAlertTriggered(id, name, metric, value, threshold, operator, incidentID)
				m.markIncidentTriggered(ctx, incidentID)
			}
		}
	}

	// Clean up entries for alerts that are no longer active (deleted or disabled).
	active := make(map[breachKey]struct{}, len(activeIDs))
	for _, k := range activeIDs {
		active[k] = struct{}{}
	}
	for k := range breachStart {
		if _, ok := active[k]; !ok {
			delete(breachStart, k)
		}
	}
}

// metricValue maps a metric name to its current observed value.
// It returns ok=false for unknown metric names so the caller can fail closed
// rather than silently treating the name as 0 (Issue #178, gap #3).
func metricValue(metric string, host HostMetrics, sys SystemMetrics) (float64, bool) {
	switch metric {
	case "cpu_percent":
		return host.CPUPercent, true
	case "mem_used_bytes":
		return float64(host.MemUsedBytes), true
	case "mem_percent":
		if host.MemTotalBytes > 0 {
			return float64(host.MemUsedBytes) / float64(host.MemTotalBytes) * 100, true
		}
		return 0, true
	case "disk_used_bytes":
		return float64(host.DiskUsedBytes), true
	case "disk_percent":
		if host.DiskTotalBytes > 0 {
			return float64(host.DiskUsedBytes) / float64(host.DiskTotalBytes) * 100, true
		}
		return 0, true
	case "goroutines":
		return float64(sys.Goroutines), true
	case "process_cpu_percent":
		return sys.CPUPercent, true
	case "process_mem_mb":
		return sys.MemoryMB, true
	default:
		return 0, false
	}
}

// metricValueWithSite resolves a metric the same way as metricValue, but also
// supports site-availability metrics (site_up, site_http_status) fed by the
// configured SiteStatusProvider. When no provider is wired (or it has not yet
// produced a reading), site metrics are treated as unknown so a rule
// referencing them never silently fires on 0.
func (m *Monitoring) metricValueWithSite(metric string, host HostMetrics, sys SystemMetrics) (float64, bool) {
	switch metric {
	case "site_up", "site_http_status":
		if m == nil || m.siteStatus == nil {
			return 0, false
		}
		st := m.siteStatus()
		// A provider that has never run returns the zero value. Treat that as
		// "unknown" so an alert rule cannot fire on an unset reading. A real
		// down-state probe should set HTTPStatus (e.g. 502/503) or Up=false
		// alongside a nonzero status.
		if !st.HasReading && st.HTTPStatus == 0 && !st.Up {
			return 0, false
		}
		switch metric {
		case "site_up":
			if st.Up {
				return 1, true
			}
			return 0, true
		case "site_http_status":
			if st.HTTPStatus == 0 {
				return 0, false
			}
			return float64(st.HTTPStatus), true
		}
	default:
		return metricValue(metric, host, sys)
	}
	return 0, false
}

// evalOperator evaluates value <op> threshold.
func evalOperator(value float64, operator string, threshold float64) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "lt":
		return value < threshold
	case "eq":
		return value == threshold
	case "gte":
		return value >= threshold
	case "lte":
		return value <= threshold
	default:
		return false
	}
}

// emitAlertTriggered fires the "alert.triggered" event on the kernel event bus.
func (m *Monitoring) emitAlertTriggered(id, name, metric string, value, threshold float64, operator, incidentID string) {
	m.logger.Warn("alert triggered",
		"alert_id", id,
		"incident_id", incidentID,
		"name", name,
		"metric", metric,
		"value", value,
		"threshold", threshold,
		"operator", operator,
	)

	if m.kctx == nil || m.kctx.Kernel == nil {
		return
	}

	evt := kernel.Event{
		Name: "alert.triggered",
		Data: map[string]any{
			"alert_id":    id,
			"incident_id": incidentID,
			"name":        name,
			"metric":      metric,
			"value":       value,
			"threshold":   threshold,
			"operator":    operator,
		},
	}
	if err := m.kctx.Kernel.EventBus().Emit(evt, m.kctx); err != nil {
		m.logger.Error("emit alert.triggered", "error", err)
	}
}
