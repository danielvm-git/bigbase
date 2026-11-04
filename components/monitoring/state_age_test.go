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
	"github.com/danielvm/bigbase/kernel"
)

var stateAgeTestNow = time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)

func newStateAgeDB(t *testing.T) *db.DB {
	t.Helper()
	database := db.New(db.Options{Path: ":memory:", Logger: kernel.NoopLogger{}})
	if err := database.Init(nil, nil); err != nil {
		t.Fatalf("db init: %v", err)
	}
	if err := database.Start(nil); err != nil {
		t.Fatalf("db start: %v", err)
	}
	t.Cleanup(func() { _ = database.Stop(nil) })
	return database
}

func newStateAgeFixture(t *testing.T) (*Monitoring, DBer) {
	t.Helper()
	database := newStateAgeDB(t)
	for _, schema := range []string{
		`CREATE TABLE deployments (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE cici_runs (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			started_at TEXT NOT NULL
		)`,
		`CREATE TABLE monitoring_alerts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			metric TEXT NOT NULL,
			threshold REAL NOT NULL DEFAULT 0,
			operator TEXT NOT NULL DEFAULT 'gt',
			enabled INTEGER NOT NULL DEFAULT 1,
			duration_seconds INTEGER NOT NULL DEFAULT 300
		)`,
	} {
		if _, err := database.ExecContext(context.Background(), schema); err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}
	m := New(Options{DB: database, Logger: kernel.NoopLogger{}})
	m.nowFn = func() time.Time { return stateAgeTestNow }
	return m, database
}

func seedDeployment(t *testing.T, d DBer, id, status string, age time.Duration) {
	t.Helper()
	if _, err := d.ExecContext(context.Background(),
		`INSERT INTO deployments (id, status, created_at) VALUES (?, ?, ?)`,
		id, status, stateAgeTestNow.Add(-age).Format(time.RFC3339)); err != nil {
		t.Fatalf("seed deployment %s: %v", id, err)
	}
}

func seedCiciRun(t *testing.T, d DBer, id, status string, age time.Duration) {
	t.Helper()
	if _, err := d.ExecContext(context.Background(),
		`INSERT INTO cici_runs (id, status, started_at) VALUES (?, ?, ?)`,
		id, status, stateAgeTestNow.Add(-age).Format(time.RFC3339)); err != nil {
		t.Fatalf("seed cici run %s: %v", id, err)
	}
}

func findReading(report StateAgeReport, machine, state string) (StateAgeReading, bool) {
	for _, r := range report.ByState {
		if r.Machine == machine && r.State == state {
			return r, true
		}
	}
	return StateAgeReading{}, false
}

func TestStateAgeReportCoversNonTerminalStates(t *testing.T) {
	m, d := newStateAgeFixture(t)

	seedDeployment(t, d, "dep-pending", "pending", 10*time.Minute)
	seedDeployment(t, d, "dep-building", "building", 45*time.Minute)
	seedDeployment(t, d, "dep-running", "running", 2*time.Hour)
	seedDeployment(t, d, "dep-failed", "failed", 3*time.Hour)

	seedCiciRun(t, d, "run-running", "running", 30*time.Minute)
	seedCiciRun(t, d, "run-queued", "queued", 5*time.Minute)
	seedCiciRun(t, d, "run-legacy-pending", "pending", 15*time.Minute)
	seedCiciRun(t, d, "run-success", "success", 4*time.Hour)

	report := m.stateAgeReport(context.Background())

	tests := []struct {
		machine string
		state   string
		age     float64
		rows    int
		oldest  string
	}{
		{"deployment", "pending", 600, 1, "dep-pending"},
		{"deployment", "building", 2700, 1, "dep-building"},
		{"deployment", "running", 7200, 1, "dep-running"},
		{"deployment", "deploying", 0, 0, ""},
		{"deployment", "draining", 0, 0, ""},
		{"cici_run", "running", 1800, 1, "run-running"},
		{"cici_run", "queued", 900, 2, "run-legacy-pending"},
	}
	for _, tt := range tests {
		r, ok := findReading(report, tt.machine, tt.state)
		if !ok {
			t.Fatalf("missing reading %s/%s", tt.machine, tt.state)
		}
		if r.AgeSeconds != tt.age {
			t.Errorf("%s/%s age = %f, want %f", tt.machine, tt.state, r.AgeSeconds, tt.age)
		}
		if r.Rows != tt.rows {
			t.Errorf("%s/%s rows = %d, want %d", tt.machine, tt.state, r.Rows, tt.rows)
		}
		if r.OldestID != tt.oldest {
			t.Errorf("%s/%s oldest = %q, want %q", tt.machine, tt.state, r.OldestID, tt.oldest)
		}
	}

	if _, ok := findReading(report, "deployment", "failed"); ok {
		t.Error("terminal deployment state must not appear in the report")
	}
	if _, ok := findReading(report, "cici_run", "success"); ok {
		t.Error("terminal cici state must not appear in the report")
	}

	if report.Worst == nil {
		t.Fatal("expected a worst offender")
	}
	if report.Worst.Machine != "deployment" || report.Worst.State != "running" {
		t.Fatalf("worst = %s/%s, want deployment/running", report.Worst.Machine, report.Worst.State)
	}
	if report.Worst.AgeSeconds != 7200 || report.Worst.OldestID != "dep-running" {
		t.Fatalf("worst reading = %+v", *report.Worst)
	}
}

