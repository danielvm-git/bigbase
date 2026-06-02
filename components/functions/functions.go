package functions

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/danielvm/bigbase/kernel"
)

var (
	errMissingName   = errors.New("name is required")
	errMissingSource = errors.New("source is required")
)

const version = "0.1.0"

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}

type DBer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Migrate(migration string) error
}

type Function struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Runtime   string            `json:"runtime"`
	Source    string            `json:"source"`
	Trigger   string            `json:"trigger"`
	Schedule  string            `json:"schedule,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Timeout   int               `json:"timeout"`
	CreatedAt string            `json:"created_at"`
}

type Functions struct {
	db       DBer
	logger   Logger
	timeout  int
	runtimes map[string]Runtime
}

type Options struct {
	DB      DBer
	Logger  Logger
	Timeout int
}

func New(opts Options) *Functions {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30
	}
	return &Functions{
		db:       opts.DB,
		logger:   logger,
		timeout:  timeout,
		runtimes: map[string]Runtime{"javascript": &jsRuntime{}},
	}
}

func (f *Functions) Name() string                  { return "functions" }
func (f *Functions) Version() string               { return version }
func (f *Functions) Dependencies() []string        { return []string{"db"} }
func (f *Functions) ConfigSchema() json.RawMessage { return nil }
func (f *Functions) Hooks() []kernel.HookDef       { return nil }

func (f *Functions) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (f *Functions) Start(ctx *kernel.Context) error {
	if err := f.db.Migrate(`CREATE TABLE IF NOT EXISTS functions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		runtime TEXT NOT NULL DEFAULT 'javascript',
		source TEXT NOT NULL,
		trigger TEXT NOT NULL DEFAULT 'http',
		schedule TEXT DEFAULT '',
		env TEXT DEFAULT '{}',
		timeout INTEGER NOT NULL DEFAULT 30,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
	)`); err != nil {
		return fmt.Errorf("migrate functions table: %w", err)
	}
	if err := f.db.Migrate(`CREATE TABLE IF NOT EXISTS function_executions (
		id TEXT PRIMARY KEY,
		function_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'success',
		logs TEXT NOT NULL DEFAULT '[]',
		error TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
		FOREIGN KEY(function_id) REFERENCES functions(id) ON DELETE CASCADE
	)`); err != nil {
		return fmt.Errorf("migrate function_executions table: %w", err)
	}
	f.logger.Info("functions component ready")
	return nil
}

func (f *Functions) Stop(ctx *kernel.Context) error {
	return nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (f *Functions) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/functions", f.handleFunctions)
	mux.HandleFunc("/api/functions/", f.handleFunctionByID)
	return mux
}

func (f *Functions) scanFunction(row interface {
	Scan(dest ...any) error
}) (Function, error) {
	var fn Function
	var envStr string
	err := row.Scan(&fn.ID, &fn.Name, &fn.Runtime, &fn.Source, &fn.Trigger, &fn.Schedule, &envStr, &fn.Timeout, &fn.CreatedAt)
	if err != nil {
		return fn, err
	}
	if err := json.Unmarshal([]byte(envStr), &fn.Env); err != nil {
		f.logger.Warn("unmarshal function env", "error", err)
	}
	if fn.Env == nil {
		fn.Env = map[string]string{}
	}
	return fn, nil
}

func (f *Functions) fetchFunctionByID(ctx context.Context, id string) (Function, error) {
	return f.scanFunction(f.db.QueryRowContext(ctx,
		"SELECT id, name, runtime, source, trigger, schedule, env, timeout, created_at FROM functions WHERE id = ?", id))
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
