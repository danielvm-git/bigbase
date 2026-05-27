package functions

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
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
	db      DBer
	logger  Logger
	timeout int
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
	return &Functions{db: opts.DB, logger: logger, timeout: timeout}
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
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate functions table: %w", err)
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

func (f *Functions) handleFunctions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		f.handleCreate(w, r)
	case "GET":
		f.handleList(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (f *Functions) handleFunctionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/functions/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	if len(parts) == 2 && parts[1] == "run" {
		if r.Method == "POST" {
			f.handleRun(w, r, id)
			return
		}
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	switch r.Method {
	case "GET":
		f.handleGet(w, r, id)
	case "PUT":
		f.handleUpdate(w, r, id)
	case "DELETE":
		f.handleDelete(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (f *Functions) handleCreate(w http.ResponseWriter, r *http.Request) {
	var fn Function
	if err := json.NewDecoder(r.Body).Decode(&fn); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if fn.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if fn.Source == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source is required"})
		return
	}
	if fn.Runtime == "" {
		fn.Runtime = "javascript"
	}
	if fn.Trigger == "" {
		fn.Trigger = "http"
	}
	if fn.Timeout == 0 {
		fn.Timeout = f.timeout
	}

	id, err := generateID()
	if err != nil {
		f.logger.Error("generate id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	fn.ID = id

	envJSON, err := json.Marshal(fn.Env)
	if err != nil {
		envJSON = []byte("{}")
	}

	_, err = f.db.ExecContext(r.Context(),
		"INSERT INTO functions (id, name, runtime, source, trigger, schedule, env, timeout, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))",
		fn.ID, fn.Name, fn.Runtime, fn.Source, fn.Trigger, fn.Schedule, string(envJSON), fn.Timeout)
	if err != nil {
		f.logger.Error("insert function", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	fn.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	writeJSON(w, http.StatusCreated, fn)
}

func (f *Functions) handleList(w http.ResponseWriter, r *http.Request) {
	rows, err := f.db.QueryContext(r.Context(),
		"SELECT id, name, runtime, source, trigger, schedule, env, timeout, created_at FROM functions ORDER BY created_at DESC")
	if err != nil {
		f.logger.Error("list functions", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	funcs := make([]Function, 0)
	for rows.Next() {
		var fn Function
		var envStr string
		if err := rows.Scan(&fn.ID, &fn.Name, &fn.Runtime, &fn.Source, &fn.Trigger, &fn.Schedule, &envStr, &fn.Timeout, &fn.CreatedAt); err != nil {
			f.logger.Error("scan function", "error", err)
			continue
		}
		_ = json.Unmarshal([]byte(envStr), &fn.Env)
		funcs = append(funcs, fn)
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": funcs})
}

func (f *Functions) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	var fn Function
	var envStr string
	err := f.db.QueryRowContext(r.Context(),
		"SELECT id, name, runtime, source, trigger, schedule, env, timeout, created_at FROM functions WHERE id = ?", id).
		Scan(&fn.ID, &fn.Name, &fn.Runtime, &fn.Source, &fn.Trigger, &fn.Schedule, &envStr, &fn.Timeout, &fn.CreatedAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}
	_ = json.Unmarshal([]byte(envStr), &fn.Env)
	writeJSON(w, http.StatusOK, fn)
}

func (f *Functions) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	var fn Function
	if err := json.NewDecoder(r.Body).Decode(&fn); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	envJSON, err := json.Marshal(fn.Env)
	if err != nil {
		envJSON = []byte("{}")
	}

	result, err := f.db.ExecContext(r.Context(),
		"UPDATE functions SET name=?, runtime=?, source=?, trigger=?, schedule=?, env=?, timeout=? WHERE id=?",
		fn.Name, fn.Runtime, fn.Source, fn.Trigger, fn.Schedule, string(envJSON), fn.Timeout, id)
	if err != nil {
		f.logger.Error("update function", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}

	fn.ID = id
	writeJSON(w, http.StatusOK, fn)
}

func (f *Functions) handleDelete(w http.ResponseWriter, r *http.Request, id string) {
	result, err := f.db.ExecContext(r.Context(), "DELETE FROM functions WHERE id = ?", id)
	if err != nil {
		f.logger.Error("delete function", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (f *Functions) handleRun(w http.ResponseWriter, r *http.Request, id string) {
	var fn Function
	var envStr string
	err := f.db.QueryRowContext(r.Context(),
		"SELECT id, name, runtime, source, trigger, schedule, env, timeout FROM functions WHERE id = ?", id).
		Scan(&fn.ID, &fn.Name, &fn.Runtime, &fn.Source, &fn.Trigger, &fn.Schedule, &envStr, &fn.Timeout)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "function not found"})
		return
	}
	_ = json.Unmarshal([]byte(envStr), &fn.Env)

	goja, ok := runtimes[fn.Runtime]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported runtime: " + fn.Runtime})
		return
	}

	output, execErr := goja.Execute(fn.Source, fn.Timeout)
	if execErr != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"logs":   output.Logs,
			"error":  execErr.Error(),
			"result": nil,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"logs":   output.Logs,
		"error":  nil,
		"result": output.Result,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
