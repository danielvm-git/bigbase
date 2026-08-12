package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/danielvm/bigbase/kernel"
)

// PostgresOptions configures the PostgresDB driver.
type PostgresOptions struct {
	DSN    string
	Logger kernel.Logger
}

// PostgresDB is a PostgreSQL driver that implements kernel.DBer using
// pgxpool for connection pooling. It wraps pgxpool via the pgx stdlib
// adapter so the existing database/sql surface area (sql.Rows, sql.Row,
// sql.Result) is preserved across the codebase.
type PostgresDB struct {
	dsn    string
	logger kernel.Logger
	pool   *pgxpool.Pool
	sqlDB  *sql.DB
}

// NewPostgres constructs a PostgresDB. Call Init and Start before use.
func NewPostgres(opts PostgresOptions) *PostgresDB {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	return &PostgresDB{
		dsn:    opts.DSN,
		logger: logger,
	}
}

// Init satisfies kernel.Component (no-op for postgres).
func (p *PostgresDB) Init(_ *kernel.Context, _ []byte) error {
	return nil
}

// Start opens the connection pool and pings the database.
func (p *PostgresDB) Start(_ *kernel.Context) error {
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(p.dsn)
	if err != nil {
		return fmt.Errorf("postgres: parse dsn: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("postgres: create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("postgres: ping: %w", err)
	}
	p.pool = pool
	// Expose pool through the database/sql adapter so callers can use
	// sql.Rows, sql.Row, and sql.Result types unchanged.
	p.sqlDB = stdlib.OpenDBFromPool(pool)
	p.logger.Info("postgres connected", "dsn", p.dsn)
	return nil
}

// Stop closes the pool and the sql.DB adapter.
func (p *PostgresDB) Stop(_ *kernel.Context) error {
	if p.sqlDB != nil {
		if err := p.sqlDB.Close(); err != nil {
			return fmt.Errorf("postgres: close sql.DB: %w", err)
		}
		p.sqlDB = nil
	}
	if p.pool != nil {
		p.pool.Close()
		p.pool = nil
	}
	return nil
}

// ExecContext executes a query with context, returning sql.Result.
func (p *PostgresDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return p.sqlDB.ExecContext(ctx, query, args...)
}

// QueryContext executes a query with context, returning *sql.Rows.
func (p *PostgresDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return p.sqlDB.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query with context, returning *sql.Row.
func (p *PostgresDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return p.sqlDB.QueryRowContext(ctx, query, args...)
}

// Exec executes a query without explicit context.
func (p *PostgresDB) Exec(query string, args ...any) (sql.Result, error) {
	return p.sqlDB.Exec(query, args...)
}

// Query executes a query without explicit context, returning *sql.Rows.
func (p *PostgresDB) Query(query string, args ...any) (*sql.Rows, error) {
	return p.sqlDB.Query(query, args...)
}

// QueryRow executes a query without explicit context, returning *sql.Row.
func (p *PostgresDB) QueryRow(query string, args ...any) *sql.Row {
	return p.sqlDB.QueryRow(query, args...)
}

// Migrate executes a raw SQL string (DDL or DML). Used for schema setup.
func (p *PostgresDB) Migrate(migration string) error {
	if _, err := p.sqlDB.Exec(migration); err != nil {
		return fmt.Errorf("postgres: migrate: %w", err)
	}
	return nil
}

// BeginTx starts a database transaction. It is the PostgreSQL driver's
// implementation of the optional kernel.TxBeginner seam.
func (p *PostgresDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return p.sqlDB.BeginTx(ctx, opts)
}
