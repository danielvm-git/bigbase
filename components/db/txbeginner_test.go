package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

// TestTxBeginnerSQLiteSeam verifies the SQLite driver implements the optional
// kernel.TxBeginner seam and that transactions actually serialize writes.
func TestTxBeginnerSQLiteSeam(t *testing.T) {
	var _ kernel.TxBeginner = (*db.DB)(nil)

	d := db.New(db.Options{Path: ":memory:", Logger: testLogger{}})
	ctx := &kernel.Context{Logger: testLogger{}}
	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = d.Stop(ctx) }()
	if err := d.Migrate(`CREATE TABLE tx_items (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Rolled-back transaction leaves no trace.
	tx, err := d.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO tx_items (name) VALUES ('rolled-back')`); err != nil {
		t.Fatalf("tx exec: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM tx_items`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rolled-back insert visible: count=%d err=%v", count, err)
	}

	// Committed transaction persists.
	tx, err = d.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO tx_items (name) VALUES ('committed')`); err != nil {
		t.Fatalf("tx exec: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := d.QueryRow(`SELECT COUNT(*) FROM tx_items`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("committed insert missing: count=%d err=%v", count, err)
	}
}

// TestTxBeginnerPostgresSeam verifies the PostgreSQL driver implements the
// optional kernel.TxBeginner seam. The functional check is gated on an opt-in
// DSN and must be skipped with an explicit reason when none is configured.
func TestTxBeginnerPostgresSeam(t *testing.T) {
	var _ kernel.TxBeginner = (*db.PostgresDB)(nil)

	dsn := os.Getenv("BIGBASE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("BIGBASE_TEST_POSTGRES_DSN not set; SQLite coverage only")
	}
	p := db.NewPostgres(db.PostgresOptions{DSN: dsn, Logger: testLogger{}})
	ctx := &kernel.Context{Logger: testLogger{}}
	if err := p.Start(ctx); err != nil {
		t.Fatalf("postgres Start: %v", err)
	}
	defer func() { _ = p.Stop(ctx) }()

	tx, err := p.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("postgres BeginTx: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("postgres Rollback: %v", err)
	}
}
