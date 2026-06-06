package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const migrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	applied_at TEXT NOT NULL
)`

// MigrationRunner applies ordered SQL migration files from a directory.
// Files must be named NNN_description.up.sql / NNN_description.down.sql.
type MigrationRunner struct {
	db  *sql.DB
	dir string
}

// NewMigrationRunner returns a runner that reads files from dir.
func NewMigrationRunner(db *sql.DB, dir string) *MigrationRunner {
	return &MigrationRunner{db: db, dir: dir}
}

// Up applies all pending up-migrations.
func (r *MigrationRunner) Up(ctx context.Context) error {
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return err
	}
	applied, err := r.appliedVersions(ctx)
	if err != nil {
		return err
	}
	files, err := r.upFiles()
	if err != nil {
		return err
	}
	for _, f := range files {
		if applied[f.version] {
			continue
		}
		sql, err := os.ReadFile(filepath.Join(r.dir, f.filename))
		if err != nil {
			return fmt.Errorf("read %s: %w", f.filename, err)
		}
		if _, err := r.db.ExecContext(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply %s: %w", f.filename, err)
		}
		if _, err := r.db.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			f.version, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("record version %d: %w", f.version, err)
		}
	}
	return nil
}

// Down reverts the latest applied migration.
func (r *MigrationRunner) Down(ctx context.Context) error {
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return err
	}
	var latest int64
	err := r.db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&latest)
	if err != nil || latest == 0 {
		return nil // nothing to roll back
	}

	files, err := r.downFiles()
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.version != latest {
			continue
		}
		sql, err := os.ReadFile(filepath.Join(r.dir, f.filename))
		if err != nil {
			return fmt.Errorf("read %s: %w", f.filename, err)
		}
		if _, err := r.db.ExecContext(ctx, string(sql)); err != nil {
			return fmt.Errorf("revert %s: %w", f.filename, err)
		}
		if _, err := r.db.ExecContext(ctx,
			`DELETE FROM schema_migrations WHERE version = ?`, f.version); err != nil {
			return fmt.Errorf("remove version %d: %w", f.version, err)
		}
		return nil
	}
	return nil
}

// Status returns the current version and dirty flag.
// Version is 0 when no migrations have been applied.
func (r *MigrationRunner) Status(ctx context.Context) (version int64, dirty bool, err error) {
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return 0, false, err
	}
	row := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`)
	if err := row.Scan(&version); err != nil {
		return 0, false, fmt.Errorf("query version: %w", err)
	}
	return version, false, nil
}

func (r *MigrationRunner) ensureMigrationsTable(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, migrationsTable); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}
	return nil
}

func (r *MigrationRunner) appliedVersions(ctx context.Context) (map[int64]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query applied: %w", err)
	}
	defer func() { _ = rows.Close() }()

	applied := make(map[int64]bool)
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

type migrationFile struct {
	version  int64
	filename string
}

func (r *MigrationRunner) upFiles() ([]migrationFile, error) {
	return r.listFiles(".up.sql")
}

func (r *MigrationRunner) downFiles() ([]migrationFile, error) {
	return r.listFiles(".down.sql")
}

func (r *MigrationRunner) listFiles(suffix string) ([]migrationFile, error) {
	entries, err := os.ReadDir(r.dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	var files []migrationFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), suffix) {
			continue
		}
		versionStr := strings.SplitN(e.Name(), "_", 2)[0]
		v, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			continue // skip non-numeric prefixes
		}
		files = append(files, migrationFile{version: v, filename: e.Name()})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].version < files[j].version
	})
	return files, nil
}
