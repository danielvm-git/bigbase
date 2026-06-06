package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

// DriverSQLite and DriverPostgres are the supported driver identifiers.
const (
	DriverSQLite   = "sqlite"
	DriverPostgres = "postgres"
)

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}
func (noopLogger) Debug(msg string, args ...any) {}

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

// Options configures the DB component.
// Driver selects "sqlite" (default) or "postgres".
// DSN is the data source name: a file path for SQLite or a connection
// URL for PostgreSQL (postgres://user:pass@host/dbname).
// Path is kept for backward-compatibility with the SQLite :memory: path.
type Options struct {
	Driver string
	DSN    string
	Path   string // legacy SQLite path; superseded by DSN when Driver=="sqlite"
	Logger Logger
}

type DB struct {
	driver string
	dsn    string
	logger Logger
	db     *sql.DB
}

func New(opts Options) *DB {
	if opts.Logger == nil {
		opts.Logger = noopLogger{}
	}
	driver := opts.Driver
	if driver == "" {
		driver = DriverSQLite
	}
	// Resolve DSN: explicit DSN wins, then legacy Path, then default file.
	dsn := opts.DSN
	if dsn == "" {
		dsn = opts.Path
	}
	return &DB{
		driver: driver,
		dsn:    dsn,
		logger: opts.Logger,
	}
}

func (d *DB) Name() string                 { return "db" }
func (d *DB) Version() string              { return version }
func (d *DB) Dependencies() []string       { return nil }
func (d *DB) ConfigSchema() json.RawMessage { return nil }
func (d *DB) Hooks() []kernel.HookDef      { return nil }

func (d *DB) Init(_ *kernel.Context, _ json.RawMessage) error {
	if d.driver == DriverSQLite && d.dsn == "" {
		d.dsn = "bigbase.db"
	}
	return nil
}

func (d *DB) Start(_ *kernel.Context) error {
	var sqldb *sql.DB
	switch d.driver {
	case DriverPostgres:
		pool, err := pgxpool.New(context.Background(), d.dsn)
		if err != nil {
			return fmt.Errorf("postgres: create pool: %w", err)
		}
		if err := pool.Ping(context.Background()); err != nil {
			pool.Close()
			return fmt.Errorf("postgres: ping: %w", err)
		}
		sqldb = stdlib.OpenDBFromPool(pool)
		d.logger.Info("postgres connected", "dsn", d.dsn)
	default: // sqlite
		var err error
		sqldb, err = sql.Open("sqlite", d.dsn)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		sqldb.SetMaxOpenConns(1)
		sqldb.SetConnMaxLifetime(5 * time.Minute)
		d.logger.Info("database opened", "path", d.dsn)
	}
	d.db = sqldb
	return nil
}

func (d *DB) Stop(_ *kernel.Context) error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *DB) DB() *sql.DB {
	return d.db
}

func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

func (d *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

func (d *DB) Exec(query string, args ...any) (sql.Result, error) {
	return d.db.Exec(query, args...)
}

func (d *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return d.db.Query(query, args...)
}

func (d *DB) QueryRow(query string, args ...any) *sql.Row {
	return d.db.QueryRow(query, args...)
}

func (d *DB) Migrate(migration string) error {
	_, err := d.db.Exec(migration)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
