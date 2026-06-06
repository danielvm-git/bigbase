package db

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Migration describes a single versioned schema migration.
// Version must be a sortable string (e.g. "001", "002", "2024-01-15_create_users").
// Up is the SQL applied when migrating forward.
// Down is the SQL applied when rolling back.
type Migration struct {
	Version string
	Up      string
	Down    string
}

// MigrationRecord holds the state of an applied migration as stored in the DB.
type MigrationRecord struct {
	Version   string
	AppliedAt time.Time
}

// Migrator applies and tracks versioned migrations against a database/sql handle.
// It creates a schema_migrations table on first use.
type Migrator struct {
	db         *sql.DB
	migrations []Migration
}

// NewMigrator returns a Migrator backed by db with the given ordered migrations.
// The caller is responsible for ordering; NewMigrator will also sort by Version
// lexicographically as a safety measure.
func NewMigrator(db *sql.DB, migrations []Migration) *Migrator {
	sorted := make([]Migration, len(migrations))
	copy(sorted, migrations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Version < sorted[j].Version
	})
	return &Migrator{db: db, migrations: sorted}
}

// ensureTable creates the schema_migrations table if it does not exist.
func (m *Migrator) ensureTable() error {
	_, err := m.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT      NOT NULL PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("migrator: create schema_migrations: %w", err)
	}
	return nil
}

// applied returns the set of already-applied migration versions.
func (m *Migrator) applied() (map[string]bool, error) {
	rows, err := m.db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("migrator: query applied: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("migrator: scan version: %w", err)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrator: iterate applied: %w", err)
	}
	return applied, nil
}

// Up applies all unapplied migrations in ascending version order.
// Each migration runs in its own transaction; a failure stops the run and
// returns the error after rolling back that migration's transaction.
func (m *Migrator) Up() error {
	if err := m.ensureTable(); err != nil {
		return err
	}
	applied, err := m.applied()
	if err != nil {
		return err
	}

	for _, mg := range m.migrations {
		if applied[mg.Version] {
			continue
		}
		if strings.TrimSpace(mg.Up) == "" {
			return fmt.Errorf("migrator: migration %q has empty Up SQL", mg.Version)
		}

		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("migrator: begin tx for %q: %w", mg.Version, err)
		}
		if _, err := tx.Exec(mg.Up); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrator: apply %q: %w", mg.Version, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			mg.Version, time.Now().UTC(),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrator: record %q: %w", mg.Version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrator: commit %q: %w", mg.Version, err)
		}
	}
	return nil
}

// Down rolls back the last n applied migrations in descending version order.
// Pass n<=0 to roll back all applied migrations.
func (m *Migrator) Down(n int) error {
	if err := m.ensureTable(); err != nil {
		return err
	}
	applied, err := m.applied()
	if err != nil {
		return err
	}

	// Collect applied migrations in reverse order.
	var toRollback []Migration
	for i := len(m.migrations) - 1; i >= 0; i-- {
		if applied[m.migrations[i].Version] {
			toRollback = append(toRollback, m.migrations[i])
		}
	}

	if n > 0 && n < len(toRollback) {
		toRollback = toRollback[:n]
	}

	for _, mg := range toRollback {
		if strings.TrimSpace(mg.Down) == "" {
			return fmt.Errorf("migrator: migration %q has empty Down SQL", mg.Version)
		}

		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("migrator: begin tx for rollback %q: %w", mg.Version, err)
		}
		if _, err := tx.Exec(mg.Down); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrator: rollback %q: %w", mg.Version, err)
		}
		if _, err := tx.Exec(
			`DELETE FROM schema_migrations WHERE version = ?`,
			mg.Version,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrator: unrecord %q: %w", mg.Version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrator: commit rollback %q: %w", mg.Version, err)
		}
	}
	return nil
}

// Status returns all known migrations annotated with their applied state.
// Migrations not yet applied have a zero AppliedAt.
func (m *Migrator) Status() ([]MigrationRecord, error) {
	if err := m.ensureTable(); err != nil {
		return nil, err
	}

	rows, err := m.db.Query(`SELECT version, applied_at FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("migrator: status query: %w", err)
	}
	defer rows.Close()

	appliedMap := make(map[string]time.Time)
	for rows.Next() {
		var v string
		var at time.Time
		if err := rows.Scan(&v, &at); err != nil {
			return nil, fmt.Errorf("migrator: status scan: %w", err)
		}
		appliedMap[v] = at
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrator: status iterate: %w", err)
	}

	records := make([]MigrationRecord, 0, len(m.migrations))
	for _, mg := range m.migrations {
		rec := MigrationRecord{Version: mg.Version}
		if at, ok := appliedMap[mg.Version]; ok {
			rec.AppliedAt = at
		}
		records = append(records, rec)
	}
	return records, nil
}
