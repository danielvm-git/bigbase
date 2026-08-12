package backup_test

import (
	"bytes"
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/danielvm/bigbase/components/backup"
	_ "modernc.org/sqlite"
)

// TestDumpRestoreRoundTripWithForeignKeys guards the restore path that the
// secret-manager backup scenario depends on: a dump taken from a database with
// foreign keys enabled must restore cleanly into a fresh database even though
// table creation order in the dump is alphabetical (children before parents),
// and the restored database must still enforce the constraints.
func TestDumpRestoreRoundTripWithForeignKeys(t *testing.T) {
	dir := t.TempDir()
	src := openFileDB(t, filepath.Join(dir, "src.db"))
	defer func() { _ = src.Close() }()

	if _, err := src.Exec(`CREATE TABLE parents (id TEXT PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("create parents: %v", err)
	}
	if _, err := src.Exec(`CREATE TABLE children (
		id TEXT PRIMARY KEY,
		parent_id TEXT NOT NULL,
		FOREIGN KEY (parent_id) REFERENCES parents(id) ON DELETE CASCADE
	)`); err != nil {
		t.Fatalf("create children: %v", err)
	}
	if _, err := src.Exec(`INSERT INTO parents (id, name) VALUES ('p1', 'parent-one')`); err != nil {
		t.Fatalf("insert parent: %v", err)
	}
	if _, err := src.Exec(`INSERT INTO children (id, parent_id) VALUES ('c1', 'p1')`); err != nil {
		t.Fatalf("insert child: %v", err)
	}

	var dump bytes.Buffer
	ctx := context.Background()
	if err := backup.Dump(ctx, src, &dump); err != nil {
		t.Fatalf("Dump: %v", err)
	}

	dst := openFileDB(t, filepath.Join(dir, "dst.db"))
	if err := backup.Restore(ctx, dst, dump.String()); err != nil {
		_ = dst.Close()
		t.Fatalf("Restore: %v", err)
	}

	var parentName, childID string
	if err := dst.QueryRow(`SELECT name FROM parents WHERE id = 'p1'`).Scan(&parentName); err != nil || parentName != "parent-one" {
		t.Fatalf("parent row not restored: name=%q err=%v", parentName, err)
	}
	if err := dst.QueryRow(`SELECT id FROM children WHERE parent_id = 'p1'`).Scan(&childID); err != nil || childID != "c1" {
		t.Fatalf("child row not restored: id=%q err=%v", childID, err)
	}

	// Constraints survive the round trip. The restore connection keeps the
	// dump's PRAGMA foreign_keys=OFF, so enforcement is verified on a fresh
	// connection — the realistic application flow after a restore.
	_ = dst.Close()
	dst2 := openFileDB(t, filepath.Join(dir, "dst.db"))
	defer func() { _ = dst2.Close() }()
	if _, err := dst2.Exec(`INSERT INTO children (id, parent_id) VALUES ('c2', 'missing-parent')`); err == nil {
		t.Fatal("foreign key not enforced after restore")
	}
}

// openFileDB opens a file-backed SQLite database with foreign keys enabled
// and a pinned connection so the dump's PRAGMA foreign_keys=OFF applies to
// every replayed statement.
func openFileDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	return db
}
