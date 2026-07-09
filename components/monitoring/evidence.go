package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/danielvm/bigbase/components/internal/eventrecorder"
	"github.com/danielvm/bigbase/kernel"
)

// InvestigationReport is stored for alert incidents.
type InvestigationReport struct {
	IncidentID        string         `json:"incident_id"`
	AlertID           string         `json:"alert_id"`
	TriggeredAt       string         `json:"triggered_at"`
	Snapshot          map[string]any `json:"snapshot"`
	RecentDeployments []any          `json:"recent_deployments"`
	RecentEvents      []any          `json:"recent_events"`
	RecentLogs        []any          `json:"recent_logs"`
	AISummary         string         `json:"ai_summary,omitempty"`
}

type EvidenceScope struct {
	SiteID      string
	WindowStart time.Time
	WindowEnd   time.Time
	Metric      string
}

func (m *Monitoring) gatherEvidence(ctx context.Context, scope EvidenceScope) (InvestigationReport, error) {
	report := InvestigationReport{
		Snapshot: map[string]any{
			"system": m.SystemMetrics(),
			"host":   m.LatestHostMetrics(),
		},
		RecentDeployments: []any{},
		RecentEvents:      []any{},
		RecentLogs:        []any{},
	}
	if m.db == nil {
		return report, nil
	}
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, site_id, status, app_type, created_at FROM deployments
		 WHERE created_at >= datetime(?, '-30 minutes') ORDER BY created_at DESC LIMIT 20`,
		scope.WindowEnd.Format(time.RFC3339))
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var id, siteID, status, appType, createdAt string
			if rows.Scan(&id, &siteID, &status, &appType, &createdAt) == nil {
				report.RecentDeployments = append(report.RecentDeployments, map[string]string{
					"id": id, "site_id": siteID, "status": status, "app_type": appType, "created_at": createdAt,
				})
			}
		}
	}
	if m.recorder != nil {
		events, err := m.recorder.Query(ctx, eventrecorder.Filter{
			SiteID:      scope.SiteID,
			WindowStart: scope.WindowStart,
			WindowEnd:   scope.WindowEnd,
			Limit:       50,
		})
		if err == nil {
			for _, ev := range events {
				report.RecentEvents = append(report.RecentEvents, ev)
			}
		}
	}
	logRows, err := m.db.QueryContext(ctx,
		`SELECT id, level, message, created_at FROM monitoring_logs ORDER BY created_at DESC LIMIT 100`)
	if err == nil {
		defer func() { _ = logRows.Close() }()
		for logRows.Next() {
			var id, level, message, createdAt string
			if logRows.Scan(&id, &level, &message, &createdAt) == nil {
				report.RecentLogs = append(report.RecentLogs, map[string]string{
					"id": id, "level": level, "message": message, "created_at": createdAt,
				})
			}
		}
	}
	return report, nil
}

func (m *Monitoring) onAlertTriggered(data map[string]any) {
	incidentID, _ := data["incident_id"].(string)
	alertID, _ := data["alert_id"].(string)
	if incidentID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	now := time.Now().UTC()
	report, err := m.gatherEvidence(ctx, EvidenceScope{
		WindowStart: now.Add(-30 * time.Minute),
		WindowEnd:   now,
		Metric:      fmt.Sprintf("%v", data["metric"]),
	})
	if err != nil {
		m.logger.Error("gather evidence", "error", err)
		return
	}
	report.IncidentID = incidentID
	report.AlertID = alertID
	report.TriggeredAt = now.Format(time.RFC3339)
	if m.llm != nil && m.llm.Enabled() {
		summaryPrompt := fmt.Sprintf("Summarize this alert investigation evidence in 2-3 sentences:\n%+v", report)
		if text, err := m.llm.Complete(ctx, summaryPrompt); err == nil {
			report.AISummary = text
		}
	}
	payload, _ := json.Marshal(report)
	id, err := newObservabilityID()
	if err != nil {
		return
	}
	_, _ = m.db.ExecContext(ctx,
		`INSERT INTO monitoring_investigations (id, incident_id, report_json, ai_summary, created_at) VALUES (?, ?, ?, ?, datetime('now'))`,
		id, incidentID, string(payload), report.AISummary)
	m.trimInvestigations(ctx)
	if m.kctx != nil && m.kctx.Kernel != nil {
		_ = m.kctx.Kernel.EventBus().Emit(kernel.Event{
			Name: "alert.investigation_complete",
			Data: map[string]any{
				"incident_id": incidentID,
				"alert_id":    alertID,
			},
		}, m.kctx)
	}
}

func (m *Monitoring) trimInvestigations(ctx context.Context) {
	var count int
	if err := m.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM monitoring_investigations`).Scan(&count); err != nil {
		return
	}
	if count > 1000 {
		excess := count - 1000
		_, _ = m.db.ExecContext(ctx,
			`DELETE FROM monitoring_investigations WHERE id IN (
				SELECT id FROM monitoring_investigations ORDER BY created_at ASC LIMIT ?
			)`, excess)
	}
}

func (m *Monitoring) openOrGetIncident(ctx context.Context, ruleID, metric string, value, threshold float64, operator string) (string, bool) {
	var id string
	var triggered int
	err := m.db.QueryRowContext(ctx,
		`SELECT id, triggered FROM monitoring_alert_incidents WHERE rule_id = ? AND resolved_at IS NULL LIMIT 1`, ruleID).
		Scan(&id, &triggered)
	if err == nil && id != "" {
		return id, triggered == 1
	}
	newID, err := newObservabilityID()
	if err != nil {
		return "", false
	}
	_, err = m.db.ExecContext(ctx,
		`INSERT INTO monitoring_alert_incidents (id, rule_id, metric, value, threshold, operator, opened_at, triggered)
		 VALUES (?, ?, ?, ?, ?, ?, datetime('now'), 0)`,
		newID, ruleID, metric, value, threshold, operator)
	if err != nil {
		return "", false
	}
	return newID, false
}

func (m *Monitoring) markIncidentTriggered(ctx context.Context, incidentID string) {
	_, _ = m.db.ExecContext(ctx,
		`UPDATE monitoring_alert_incidents SET triggered = 1 WHERE id = ?`, incidentID)
}

func (m *Monitoring) resolveIncidents(ctx context.Context, ruleID string) {
	_, _ = m.db.ExecContext(ctx,
		`UPDATE monitoring_alert_incidents SET resolved_at = datetime('now') WHERE rule_id = ? AND resolved_at IS NULL`, ruleID)
}
