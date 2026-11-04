package cici

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func newTestCICI(t *testing.T) (*CICI, *db.DB) {
	t.Helper()
	logger := kernel.NoopLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	k.Register(d)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	c := New(Options{DB: d, Logger: logger})
	if err := c.Start(&kernel.Context{}); err != nil {
		t.Fatalf("cici start: %v", err)
	}
	return c, d
}

func TestFinishRunRejectsNonTerminalStatus(t *testing.T) {
	c, d := newTestCICI(t)

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := d.ExecContext(context.Background(),
		"INSERT INTO cici_runs (id, workflow_id, event, status, started_at) VALUES (?, ?, ?, ?, ?)",
		"r1", "wf-1", "manual", "running", now); err != nil {
		t.Fatalf("seed run: %v", err)
	}

	err := c.finishRun(context.Background(), "r1", RunStatusRunning)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("finishRun(running) = %v, want ErrInvalidTransition", err)
	}

	var status string
	var finishedAt *string
	if err := d.QueryRowContext(context.Background(),
		"SELECT status, finished_at FROM cici_runs WHERE id = ?", "r1").
		Scan(&status, &finishedAt); err != nil {
		t.Fatalf("query run: %v", err)
	}
	if status != string(RunStatusRunning) || finishedAt != nil {
		t.Fatalf("rejected finish mutated the row: status=%q finished_at=%v", status, finishedAt)
	}

	if err := c.finishRun(context.Background(), "r1", RunStatusSuccess); err != nil {
		t.Fatalf("finishRun(success) = %v, want nil", err)
	}
	if err := d.QueryRowContext(context.Background(),
		"SELECT status, finished_at FROM cici_runs WHERE id = ?", "r1").
		Scan(&status, &finishedAt); err != nil {
		t.Fatalf("query run: %v", err)
	}
	if status != string(RunStatusSuccess) || finishedAt == nil {
		t.Fatalf("finishRun(success) did not persist: status=%q finished_at=%v", status, finishedAt)
	}
}

func TestFinishRunDoesNotOverwriteTerminalStatus(t *testing.T) {
	c, d := newTestCICI(t)

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := d.ExecContext(context.Background(),
		"INSERT INTO cici_runs (id, workflow_id, event, status, started_at, finished_at) VALUES (?, ?, ?, ?, ?, ?)",
		"r-terminal", "wf-1", "manual", "failure", now, now); err != nil {
		t.Fatalf("seed terminal run: %v", err)
	}

	// finishRun attempting to mark success on an already-terminal failed run
	if err := c.finishRun(context.Background(), "r-terminal", RunStatusSuccess); err != nil {
		t.Fatalf("finishRun on terminal run returned error: %v", err)
	}

	var status string
	if err := d.QueryRowContext(context.Background(),
		"SELECT status FROM cici_runs WHERE id = ?", "r-terminal").Scan(&status); err != nil {
		t.Fatalf("query run: %v", err)
	}
	if status != string(RunStatusFailure) {
		t.Fatalf("status = %q, want %q (must not overwrite terminal status)", status, RunStatusFailure)
	}
}

func TestFinishRunSurfacesDBErrors(t *testing.T) {
	c, d := newTestCICI(t)

	if err := d.Migrate("DROP TABLE cici_runs"); err != nil {
		t.Fatalf("drop table: %v", err)
	}
	if err := c.finishRun(context.Background(), "r1", RunStatusFailure); err == nil {
		t.Fatal("finishRun() = nil, want a surfaced database error")
	}
}
