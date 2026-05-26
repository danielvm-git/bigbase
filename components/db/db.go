package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type Options struct {
	Path   string
	Logger Logger
}

type DB struct {
	path   string
	logger Logger
	db     *sql.DB
}

func New(opts Options) *DB {
	return &DB{
		path:   opts.Path,
		logger: opts.Logger,
	}
}

func (d *DB) Name() string                 { return "db" }
func (d *DB) Version() string              { return version }
func (d *DB) Dependencies() []string       { return nil }
func (d *DB) ConfigSchema() json.RawMessage { return nil }
func (d *DB) Hooks() []kernel.HookDef      { return nil }

func (d *DB) Init(ctx *kernel.Context, config json.RawMessage) error {
	if d.path == "" {
		d.path = "bigbase.db"
	}
	return nil
}

func (d *DB) Start(ctx *kernel.Context) error {
	sqldb, err := sql.Open("sqlite", d.path)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	sqldb.SetMaxOpenConns(1)
	d.db = sqldb
	d.logger.Info("database opened", "path", d.path)
	return nil
}

func (d *DB) Stop(ctx *kernel.Context) error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *DB) DB() *sql.DB {
	return d.db
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
