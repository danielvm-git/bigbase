package monitoring

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielvm/bigbase/components/internal/eventrecorder"
	"github.com/danielvm/bigbase/kernel"
)

const (
	perCategoryEventLimit = 20
	relatedEventsWindow   = 30 * time.Minute
)

// Diagnosis is an AI-generated deploy failure explanation (owned here for ECC).
type Diagnosis struct {
	Diagnosis string `json:"diagnosis"`
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
}

// CorrelatedEvent is a single related monitoring event.
type CorrelatedEvent struct {
	Hook      string         `json:"hook"`
	Data      map[string]any `json:"data"`
	Timestamp string         `json:"timestamp"`
}

// RelatedEvents bundles correlated events for a deployment window.
type RelatedEvents struct {
	DeployID    string                       `json:"deploy_id"`
	WindowStart string                       `json:"window_start"`
	WindowEnd   string                       `json:"window_end"`
	Events      map[string][]CorrelatedEvent `json:"events"`
}

var knownEventHooks = []string{
	"mutation", "request",
	"deploy.state_changed", "deploy.failed", "deploy.diagnosed",
	"alert.triggered", "alert.investigation_complete",
	"scaffold_repo",
}

func (m *Monitoring) initObservability() error {
	rec, err := eventrecorder.New(m.db)
	if err != nil {
		return err
	}
	m.recorder = rec
	if err := m.db.Migrate(`CREATE TABLE IF NOT EXISTS deploy_diagnoses (
		deploy_id TEXT PRIMARY KEY,
		diagnosis TEXT NOT NULL,
		model TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}
	if err := m.db.Migrate(`CREATE TABLE IF NOT EXISTS monitoring_alert_incidents (
		id TEXT PRIMARY KEY,
		rule_id TEXT NOT NULL,
		metric TEXT NOT NULL,
		value REAL NOT NULL DEFAULT 0,
		threshold REAL NOT NULL DEFAULT 0,
		operator TEXT NOT NULL DEFAULT 'gt',
		opened_at TEXT NOT NULL DEFAULT (datetime('now')),
		resolved_at TEXT DEFAULT NULL,
		triggered INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		return err
	}
	// Add org_id column for tenant isolation (BUG-143).
	if _, err := m.db.Exec(
		"ALTER TABLE monitoring_alert_incidents ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0"); err != nil {
		if !strings.Contains(err.Error(), "duplicate column") {
			m.logger.Warn("add org_id column to incidents", "error", err)
		}
	}
	if err := m.db.Migrate(`CREATE TABLE IF NOT EXISTS monitoring_investigations (
		id TEXT PRIMARY KEY,
		incident_id TEXT NOT NULL,
		report_json TEXT NOT NULL,
		ai_summary TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}
	// Add org_id column for tenant isolation (BUG-143).
	if _, err := m.db.Exec(
		"ALTER TABLE monitoring_investigations ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0"); err != nil {
		if !strings.Contains(err.Error(), "duplicate column") {
			m.logger.Warn("add org_id column to investigations", "error", err)
		}
	}
	return nil
}

func (m *Monitoring) wireObservabilityHooks(bus *kernel.EventBus) {
	if bus == nil {
		return
	}
	m.WithEventBus(bus, knownEventHooks)
	bus.Subscribe(kernel.HookDef{
		Name:     "deploy.failed",
		Priority: 500,
		Handler: func(_ *kernel.Context, ev kernel.Event) error {
			go m.onDeployFailed(ev.Data)
			return nil
		},
	})
	bus.Subscribe(kernel.HookDef{
		Name:     "alert.triggered",
		Priority: 500,
		Handler: func(_ *kernel.Context, ev kernel.Event) error {
			go m.onAlertTriggered(ev.Data)
			// Deliver to the configured notifier (SMTP email etc.) so an alert
			// actually reaches a human instead of only landing in the dashboard.
			// (Issue #178, gap #2.)
			go m.deliverAlert(ev.Data)
			return nil
		},
	})
}

func eventSiteID(data map[string]any) string {
	if data == nil {
		return ""
	}
	if v, ok := data["site_id"].(string); ok {
		return v
	}
	return ""
}

func (m *Monitoring) onDeployFailed(data map[string]any) {
	deployID, _ := data["deployment_id"].(string)
	siteID, _ := data["site_id"].(string)
	if deployID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	m.snapshotRelatedEvents(ctx, deployID, siteID)
	if m.llm != nil && m.llm.Enabled() {
		m.runDeployDiagnosis(ctx, deployID, data)
	}
}

func (m *Monitoring) snapshotRelatedEvents(ctx context.Context, deployID, siteID string) {
	if m.recorder == nil || m.db == nil {
		return
	}
	var createdAt string
	if err := m.db.QueryRowContext(ctx,
		`SELECT created_at FROM deployments WHERE id = ?`, deployID).Scan(&createdAt); err != nil {
		return
	}
	end, _ := time.Parse(time.RFC3339, createdAt)
	if end.IsZero() {
		end = time.Now().UTC()
	}
	start := end.Add(-relatedEventsWindow)
	bundle := m.correlateEvents(ctx, siteID, start, end)
	bundle.DeployID = deployID
	payload, err := json.Marshal(bundle)
	if err != nil {
		return
	}
	_, _ = m.db.ExecContext(ctx,
		`UPDATE deployments SET related_events_snapshot = ? WHERE id = ?`, string(payload), deployID)
}

func (m *Monitoring) correlateEvents(ctx context.Context, siteID string, start, end time.Time) RelatedEvents {
	out := RelatedEvents{
		WindowStart: start.UTC().Format(time.RFC3339),
		WindowEnd:   end.UTC().Format(time.RFC3339),
		Events: map[string][]CorrelatedEvent{
			"db_schema_changes": {},
			"repo_changes":      {},
			"data_mutations":    {},
			"traffic":           {},
		},
	}
	if m.recorder == nil {
		return out
	}
	events, err := m.recorder.Query(ctx, eventrecorder.Filter{
		SiteID:      siteID,
		WindowStart: start,
		WindowEnd:   end,
		Limit:       200,
	})
	if err != nil {
		return out
	}
	categoryCounts := map[string]int{}
	for _, ev := range events {
		cat := eventCategory(ev.Hook, ev.Data)
		if cat == "" {
			continue
		}
		if categoryCounts[cat] >= perCategoryEventLimit {
			continue
		}
		categoryCounts[cat]++
		out.Events[cat] = append(out.Events[cat], CorrelatedEvent{
			Hook:      ev.Hook,
			Data:      ev.Data,
			Timestamp: ev.Timestamp,
		})
	}
	return out
}

func eventCategory(hook string, data map[string]any) string {
	switch hook {
	case "scaffold_db":
		return "db_schema_changes"
	case "scaffold_repo":
		return "repo_changes"
	case "mutation":
		t, _ := data["type"].(string)
		if t == "scaffold_db" {
			return "db_schema_changes"
		}
		if t == "scaffold_function" {
			return "repo_changes"
		}
		return "data_mutations"
	case "request":
		return "traffic"
	default:
		return ""
	}
}

func BuildDiagnosisPrompt(appType, errMsg, buildLog string) string {
	const maxLines = 200
	lines := strings.Split(buildLog, "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	tail := strings.Join(lines, "\n")
	return fmt.Sprintf(`You are a deployment debugging assistant for BigBase, a BaaS platform.
App type: %s
Error: %s
Build log tail:
%s

Provide a concise diagnosis and actionable fix suggestions. Label as AI suggestion — verify before applying.`, appType, errMsg, tail)
}

func (m *Monitoring) runDeployDiagnosis(ctx context.Context, deployID string, data map[string]any) {
	appType, _ := data["app_type"].(string)
	errMsg, _ := data["error_message"].(string)
	buildLog, _ := data["build_log"].(string)
	prompt := BuildDiagnosisPrompt(appType, errMsg, buildLog)
	text, err := m.llm.Complete(ctx, prompt)
	if err != nil {
		m.logger.Warn("deploy diagnosis failed", "deploy_id", deployID, "error", err)
		return
	}
	model := m.llmModel()
	_, _ = m.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO deploy_diagnoses (deploy_id, diagnosis, model, created_at) VALUES (?, ?, ?, datetime('now'))`,
		deployID, text, model)
	if m.kctx != nil && m.kctx.Kernel != nil {
		_ = m.kctx.Kernel.EventBus().Emit(kernel.Event{
			Name: "deploy.diagnosed",
			Data: map[string]any{
				"deploy_id": deployID,
				"diagnosis": text,
				"model":     model,
			},
		}, m.kctx)
	}
}

func (m *Monitoring) llmModel() string {
	if m.llmModelName != "" {
		return m.llmModelName
	}
	return "deepseek-chat"
}

// GetDiagnosis returns a stored deploy failure diagnosis.
func (m *Monitoring) GetDiagnosis(ctx context.Context, deployID string) (Diagnosis, bool, error) {
	var d Diagnosis
	err := m.db.QueryRowContext(ctx,
		`SELECT diagnosis, model, created_at FROM deploy_diagnoses WHERE deploy_id = ?`, deployID).
		Scan(&d.Diagnosis, &d.Model, &d.CreatedAt)
	if err != nil {
		return d, false, nil
	}
	return d, true, nil
}

// GetRelatedEvents returns correlated events for a deployment.
func (m *Monitoring) GetRelatedEvents(ctx context.Context, deployID string) (RelatedEvents, bool, error) {
	var snapshot string
	err := m.db.QueryRowContext(ctx,
		`SELECT COALESCE(related_events_snapshot,'') FROM deployments WHERE id = ?`, deployID).Scan(&snapshot)
	if err != nil {
		return RelatedEvents{}, false, nil
	}
	if snapshot != "" {
		var rel RelatedEvents
		if json.Unmarshal([]byte(snapshot), &rel) == nil {
			return rel, true, nil
		}
	}
	var siteID, createdAt string
	if err := m.db.QueryRowContext(ctx,
		`SELECT COALESCE(site_id,''), created_at FROM deployments WHERE id = ?`, deployID).
		Scan(&siteID, &createdAt); err != nil {
		return RelatedEvents{}, false, nil
	}
	end, _ := time.Parse(time.RFC3339, createdAt)
	if end.IsZero() {
		end = time.Now().UTC()
	}
	rel := m.correlateEvents(ctx, siteID, end.Add(-relatedEventsWindow), end)
	rel.DeployID = deployID
	return rel, true, nil
}

func (m *Monitoring) handleIncidents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	// Require org_id for tenant isolation (BUG-143).
	orgID, ok := kernel.OrgIDFromContext(r.Context())
	if !ok {
		kernel.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "org_id required"})
		return
	}
	status := r.URL.Query().Get("status")
	query := `SELECT id, rule_id, metric, value, threshold, operator, opened_at, resolved_at FROM monitoring_alert_incidents WHERE org_id = ?`
	args := []any{orgID}
	if status == "open" {
		query += ` AND resolved_at IS NULL`
	}
	query += ` ORDER BY opened_at DESC LIMIT 100`
	rows, err := m.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()
	type incident struct {
		ID         string  `json:"id"`
		RuleID     string  `json:"rule_id"`
		Metric     string  `json:"metric"`
		Value      float64 `json:"value"`
		Threshold  float64 `json:"threshold"`
		Operator   string  `json:"operator"`
		OpenedAt   string  `json:"opened_at"`
		ResolvedAt *string `json:"resolved_at,omitempty"`
	}
	var items []incident
	for rows.Next() {
		var it incident
		var resolved sql.NullString
		if err := rows.Scan(&it.ID, &it.RuleID, &it.Metric, &it.Value, &it.Threshold, &it.Operator, &it.OpenedAt, &resolved); err != nil {
			continue
		}
		if resolved.Valid {
			it.ResolvedAt = &resolved.String
		}
		items = append(items, it)
	}
	if items == nil {
		items = []incident{}
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (m *Monitoring) handleIncidentInvestigation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/monitoring/incidents/")
	id = strings.TrimSuffix(id, "/investigation")
	id = strings.Trim(id, "/")
	if id == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}
	var reportJSON, aiSummary string
	err := m.db.QueryRowContext(r.Context(),
		`SELECT report_json, COALESCE(ai_summary,'') FROM monitoring_investigations WHERE incident_id = ? ORDER BY created_at DESC LIMIT 1`, id).
		Scan(&reportJSON, &aiSummary)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "no investigation"})
		return
	}
	var report map[string]any
	if err := json.Unmarshal([]byte(reportJSON), &report); err != nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if aiSummary != "" {
		report["ai_summary"] = aiSummary
	}
	kernel.WriteJSON(w, http.StatusOK, report)
}

func newObservabilityID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
