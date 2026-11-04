package deploy

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

type reconcileTestLogger struct{}

func (reconcileTestLogger) Info(string, ...any)  {}
func (reconcileTestLogger) Warn(string, ...any)  {}
func (reconcileTestLogger) Error(string, ...any) {}
func (reconcileTestLogger) Debug(string, ...any) {}

// newReconcileTestDeploy creates a Deploy over an in-memory DB and runs Start
// once so all migrations (including status_history/error_message) are applied.
func newReconcileTestDeploy(t *testing.T, staleAfter time.Duration) (*Deploy, *db.DB) {
	t.Helper()
	logger := reconcileTestLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	if err := database.Start(&kernel.Context{}); err != nil {
		t.Fatalf("db start: %v", err)
	}
	t.Cleanup(func() { _ = database.Stop(&kernel.Context{}) })

	d := New(Options{
		DB:                  database,
		Logger:              logger,
		BuildsDir:           t.TempDir(),
		GitDir:              t.TempDir(),
		ReconcileStaleAfter: staleAfter,
	})
	if err := d.Start(&kernel.Context{}); err != nil {
		t.Fatalf("deploy start: %v", err)
	}
	return d, database
}

func seedDeployment(t *testing.T, database *db.DB, id, status string, port int, createdAt string) {
	t.Helper()
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, branch, status, url, port, app_type, created_at)
		 VALUES (?, 'repo-1', 'site-1', 'main', ?, '', ?, '', ?)`,
		id, status, port, createdAt)
	if err != nil {
		t.Fatalf("seed deployment %s: %v", id, err)
	}
}

func seedDeploymentNullPort(t *testing.T, database *db.DB, id, status, createdAt string) {
	t.Helper()
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO deployments (id, repo_id, site_id, branch, status, url, port, app_type, created_at)
		 VALUES (?, 'repo-1', 'site-1', 'main', ?, '', NULL, '', ?)`,
		id, status, createdAt)
	if err != nil {
		t.Fatalf("seed deployment %s: %v", id, err)
	}
}

type deploymentSnapshot struct {
	status  string
	errMsg  string
	history []StatusTransition
}

func readDeployment(t *testing.T, database *db.DB, id string) deploymentSnapshot {
	t.Helper()
	var snap deploymentSnapshot
	var historyJSON string
	if err := database.QueryRowContext(context.Background(),
		`SELECT status, COALESCE(error_message, ''), COALESCE(status_history, '[]')
		 FROM deployments WHERE id = ?`, id).Scan(&snap.status, &snap.errMsg, &historyJSON); err != nil {
		t.Fatalf("read deployment %s: %v", id, err)
	}
	if err := json.Unmarshal([]byte(historyJSON), &snap.history); err != nil {
		t.Fatalf("parse status_history for %s: %v (raw: %s)", id, err, historyJSON)
	}
	return snap
}

func assertFailedByReconciliation(t *testing.T, snap deploymentSnapshot, id, wantFrom string) {
	t.Helper()
	if snap.status != string(StateFailed) {
		t.Errorf("%s status = %q, want failed", id, snap.status)
	}
	if !strings.Contains(snap.errMsg, "reconciliation") {
		t.Errorf("%s error_message = %q, want a reconciliation reason", id, snap.errMsg)
	}
	if len(snap.history) != 1 {
		t.Fatalf("%s history = %+v, want exactly 1 entry", id, snap.history)
	}
	if snap.history[0].From != wantFrom || snap.history[0].To != string(StateFailed) {
		t.Errorf("%s history[0] = %+v, want %s→failed", id, snap.history[0], wantFrom)
	}
}

