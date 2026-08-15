package monitoring_test

// e86s04: monitoring auto-ingests deploy lifecycle events into org-scoped
// monitoring_logs rows.

import (
	"context"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/monitoring"
	"github.com/danielvm/bigbase/kernel"
)

func TestDeployIngest(t *testing.T) {
	logger := testLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	k := kernel.New(logger)
	m := monitoring.New(monitoring.Options{DB: d, Logger: logger})
	k.Register(d)
	k.Register(m)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	bus := k.EventBus()

	// A successful transition → an info row for that org.
	_ = bus.Emit(kernel.Event{Name: "deploy.state_changed", Data: map[string]any{
		"deployment_id": "dep-1", "org_id": int64(42),
		"from_state": "building", "to_state": "running",
	}}, &kernel.Context{})

	// A failure → an error row for that org.
	_ = bus.Emit(kernel.Event{Name: "deploy.failed", Data: map[string]any{
		"deployment_id": "dep-2", "org_id": int64(42), "error_message": "boom",
	}}, &kernel.Context{})

	// Handlers run synchronously, so rows exist now.
	var info, errc int
	if err := d.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM monitoring_logs WHERE org_id = 42 AND level = 'info'").Scan(&info); err != nil {
		t.Fatalf("count info: %v", err)
	}
	if err := d.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM monitoring_logs WHERE org_id = 42 AND level = 'error'").Scan(&errc); err != nil {
		t.Fatalf("count error: %v", err)
	}
	if info < 1 {
		t.Error("deploy.state_changed should ingest an info log row scoped to org 42")
	}
	if errc < 1 {
		t.Error("deploy.failed should ingest an error log row scoped to org 42")
	}

	// Isolation: a different org must not see these rows via the search handler.
	h := m.Handler()
	resp := searchLogs(t, h, 99, "")
	for _, l := range resp.Data {
		if l.OrgID == 42 {
			t.Error("ingested deploy logs leaked to a different org")
		}
	}
}
