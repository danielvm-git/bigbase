package cici

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

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

type DBer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Migrate(migration string) error
}

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
	Name string          `yaml:"name"`
	On   yaml.Node       `yaml:"on"`
	Jobs map[string]job  `yaml:"jobs"`
}

type job struct {
	RunsOn string `yaml:"runs-on"`
	Steps  []step `yaml:"steps"`
}

type step struct {
	Run     string `yaml:"run"`
	Uses    string `yaml:"uses"`
	With    map[string]string `yaml:"with"`
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

func (c *CICI) Name() string                    { return "cici" }
func (c *CICI) Version() string                 { return version }
func (c *CICI) Dependencies() []string          { return []string{"db"} }
func (c *CICI) ConfigSchema() json.RawMessage   { return nil }
func (c *CICI) Hooks() []kernel.HookDef         { return nil }

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
	mux.HandleFunc("/api/cici/runs", c.handleRuns)
	mux.HandleFunc("/api/cici/runs/", c.handleRunsByID)
	mux.HandleFunc("/api/cici/", c.handleRepoWorkflows)
	return mux
}

func (c *CICI) handleRepoWorkflows(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/cici/")
	parts := strings.SplitN(path, "/", 4)
	if len(parts) < 2 || parts[0] == "" || parts[1] != "workflows" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	repoID := parts[0]

	if r.Method == "GET" {
		c.listWorkflows(w, r, repoID)
		return
	}
	if r.Method == "PUT" {
		c.saveWorkflow(w, r, repoID)
		return
	}

	if len(parts) >= 4 && parts[1] == "workflows" && parts[3] == "run" && r.Method == "POST" {
		c.triggerRun(w, r, repoID, parts[2])
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func (c *CICI) saveWorkflow(w http.ResponseWriter, r *http.Request, repoID string) {
	var req struct {
		Name string `json:"name"`
		YAML string `json:"yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" || req.YAML == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and yaml are required"})
		return
	}

	var wf workflowYAML
	if err := yaml.Unmarshal([]byte(req.YAML), &wf); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid yaml: " + err.Error()})
		return
	}

	id, err := generateID()
	if err != nil {
		c.logger.Error("generate id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := c.db.ExecContext(r.Context(),
		"INSERT INTO cici_workflows (id, repo_id, name, yaml, created_at) VALUES (?, ?, ?, ?, ?)",
		id, repoID, req.Name, req.YAML, now); err != nil {
		c.logger.Error("insert workflow", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, WorkflowDef{
		ID: id, RepoID: repoID, Name: req.Name, YAML: req.YAML,
	})
}

func (c *CICI) listWorkflows(w http.ResponseWriter, r *http.Request, repoID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := c.db.QueryContext(ctx,
		"SELECT id, repo_id, name, yaml FROM cici_workflows WHERE repo_id = ? ORDER BY name", repoID)
	if err != nil {
		c.logger.Error("list workflows", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	workflows := make([]WorkflowDef, 0)
	for rows.Next() {
		var w WorkflowDef
		if err := rows.Scan(&w.ID, &w.RepoID, &w.Name, &w.YAML); err != nil {
			c.logger.Error("scan workflow", "error", err)
			continue
		}
		workflows = append(workflows, w)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": workflows})
}

func (c *CICI) fetchWorkflow(ctx context.Context, id string) (*WorkflowDef, error) {
	var w WorkflowDef
	err := c.db.QueryRowContext(ctx,
		"SELECT id, repo_id, name, yaml FROM cici_workflows WHERE id = ?", id).
		Scan(&w.ID, &w.RepoID, &w.Name, &w.YAML)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (c *CICI) triggerRun(w http.ResponseWriter, r *http.Request, repoID, workflowID string) {
	var req struct {
		Event string `json:"event"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Event == "" {
		req.Event = "manual"
	}

	wf, err := c.fetchWorkflow(r.Context(), workflowID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "workflow not found"})
		return
	}

	runID, err := generateID()
	if err != nil {
		c.logger.Error("generate run id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := c.db.ExecContext(r.Context(),
		"INSERT INTO cici_runs (id, workflow_id, event, status, started_at) VALUES (?, ?, ?, 'running', ?)",
		runID, workflowID, req.Event, now); err != nil {
		c.logger.Error("insert run", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	go c.executeRun(runID, wf)

	writeJSON(w, http.StatusAccepted, PipelineRun{
		ID: runID, WorkflowID: workflowID, Event: req.Event,
		Status: "running", StartedAt: now,
	})
}

func (c *CICI) executeRun(runID string, wf *WorkflowDef) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var wfYAML workflowYAML
	if err := yaml.Unmarshal([]byte(wf.YAML), &wfYAML); err != nil {
		c.finishRun(ctx, runID, "failure")
		return
	}

	overallStatus := "success"
	for jobName, jobDef := range wfYAML.Jobs {
		for i, stepDef := range jobDef.Steps {
			if stepDef.Run == "" {
				continue
			}
			stepName := fmt.Sprintf("%s.step-%d", jobName, i)
			if err := c.executeStep(ctx, runID, stepName, stepDef.Run); err != nil {
				c.logger.Error("step failed", "run_id", runID, "step", stepName, "error", err)
				c.appendLog(ctx, runID, stepName, "FAIL: "+err.Error())
				overallStatus = "failure"
				break
			}
		}
	}

	c.finishRun(ctx, runID, overallStatus)
}

func (c *CICI) executeStep(ctx context.Context, runID, name, command string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += strings.TrimSpace(stderr.String())
	}

	c.appendLog(ctx, runID, name, output)
	return err
}

func (c *CICI) appendLog(ctx context.Context, runID, step, output string) {
	_, err := c.db.ExecContext(ctx,
		"INSERT INTO cici_logs (run_id, step, output) VALUES (?, ?, ?)",
		runID, step, output)
	if err != nil {
		c.logger.Error("append log", "run_id", runID, "error", err)
	}
}

func (c *CICI) finishRun(ctx context.Context, runID, status string) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := c.db.ExecContext(ctx,
		"UPDATE cici_runs SET status = ?, finished_at = ? WHERE id = ?",
		status, now, runID)
	if err != nil {
		c.logger.Error("finish run", "run_id", runID, "error", err)
	}
}

func (c *CICI) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := c.db.QueryContext(ctx,
		"SELECT id, workflow_id, event, status, started_at, finished_at FROM cici_runs ORDER BY started_at DESC")
	if err != nil {
		c.logger.Error("list runs", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	runs := make([]PipelineRun, 0)
	for rows.Next() {
		var r PipelineRun
		if err := rows.Scan(&r.ID, &r.WorkflowID, &r.Event, &r.Status, &r.StartedAt, &r.FinishedAt); err != nil {
			c.logger.Error("scan run", "error", err)
			continue
		}
		runs = append(runs, r)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": runs})
}

func (c *CICI) handleRunsByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/cici/runs/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}
	runID := parts[0]

	if len(parts) == 2 && parts[1] == "logs" {
		c.getRunLogs(w, r, runID)
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func (c *CICI) getRunLogs(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := c.db.QueryContext(ctx,
		"SELECT id, run_id, step, output FROM cici_logs WHERE run_id = ? ORDER BY id", runID)
	if err != nil {
		c.logger.Error("get logs", "run_id", runID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	logs := make([]LogEntry, 0)
	for rows.Next() {
		var l LogEntry
		if err := rows.Scan(&l.ID, &l.RunID, &l.Step, &l.Output); err != nil {
			c.logger.Error("scan log", "error", err)
			continue
		}
		logs = append(logs, l)
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