func TestStateAgeReportParsesSQLiteDatetime(t *testing.T) {
	m, d := newStateAgeFixture(t)

	if _, err := d.ExecContext(context.Background(),
		`INSERT INTO deployments (id, status, created_at) VALUES (?, ?, ?)`,
		"dep-sqlite-ts", "deploying", stateAgeTestNow.Add(-20*time.Minute).Format("2006-01-02 15:04:05")); err != nil {
		t.Fatalf("seed: %v", err)
	}

	report := m.stateAgeReport(context.Background())
	r, ok := findReading(report, "deployment", "deploying")
	if !ok {
		t.Fatal("missing deployment/deploying reading")
	}
	if r.AgeSeconds != 1200 {
		t.Fatalf("age = %f, want 1200", r.AgeSeconds)
	}
}

func TestStateAgeReportMissingTableIsUnavailable(t *testing.T) {
	database := newStateAgeDB(t)
	m := New(Options{DB: database, Logger: kernel.NoopLogger{}})
	m.nowFn = func() time.Time { return stateAgeTestNow }

	report := m.stateAgeReport(context.Background())
	if len(report.ByState) != 0 {
		t.Fatalf("expected no readings without tables, got %d", len(report.ByState))
	}
	if len(report.Unavailable) != len(stateMachineSpecs) {
		t.Fatalf("unavailable = %v, want all machines", report.Unavailable)
	}

	if _, ok := m.metricValueWithSite("deployment_state_age_seconds", HostMetrics{}, SystemMetrics{}); ok {
		t.Fatal("state-age metric must be unknown when its table is unavailable")
	}
}

func TestStateAgeMetricValue(t *testing.T) {
	m, d := newStateAgeFixture(t)
	seedDeployment(t, d, "dep-pending", "pending", 10*time.Minute)
	seedDeployment(t, d, "dep-building", "building", 45*time.Minute)
	seedCiciRun(t, d, "run-queued", "queued", 15*time.Minute)

	tests := []struct {
		metric string
		want   float64
		ok     bool
	}{
		{"state_age_seconds", 2700, true},
		{"deployment_state_age_seconds", 2700, true},
		{"deployment_pending_age_seconds", 600, true},
		{"deployment_running_age_seconds", 0, true},
		{"cici_run_state_age_seconds", 900, true},
		{"cici_state_age_seconds", 900, true},
		{"cici_run_queued_age_seconds", 900, true},
		{"cici_queued_age_seconds", 900, true},
		{"deployment_unknown_age_seconds", 0, false},
		{"unrelated_metric", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			got, ok := m.metricValueWithSite(tt.metric, HostMetrics{}, SystemMetrics{})
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("value = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestStateAgeAlertRuleTriggersIncident(t *testing.T) {
	m, d := newStateAgeFixture(t)
	if err := m.initObservability(); err != nil {
		t.Fatalf("init observability: %v", err)
	}
	seedDeployment(t, d, "dep-stuck", "building", 30*time.Minute)

	if _, err := d.ExecContext(context.Background(),
		`INSERT INTO monitoring_alerts (id, name, metric, threshold, operator, enabled, duration_seconds)
		 VALUES ('stuck-deploy', 'stuck deploy', 'deployment_state_age_seconds', 600, 'gt', 1, 0)`); err != nil {
		t.Fatalf("seed alert: %v", err)
	}

	m.checkAlertRules(map[breachKey]time.Time{})

	var count, triggered int
	if err := d.QueryRowContext(context.Background(),
		`SELECT COUNT(*), COALESCE(SUM(triggered), 0) FROM monitoring_alert_incidents WHERE rule_id = 'stuck-deploy'`).Scan(&count, &triggered); err != nil {
		t.Fatalf("count incidents: %v", err)
	}
	if count != 1 || triggered != 1 {
		t.Fatalf("incidents = %d, triggered = %d, want 1/1", count, triggered)
	}
}

func TestStateAgeAlertRuleBelowThresholdDoesNotTrigger(t *testing.T) {
	m, d := newStateAgeFixture(t)
	if err := m.initObservability(); err != nil {
		t.Fatalf("init observability: %v", err)
	}
	seedDeployment(t, d, "dep-fresh", "building", 5*time.Minute)

	if _, err := d.ExecContext(context.Background(),
		`INSERT INTO monitoring_alerts (id, name, metric, threshold, operator, enabled, duration_seconds)
		 VALUES ('stuck-deploy', 'stuck deploy', 'deployment_state_age_seconds', 600, 'gt', 1, 0)`); err != nil {
		t.Fatalf("seed alert: %v", err)
	}

	m.checkAlertRules(map[breachKey]time.Time{})

	var count int
	if err := d.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM monitoring_alert_incidents WHERE rule_id = 'stuck-deploy'`).Scan(&count); err != nil {
		t.Fatalf("count incidents: %v", err)
	}
	if count != 0 {
		t.Fatalf("incidents = %d, want 0", count)
	}
}

func TestStateAgeExposedOnMetricsEndpoints(t *testing.T) {
	m, d := newStateAgeFixture(t)
	seedDeployment(t, d, "dep-stuck", "building", 30*time.Minute)
	seedCiciRun(t, d, "run-stuck", "running", 10*time.Minute)

	mux := m.Handler()

	t.Run("json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/monitoring/metrics", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body.String())
		}
		var body struct {
			StateAges StateAgeReport `json:"state_ages"`
		}
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.StateAges.Worst == nil || body.StateAges.Worst.OldestID != "dep-stuck" {
			t.Fatalf("worst = %+v, want dep-stuck", body.StateAges.Worst)
		}
	})

	t.Run("prometheus", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/monitoring/metrics/prometheus", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		for _, want := range []string{
			`bigbase_state_age_seconds{machine="deployment",state="building"} 1800.00`,
			`bigbase_state_age_seconds{machine="cici_run",state="running"} 600.00`,
			`# TYPE bigbase_state_age_seconds gauge`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("prometheus output missing %q\n%s", want, body)
			}
		}
	})
}
