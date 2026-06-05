package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielvm/bigbase/components/backup"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func setupDBForMigrate(t *testing.T) *db.DB {
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

func makeMigrationsDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write migration file %s: %v", name, err)
		}
	}
	return dir
}

func TestMigrationTool(t *testing.T) {
	t.Run("up applies pending migrations in order", func(t *testing.T) {
		d := setupDBForMigrate(t)
		dir := makeMigrationsDir(t, map[string]string{
			"001_create_foo.up.sql":   "CREATE TABLE foo (id INTEGER PRIMARY KEY, name TEXT);",
			"001_create_foo.down.sql": "DROP TABLE IF EXISTS foo;",
			"002_create_bar.up.sql":   "CREATE TABLE bar (id INTEGER PRIMARY KEY);",
			"002_create_bar.down.sql": "DROP TABLE IF EXISTS bar;",
		})

		runner := backup.NewMigrationRunner(d.DB(), dir)
		if err := runner.Up(context.Background()); err != nil {
			t.Fatalf("Up: %v", err)
		}

		// Both tables must exist.
		for _, tbl := range []string{"foo", "bar"} {
			var name string
			err := d.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&name)
			if err != nil {
				t.Errorf("table %q not found after Up: %v", tbl, err)
			}
		}

		ver, dirty, err := runner.Status(context.Background())
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		if ver != 2 {
			t.Errorf("expected version 2, got %d", ver)
		}
		if dirty {
			t.Error("expected dirty=false")
		}
	})

	t.Run("up is idempotent", func(t *testing.T) {
		d := setupDBForMigrate(t)
		dir := makeMigrationsDir(t, map[string]string{
			"001_init.up.sql":   "CREATE TABLE idempotent_test (id INTEGER PRIMARY KEY);",
			"001_init.down.sql": "DROP TABLE IF EXISTS idempotent_test;",
		})

		runner := backup.NewMigrationRunner(d.DB(), dir)
		if err := runner.Up(context.Background()); err != nil {
			t.Fatalf("first Up: %v", err)
		}
		if err := runner.Up(context.Background()); err != nil {
			t.Fatalf("second Up (idempotent): %v", err)
		}
	})

	t.Run("down reverts latest migration", func(t *testing.T) {
		d := setupDBForMigrate(t)
		dir := makeMigrationsDir(t, map[string]string{
			"001_create_rev.up.sql":   "CREATE TABLE rev (id INTEGER PRIMARY KEY);",
			"001_create_rev.down.sql": "DROP TABLE IF EXISTS rev;",
		})

		runner := backup.NewMigrationRunner(d.DB(), dir)
		if err := runner.Up(context.Background()); err != nil {
			t.Fatalf("Up: %v", err)
		}
		if err := runner.Down(context.Background()); err != nil {
			t.Fatalf("Down: %v", err)
		}

		// Table must be gone.
		var name string
		err := d.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='rev'").Scan(&name)
		if err == nil {
			t.Error("expected table rev to be dropped after Down")
		}

		ver, _, err := runner.Status(context.Background())
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		if ver != 0 {
			t.Errorf("expected version 0 after Down, got %d", ver)
		}
	})

	t.Run("status with no migrations", func(t *testing.T) {
		d := setupDBForMigrate(t)
		dir := makeMigrationsDir(t, map[string]string{})

		runner := backup.NewMigrationRunner(d.DB(), dir)
		ver, dirty, err := runner.Status(context.Background())
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		if ver != 0 {
			t.Errorf("expected version 0, got %d", ver)
		}
		if dirty {
			t.Error("expected dirty=false")
		}
	})

	t.Run("missing migrations dir is handled", func(t *testing.T) {
		d := setupDBForMigrate(t)
		runner := backup.NewMigrationRunner(d.DB(), "/nonexistent/migrations")
		if err := runner.Up(context.Background()); err != nil {
			t.Fatalf("Up with missing dir should not error: %v", err)
		}
	})
}
