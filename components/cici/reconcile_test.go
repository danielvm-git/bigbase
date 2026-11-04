package cici_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/cici"
	"github.com/danielvm/bigbase/kernel"
)

func TestReconcileOrphanedRunsOnStart(t *testing.T) {
	c, d := setupCICI(t)

	now := time.Now().UTC().Format(time.RFC3339)
	rows := []struct {
		id     string
		status string
	}{
		{"stale-running", "running"},
		{"stale-queued", "queued"},
		{"stale-pending", "pending"},
		{"finished-success", "success"},
		{"finished-failure", "failure"},
	}
	for _, row := range rows {
		if _, err := d.ExecContext(context.Background(),
			"INSERT INTO cici_runs (id, workflow_id, event, status, started_at) VALUES (?, ?, ?, ?, ?)",
			row.id, "wf-1", "manual", row.status, now); err != nil {
			t.Fatalf("seed run %s: %v", row.id, err)
		}
	}

	// Start is idempotent and performs boot reconciliation.
	if err := c.Start(&kernel.Context{}); err != nil {
		t.Fatalf("start: %v", err)
	}

	wantStatus := map[string]struct {
		status string
		stale  bool
	}{
		"stale-running":    {"failure", true},
		"stale-queued":     {"failure", true},
		"stale-pending":    {"failure", true},
		"finished-success": {"success", false},
		"finished-failure": {"failure", false},
	}
	for id, want := range wantStatus {
		var status string
		var finishedAt *string
		if err := d.QueryRowContext(context.Background(),
			"SELECT status, finished_at FROM cici_runs WHERE id = ?", id).
			Scan(&status, &finishedAt); err != nil {
			t.Fatalf("query run %s: %v", id, err)
		}
		if status != want.status {
			t.Errorf("run %s status = %q, want %q", id, status, want.status)
		}
		if want.stale && finishedAt == nil {
			t.Errorf("run %s finished_at = nil, want a timestamp", id)
		}
	}
}

func TestListRunsNormalizesLegacyPending(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := d.ExecContext(context.Background(),
		"INSERT INTO cici_runs (id, workflow_id, event, status, started_at) VALUES (?, ?, ?, ?, ?)",
		"legacy-pending", "wf-1", "manual", "pending", now); err != nil {
		t.Fatalf("seed legacy run: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/cici/runs", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string][]map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	for _, r := range resp["data"] {
		if r["id"] == "legacy-pending" {
			if r["status"] != string(cici.RunStatusQueued) {
				t.Fatalf("legacy pending status = %v, want %q", r["status"], cici.RunStatusQueued)
			}
			return
		}
	}
	t.Fatal("legacy-pending run not returned by GET /api/cici/runs")
}

func TestTriggerRunPersistsDeadline(t *testing.T) {
	c, d := setupCICI(t)
	h := c.Handler()

	seedRepo(t, d, "r", "my-app", 100)
	wfID := saveWorkflow(t, h, "r", "CI", `name: CI
on: [push]
jobs:
  build:
    steps:
      - run: echo "ok"
`, 100)

	runBody, _ := json.Marshal(map[string]string{"event": "manual"})
	runReq := httptest.NewRequest("POST", "/api/cici/r/workflows/"+wfID+"/run", bytes.NewReader(runBody))
	runReq.Header.Set("Content-Type", "application/json")
	runReq = withOrgID(runReq, 100)
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	if runW.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", runW.Code, runW.Body.String())
	}
	var run map[string]any
	if err := json.NewDecoder(runW.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	runID, _ := run["id"].(string)
	if run["status"] != string(cici.RunStatusRunning) {
		t.Fatalf("response status = %v, want %q", run["status"], cici.RunStatusRunning)
	}

	var deadlineAt *string
	if err := d.QueryRowContext(context.Background(),
		"SELECT deadline_at FROM cici_runs WHERE id = ?", runID).Scan(&deadlineAt); err != nil {
		t.Fatalf("query deadline: %v", err)
	}
	if deadlineAt == nil || *deadlineAt == "" {
		t.Fatal("deadline_at was not persisted")
	}

	pollForStatus(t, h, runID, "success")
}
