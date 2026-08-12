package kernel

import (
	"context"
	"database/sql"
)

// QueryExecDBer abstracts the minimal database query and exec operations.
// Both *sql.DB and kernel.DBer implement this interface, making it safe
// for component tests that pass raw *sql.DB values.
type QueryExecDBer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// DBer abstracts database operations used across components.
// Both the SQLite and PostgreSQL drivers implement this interface.
type DBer interface {
	QueryExecDBer
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Migrate(migration string) error
}

// TxBeginner is an optional seam for databases that can start transactions.
// It is intentionally separate from DBer: only components that need atomic
// multi-statement operations (for example SecretManager first-key creation)
// type-assert to it, so unrelated component interfaces stay narrow. Both the
// SQLite and PostgreSQL drivers implement it, and so does *sql.DB, which keeps
// the seam usable by component tests that pass raw handles.
type TxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
