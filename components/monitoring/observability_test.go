package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/internal/llm"
	"github.com/danielvm/bigbase/kernel"
)

type obsTestLogger struct{}

func (obsTestLogger) Info(string, ...any)  {}
func (obsTestLogger) Warn(string, ...any)  {}
func (obsTestLogger) Error(string, ...any) {}
func (obsTestLogger) Debug(string, ...any) {}

func setupObsMonitoring(t *testing.T, llmCfg llm.Config) (*Monitoring, *db.DB) {
	t.Helper()
	database := db.New(db.Options{Path: ":memory:", Logger: obsTestLogger{}})
	m := New(Options{DB: database, Logger: obsTestLogger{}, LLM: llmCfg})
	k := kernel.New(obsTestLogger{})
	k.Register(database)
	k.Register(m)
	if err := k.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return m, database
}

func TestEventPersistence(t *testing.T) {
	m, database := setupObsMonitoring(t, llm.Config{})
	if err := m.recorder.Record(context.Background(), "mutation", "site-1", map[string]any{"type": "create"}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	var count int
	if err := database.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM monitoring_events WHERE site_id = 'site-1'`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 event, got %d", count)
	}
}

func TestDeployFailedDiagnosis(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "Fix package.json"}}},
		})
	}))
	defer srv.Close()

	m, database := setupObsMonitoring(t, llm.Config{APIKey: "k", BaseURL: srv.URL, Model: "deepseek-chat"})
	_, _ = database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, branch, status, app_type, error_message, build_log, created_at)
		 VALUES ('dep-dx', 'r1', 'site-1', 'main', 'failed', 'node', 'exit 1', 'npm ERR!', datetime('now'))`)
	m.onDeployFailed(map[string]any{
		"deployment_id": "dep-dx", "site_id": "site-1", "app_type": "node",
		"error_message": "exit 1", "build_log": "npm ERR!",
	})
	time.Sleep(50 * time.Millisecond)
	diag, ok, err := m.GetDiagnosis(context.Background(), "dep-dx")
	if err != nil || !ok || diag.Diagnosis == "" {
		t.Fatalf("GetDiagnosis: ok=%v err=%v diag=%q", ok, err, diag.Diagnosis)
	}
}

func TestAlertIncidentDedup(t *testing.T) {
	m, _ := setupObsMonitoring(t, llm.Config{})
	ctx := context.Background()
	id1, triggered1 := m.openOrGetIncident(ctx, "rule-1", "cpu_percent", 90, 80, "gt")
	if id1 == "" || triggered1 {
		t.Fatalf("unexpected first incident id=%q triggered=%v", id1, triggered1)
	}
	id2, triggered2 := m.openOrGetIncident(ctx, "rule-1", "cpu_percent", 91, 80, "gt")
	if id2 != id1 || triggered2 {
		t.Fatalf("expected same open incident, got id2=%q triggered=%v", id2, triggered2)
	}
	m.markIncidentTriggered(ctx, id1)
	_, triggered3 := m.openOrGetIncident(ctx, "rule-1", "cpu_percent", 92, 80, "gt")
	if !triggered3 {
		t.Fatal("expected triggered flag on open incident")
	}
}

func TestGatherEvidence(t *testing.T) {
	m, database := setupObsMonitoring(t, llm.Config{})
	_, _ = database.ExecContext(context.Background(),
		`INSERT INTO monitoring_logs (id, level, message) VALUES ('l1', 'warn', 'cpu high')`)
	_, _ = database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, branch, status, app_type, created_at)
		 VALUES ('d1', 'r1', 'site-1', 'main', 'running', 'node', datetime('now'))`)
	report, err := m.gatherEvidence(context.Background(), EvidenceScope{
		WindowStart: time.Now().Add(-time.Hour),
		WindowEnd:   time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("gatherEvidence: %v", err)
	}
	if len(report.RecentLogs) == 0 {
		t.Fatal("expected logs in evidence")
	}
}

func TestAlertInvestigationTrigger(t *testing.T) {
	m, database := setupObsMonitoring(t, llm.Config{})
	ctx := context.Background()
	incidentID, _ := m.openOrGetIncident(ctx, "rule-x", "cpu_percent", 99, 1, "gt")
	m.onAlertTriggered(map[string]any{"incident_id": incidentID, "alert_id": "rule-x", "metric": "cpu_percent"})
	var count int
	if err := database.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM monitoring_investigations WHERE incident_id = ?`, incidentID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected investigation stored, got %d", count)
	}
}

func TestBuildDiagnosisPromptTruncates(t *testing.T) {
	var lines []string
	for i := 0; i < 300; i++ {
		lines = append(lines, "line")
	}
	prompt := BuildDiagnosisPrompt("node", "fail", strings.Join(lines, "\n"))
	if strings.Count(prompt, "line") > 210 {
		t.Fatalf("prompt should truncate log tail")
	}
}
