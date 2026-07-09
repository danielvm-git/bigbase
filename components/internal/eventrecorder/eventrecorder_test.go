package eventrecorder_test

import (
	"context"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/internal/eventrecorder"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(string, ...any)  {}
func (testLogger) Warn(string, ...any)  {}
func (testLogger) Error(string, ...any) {}
func (testLogger) Debug(string, ...any) {}

func TestEventRecorder(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	k.Register(database)
	if err := k.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	rec, err := eventrecorder.New(database)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	if err := rec.Record(ctx, "mutation", "site-1", map[string]any{"type": "create"}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := rec.Record(ctx, "scaffold_db", "site-1", map[string]any{"name": "posts"}); err != nil {
		t.Fatalf("Record scaffold: %v", err)
	}
	if err := rec.Record(ctx, "request", "site-2", map[string]any{"path": "/"}); err != nil {
		t.Fatalf("Record other site: %v", err)
	}

	events, err := rec.Query(ctx, eventrecorder.Filter{
		SiteID:      "site-1",
		WindowStart: now.Add(-time.Minute),
		WindowEnd:   now.Add(time.Minute),
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events for site-1, got %d", len(events))
	}
}

func TestEventRecorderFIFOCap(t *testing.T) {
	logger := testLogger{}
	database := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	k.Register(database)
	if err := k.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	rec, err := eventrecorder.New(database)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	for i := 0; i < 5010; i++ {
		if err := rec.Record(ctx, "mutation", "", map[string]any{"i": i}); err != nil {
			t.Fatalf("Record %d: %v", i, err)
		}
	}
	var count int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM monitoring_events`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count > 5000 {
		t.Fatalf("expected FIFO cap 5000, got %d", count)
	}
}