func TestReconcileOrphanedDeployments(t *testing.T) {
	d, database := newReconcileTestDeploy(t, 30*time.Minute)

	old := time.Now().UTC().Add(-2 * time.Hour)
	oldRFC3339 := old.Format(time.RFC3339)
	oldSQLite := old.Format("2006-01-02 15:04:05")
	fresh := time.Now().UTC().Format(time.RFC3339)

	seedDeployment(t, database, "stale-pending", "pending", 0, oldRFC3339)
	seedDeployment(t, database, "stale-building", "building", 0, oldRFC3339)
	seedDeployment(t, database, "stale-draining", "draining", 0, oldRFC3339)
	// Mixed timestamp format — the legacy migration default uses SQLite's format.
	seedDeployment(t, database, "stale-deploying", "deploying", 0, oldSQLite)
	seedDeployment(t, database, "running-noport", "running", 0, oldRFC3339)
	seedDeploymentNullPort(t, database, "running-nullport", "running", oldRFC3339)
	seedDeployment(t, database, "fresh-building", "building", 0, fresh)
	seedDeployment(t, database, "terminal-failed", "failed", 0, oldRFC3339)

	got := d.reconcileOrphanedDeployments(context.Background())
	if got != 6 {
		t.Fatalf("reconciled = %d, want 6", got)
	}

	assertFailedByReconciliation(t, readDeployment(t, database, "stale-pending"), "stale-pending", "pending")
	assertFailedByReconciliation(t, readDeployment(t, database, "stale-building"), "stale-building", "building")
	assertFailedByReconciliation(t, readDeployment(t, database, "stale-draining"), "stale-draining", "draining")
	assertFailedByReconciliation(t, readDeployment(t, database, "stale-deploying"), "stale-deploying", "deploying")
	assertFailedByReconciliation(t, readDeployment(t, database, "running-noport"), "running-noport", "running")
	assertFailedByReconciliation(t, readDeployment(t, database, "running-nullport"), "running-nullport", "running")

	freshSnap := readDeployment(t, database, "fresh-building")
	if freshSnap.status != string(StateBuilding) {
		t.Errorf("fresh-building status = %q, want building (fresh rows must be untouched)", freshSnap.status)
	}
	if freshSnap.errMsg != "" {
		t.Errorf("fresh-building error_message = %q, want empty", freshSnap.errMsg)
	}
	if len(freshSnap.history) != 0 {
		t.Errorf("fresh-building history = %+v, want none", freshSnap.history)
	}

	terminalSnap := readDeployment(t, database, "terminal-failed")
	if terminalSnap.status != string(StateFailed) {
		t.Errorf("terminal-failed status = %q, want failed", terminalSnap.status)
	}
	if terminalSnap.errMsg != "" {
		t.Errorf("terminal-failed error_message = %q, want untouched empty", terminalSnap.errMsg)
	}
	if len(terminalSnap.history) != 0 {
		t.Errorf("terminal-failed history = %+v, want untouched", terminalSnap.history)
	}
}

func TestReconcileIsIdempotent(t *testing.T) {
	d, database := newReconcileTestDeploy(t, 30*time.Minute)

	old := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	seedDeployment(t, database, "stale-building", "building", 0, old)

	if got := d.reconcileOrphanedDeployments(context.Background()); got != 1 {
		t.Fatalf("first reconcile = %d, want 1", got)
	}
	first := readDeployment(t, database, "stale-building")
	assertFailedByReconciliation(t, first, "stale-building", "building")

	if got := d.reconcileOrphanedDeployments(context.Background()); got != 0 {
		t.Fatalf("second reconcile = %d, want 0 (idempotent)", got)
	}
	second := readDeployment(t, database, "stale-building")
	if second.status != first.status || second.errMsg != first.errMsg {
		t.Errorf("second reconcile changed the row: first=%+v second=%+v", first, second)
	}
	if len(second.history) != 1 {
		t.Fatalf("history after second reconcile = %+v, want no duplicate entry", second.history)
	}
}

// TestReconcileOnStart proves the reconciliation is wired into Deploy.Start,
// not just callable directly: stale rows seeded behind a running Deploy are
// healed when a fresh Deploy starts over the same database.
func TestReconcileOnStart(t *testing.T) {
	_, database := newReconcileTestDeploy(t, 30*time.Minute)

	old := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	seedDeployment(t, database, "stale-building", "building", 0, old)

	restarted := New(Options{
		DB:                  database,
		Logger:              reconcileTestLogger{},
		BuildsDir:           t.TempDir(),
		GitDir:              t.TempDir(),
		ReconcileStaleAfter: 30 * time.Minute,
	})
	if err := restarted.Start(&kernel.Context{}); err != nil {
		t.Fatalf("restarted deploy start: %v", err)
	}
	t.Cleanup(func() { _ = restarted.Stop(&kernel.Context{}) })

	assertFailedByReconciliation(t, readDeployment(t, database, "stale-building"), "stale-building", "building")
}
