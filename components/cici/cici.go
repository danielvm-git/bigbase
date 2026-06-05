package cici

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/danielvm/bigbase/kernel"
	"gopkg.in/yaml.v3"
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

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type WorkflowDef struct {
	ID     string `json:"id"`
	RepoID string `json:"repo_id"`
	Name   string `json:"name"`
	YAML   string `json:"yaml"`
}

type PipelineRun struct {
	ID         string  `json:"id"`
	WorkflowID string  `json:"workflow_id"`
	Event      string  `json:"event"`
	Status     string  `json:"status"`
	StartedAt  string  `json:"started_at"`
	FinishedAt *string `json:"finished_at"`
}

type LogEntry struct {
	ID     int64  `json:"id"`
	RunID  string `json:"run_id"`
	Step   string `json:"step"`
	Output string `json:"output"`
}

type workflowYAML struct {
	Name string         `yaml:"name"`
	On   yaml.Node      `yaml:"on"`
	Jobs map[string]job `yaml:"jobs"`
}

type job struct {
	RunsOn string `yaml:"runs-on"`
	Steps  []step `yaml:"steps"`
}

type step struct {
	Run  string            `yaml:"run"`
	Uses string            `yaml:"uses"`
	With map[string]string `yaml:"with"`
}

type Options struct {
	DB     DBer
	Logger Logger
}

type CICI struct {
	db     DBer
	logger Logger
}

func New(opts Options) *CICI {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	return &CICI{db: opts.DB, logger: logger}
}

func (c *CICI) Name() string                  { return "cici" }
func (c *CICI) Version() string               { return version }
func (c *CICI) Dependencies() []string        { return []string{"db"} }
func (c *CICI) ConfigSchema() json.RawMessage { return nil }
func (c *CICI) Hooks() []kernel.HookDef       { return nil }

func (c *CICI) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (c *CICI) Start(ctx *kernel.Context) error {
	for _, m := range []string{
		`CREATE TABLE IF NOT EXISTS cici_workflows (
			id TEXT PRIMARY KEY, repo_id TEXT NOT NULL, name TEXT NOT NULL,
			yaml TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS cici_runs (
			id TEXT PRIMARY KEY, workflow_id TEXT NOT NULL, event TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending', started_at TEXT NOT NULL,
			finished_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS cici_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT, run_id TEXT NOT NULL,
			step TEXT NOT NULL, output TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')))`,
	} {
		if err := c.db.Migrate(m); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	c.logger.Info("cici component ready")
	return nil
}

func (c *CICI) Stop(ctx *kernel.Context) error {
	return nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (c *CICI) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/cici/runs", c.handleRuns)
	mux.HandleFunc("GET /api/cici/runs/{id}/logs", c.getRunLogs)
	mux.HandleFunc("GET /api/cici/{repo}/workflows", c.listWorkflows)
	mux.HandleFunc("PUT /api/cici/{repo}/workflows", c.saveWorkflow)
	mux.HandleFunc("POST /api/cici/{repo}/workflows/{id}/run", c.triggerRun)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
