package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

// TestPostgres tests the PostgresDB driver.
// If POSTGRES_DSN is not set, the test uses a mock that verifies
// the struct satisfies the DBer interface (compile-time check only).
func TestPostgres(t *testing.T) {
	t.Run("implements_kernel_DBer", func(t *testing.T) {
		// Compile-time assertion: PostgresDB must satisfy kernel.DBer.
		var _ kernel.DBer = (*db.PostgresDB)(nil)
	})

	t.Run("new_returns_non_nil", func(t *testing.T) {
		p := db.NewPostgres(db.PostgresOptions{
			DSN:    "postgres://localhost/test",
			Logger: testLogger{},
		})
		if p == nil {
			t.Fatal("expected non-nil PostgresDB")
		}
	})

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Log("POSTGRES_DSN not set — skipping live DB tests")
		return
	}

	t.Run("start_stop", func(t *testing.T) {
		p := db.NewPostgres(db.PostgresOptions{
			DSN:    dsn,
			Logger: testLogger{},
		})
		ctx := &kernel.Context{Logger: testLogger{}}

		if err := p.Init(ctx, nil); err != nil {
			t.Fatalf("Init: %v", err)
		}
		if err := p.Start(ctx); err != nil {
			t.Fatalf("Start: %v", err)
		}
		if err := p.Stop(ctx); err != nil {
			t.Fatalf("Stop: %v", err)
		}
	})

	t.Run("exec_query_queryrow", func(t *testing.T) {
		p := db.NewPostgres(db.PostgresOptions{
			DSN:    dsn,
			Logger: testLogger{},
		})
		ctx := &kernel.Context{Logger: testLogger{}}
		if err := p.Init(ctx, nil); err != nil {
			t.Fatalf("Init: %v", err)
		}
		if err := p.Start(ctx); err != nil {
			t.Fatalf("Start: %v", err)
		}
		defer func() { _ = p.Stop(ctx) }()

		bg := context.Background()

		// Exec — DDL
		if err := p.Migrate(`
			CREATE TEMP TABLE IF NOT EXISTS pg_test_items (
				id   SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			)
		`); err != nil {
			t.Fatalf("Migrate: %v", err)
		}

		// ExecContext — INSERT
		if _, err := p.ExecContext(bg, "INSERT INTO pg_test_items(name) VALUES($1)", "hello"); err != nil {
			t.Fatalf("ExecContext: %v", err)
		}

		// QueryRowContext — COUNT
		var count int
		if err := p.QueryRowContext(bg, "SELECT COUNT(*) FROM pg_test_items").Scan(&count); err != nil {
			t.Fatalf("QueryRowContext: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 row, got %d", count)
		}

		// QueryContext — SELECT
		rows, err := p.QueryContext(bg, "SELECT name FROM pg_test_items")
		if err != nil {
			t.Fatalf("QueryContext: %v", err)
		}
		defer func() { _ = rows.Close() }()

		var names []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatalf("scan: %v", err)
			}
			names = append(names, name)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows.Err: %v", err)
		}
		if len(names) != 1 || names[0] != "hello" {
			t.Fatalf("unexpected names: %v", names)
		}
	})
}
