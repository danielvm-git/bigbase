package backup_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/backup"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupDB(t *testing.T) *db.DB {
	t.Helper()
	d := db.New(db.Options{Path: ":memory:", Logger: testLogger{}})
	ctx := &kernel.Context{Logger: testLogger{}}
	if err := d.Init(ctx, nil); err != nil {
		t.Fatalf("db Init: %v", err)
	}
	if err := d.Start(ctx); err != nil {
		t.Fatalf("db Start: %v", err)
	}
	t.Cleanup(func() { _ = d.Stop(ctx) })
	return d
}

func TestDumpEmpty(t *testing.T) {
	d := setupDB(t)
	var buf bytes.Buffer
	if err := backup.Dump(context.Background(), d.DB(), &buf); err != nil {
		t.Fatalf("Dump: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "BEGIN TRANSACTION") {
		t.Errorf("expected BEGIN TRANSACTION in dump, got:\n%s", got)
	}
	if !strings.Contains(got, "COMMIT") {
		t.Errorf("expected COMMIT in dump, got:\n%s", got)
	}
}

func TestDumpWithData(t *testing.T) {
	d := setupDB(t)

	if err := d.Migrate(`CREATE TABLE IF NOT EXISTS items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := d.Exec("INSERT INTO items (name) VALUES (?), (?)", "alpha", "beta"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var buf bytes.Buffer
	if err := backup.Dump(context.Background(), d.DB(), &buf); err != nil {
		t.Fatalf("Dump: %v", err)
	}
	got := buf.String()

	cases := []struct {
		label    string
		contains string
	}{
		{"DDL", "CREATE TABLE"},
		{"table name", "items"},
		{"row alpha", "'alpha'"},
		{"row beta", "'beta'"},
		{"transaction", "COMMIT"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			if !strings.Contains(got, tc.contains) {
				t.Errorf("dump missing %q:\n%s", tc.contains, got)
			}
		})
	}
}

func TestDumpQuotesSpecialChars(t *testing.T) {
	d := setupDB(t)
	if err := d.Migrate(`CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY, txt TEXT)`); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := d.Exec("INSERT INTO notes (txt) VALUES (?)", "it's a test"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var buf bytes.Buffer
	if err := backup.Dump(context.Background(), d.DB(), &buf); err != nil {
		t.Fatalf("Dump: %v", err)
	}
	if !strings.Contains(buf.String(), "it''s a test") {
		t.Errorf("expected escaped apostrophe in dump:\n%s", buf.String())
	}
}
