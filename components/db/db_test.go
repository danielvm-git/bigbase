package db_test

import (
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func TestDBImplementsComponent(t *testing.T) {
	var _ kernel.Component = &db.DB{}
}

func TestDBName(t *testing.T) {
	d := &db.DB{}
	if got := d.Name(); got != "db" {
		t.Fatalf("expected Name()='db', got '%s'", got)
	}
}

func TestDBInitMemory(t *testing.T) {
	d := db.New(db.Options{Path: ":memory:", Logger: testLogger{}})
	ctx := &kernel.Context{Logger: testLogger{}}
	if err := d.Init(ctx, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestDBStartStop(t *testing.T) {
	d := db.New(db.Options{Path: ":memory:", Logger: testLogger{}})
	ctx := &kernel.Context{Logger: testLogger{}}

	if err := d.Init(ctx, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if d.DB() == nil {
		t.Fatal("expected non-nil DB handle after Start")
	}

	if err := d.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDBStartWithNilLogger(t *testing.T) {
	d := db.New(db.Options{Path: ":memory:"})
	ctx := &kernel.Context{Logger: testLogger{}}
	if err := d.Init(ctx, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := d.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDBMigrate(t *testing.T) {
	d := db.New(db.Options{Path: ":memory:", Logger: testLogger{}})
	ctx := &kernel.Context{Logger: testLogger{}}
	if err := d.Init(ctx, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() {
		if err := d.Stop(ctx); err != nil {
			t.Fatalf("Stop failed: %v", err)
		}
	}()

	if err := d.Migrate(`CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	var count int
	row := d.DB().QueryRow("SELECT COUNT(*) FROM users")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows, got %d", count)
	}
}

func TestDBExecQuery(t *testing.T) {
	d := db.New(db.Options{Path: ":memory:", Logger: testLogger{}})
	ctx := &kernel.Context{Logger: testLogger{}}
	if err := d.Init(ctx, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() {
		if err := d.Stop(ctx); err != nil {
			t.Fatalf("Stop failed: %v", err)
		}
	}()

	if err := d.Migrate(`CREATE TABLE IF NOT EXISTS items (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if _, err := d.Exec("INSERT INTO items (name) VALUES (?)", "test-item"); err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	rows, err := d.Query("SELECT id, name FROM items ORDER BY id")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Fatalf("rows.Close failed: %v", err)
		}
	}()

	if !rows.Next() {
		t.Fatal("expected at least one row")
	}

	var id int
	var name string
	if err := rows.Scan(&id, &name); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if name != "test-item" {
		t.Fatalf("expected 'test-item', got '%s'", name)
	}
}
