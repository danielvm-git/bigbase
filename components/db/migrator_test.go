package db_test

import (
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

// openTestDB is a helper that returns a started, in-memory SQLite DB component.
func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	d := db.New(db.Options{
		Driver: db.DriverSQLite,
		DSN:    ":memory:",
		Logger: testLogger{},
	})
	ctx := &kernel.Context{Logger: testLogger{}}
	if err := d.Init(ctx, nil); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		if err := d.Stop(ctx); err != nil {
			t.Errorf("Stop: %v", err)
		}
	})
	return d
}

func TestMigrationUp(t *testing.T) {
	d := openTestDB(t)
	migs := []db.Migration{
		{
			Version: "001_create_items",
			Up:      `CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
			Down:    `DROP TABLE items`,
		},
		{
			Version: "002_add_price",
			Up:      `ALTER TABLE items ADD COLUMN price INTEGER NOT NULL DEFAULT 0`,
			Down:    `CREATE TABLE items_new (id INTEGER PRIMARY KEY, name TEXT NOT NULL); INSERT INTO items_new SELECT id, name FROM items; DROP TABLE items; ALTER TABLE items_new RENAME TO items`,
		},
	}
	migrator := db.NewMigrator(d.DB(), migs)

	if err := migrator.Up(); err != nil {
		t.Fatalf("Up: %v", err)
	}

	// Verify schema: insert a row that requires both columns.
	if _, err := d.DB().Exec("INSERT INTO items (name, price) VALUES (?, ?)", "widget", 42); err != nil {
		t.Fatalf("insert after migration: %v", err)
	}

	var name string
	var price int
	row := d.DB().QueryRow("SELECT name, price FROM items WHERE id = 1")
	if err := row.Scan(&name, &price); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if name != "widget" || price != 42 {
		t.Fatalf("unexpected row: name=%q price=%d", name, price)
	}
}

func TestMigrationUpIdempotent(t *testing.T) {
	d := openTestDB(t)
	migs := []db.Migration{
		{
			Version: "001_create_items",
			Up:      `CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
			Down:    `DROP TABLE items`,
		},
	}
	migrator := db.NewMigrator(d.DB(), migs)

	if err := migrator.Up(); err != nil {
		t.Fatalf("first Up: %v", err)
	}
	// Calling Up again must not re-run the already-applied migration.
	if err := migrator.Up(); err != nil {
		t.Fatalf("second Up (idempotent): %v", err)
	}
}

func TestMigrationDown(t *testing.T) {
	d := openTestDB(t)
	migs := []db.Migration{
		{
			Version: "001_create_items",
			Up:      `CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
			Down:    `DROP TABLE items`,
		},
		{
			Version: "002_create_tags",
			Up:      `CREATE TABLE tags (id INTEGER PRIMARY KEY, label TEXT NOT NULL)`,
			Down:    `DROP TABLE tags`,
		},
	}
	migrator := db.NewMigrator(d.DB(), migs)

	if err := migrator.Up(); err != nil {
		t.Fatalf("Up: %v", err)
	}

	// Roll back just the last migration.
	if err := migrator.Down(1); err != nil {
		t.Fatalf("Down(1): %v", err)
	}

	// tags table must be gone.
	_, err := d.DB().Exec("INSERT INTO tags (label) VALUES (?)", "x")
	if err == nil {
		t.Fatal("expected error querying dropped tags table, got none")
	}

	// items table must still exist.
	if _, err := d.DB().Exec("INSERT INTO items (name) VALUES (?)", "y"); err != nil {
		t.Fatalf("items table should still exist: %v", err)
	}
}

func TestMigrationStatus(t *testing.T) {
	d := openTestDB(t)
	migs := []db.Migration{
		{
			Version: "001_create_items",
			Up:      `CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
			Down:    `DROP TABLE items`,
		},
		{
			Version: "002_create_tags",
			Up:      `CREATE TABLE tags (id INTEGER PRIMARY KEY, label TEXT NOT NULL)`,
			Down:    `DROP TABLE tags`,
		},
	}
	migrator := db.NewMigrator(d.DB(), migs)

	// Apply only first migration.
	oneMig := db.NewMigrator(d.DB(), migs[:1])
	if err := oneMig.Up(); err != nil {
		t.Fatalf("Up: %v", err)
	}

	records, err := migrator.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 status records, got %d", len(records))
	}

	// First should be applied (non-zero AppliedAt).
	if records[0].AppliedAt.IsZero() {
		t.Errorf("migration 001 should be applied, got zero AppliedAt")
	}
	// Second should be pending (zero AppliedAt).
	if !records[1].AppliedAt.IsZero() {
		t.Errorf("migration 002 should be pending, got non-zero AppliedAt: %v", records[1].AppliedAt)
	}
}

func TestMigrationDownAll(t *testing.T) {
	d := openTestDB(t)
	migs := []db.Migration{
		{
			Version: "001_create_items",
			Up:      `CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
			Down:    `DROP TABLE items`,
		},
		{
			Version: "002_create_tags",
			Up:      `CREATE TABLE tags (id INTEGER PRIMARY KEY, label TEXT NOT NULL)`,
			Down:    `DROP TABLE tags`,
		},
	}
	migrator := db.NewMigrator(d.DB(), migs)
	if err := migrator.Up(); err != nil {
		t.Fatalf("Up: %v", err)
	}

	// n=0 rolls back all.
	if err := migrator.Down(0); err != nil {
		t.Fatalf("Down(0): %v", err)
	}

	// Both tables must be gone.
	for _, tbl := range []string{"items", "tags"} {
		tbl := tbl
		_, err := d.DB().Exec("INSERT INTO " + tbl + " (name) VALUES (?)")
		if err == nil {
			t.Errorf("expected %q table to be dropped, but insert succeeded", tbl)
		}
	}
}
