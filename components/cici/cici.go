package cici

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/kernel"
)

// ErrRepoNotFound is returned when a repo_id does not exist.
var ErrRepoNotFound = errors.New("repo not found")

// ErrAccessDenied is returned when the caller does not own the repo.
var ErrAccessDenied = errors.New("access denied")

// dangerousCmdPatterns matches shell commands that enable network exfiltration,
// reverse shells, or arbitrary code execution.
var dangerousCmdPatterns = regexp.MustCompile(`(?i)(\bcurl\b|\bwget\b|\bnc\b|\bncat\b|\bnetcat\b|\bssh\b|\|\s*(sh|bash|zsh|dash)\b|\beval\b|\bexec\b|\$\(|\x60)`)

const version = "0.1.0"

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
	Logger kernel.Logger
}

type CICI struct {
	db     DBer
	logger kernel.Logger
}

var _ kernel.Component = (*CICI)(nil)

func New(opts Options) *CICI {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	return &CICI{db: opts.DB, logger: logger}
}

func (c *CICI) Name() string                  { return "cici" }
func (c *CICI) Version() string               { return version }
func (c *CICI) Dependencies() []string        { return []string{"db"} }

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

// verifyRepoOwnership checks that the repo belongs to the caller's org.
// Returns ErrRepoNotFound if the repo does not exist, ErrAccessDenied if the
// caller's org does not match the repo's owner_id.
func (c *CICI) verifyRepoOwnership(ctx context.Context, repoID string) error {
	var ownerID int64
	err := c.db.QueryRowContext(ctx,
		"SELECT owner_id FROM git_repos WHERE id = ?", repoID).Scan(&ownerID)
	if err != nil {
		return ErrRepoNotFound
	}
	orgID, ok := auth.OrgIDFromContext(ctx)
	if !ok {
		return nil // no auth context (e.g., internal call) — allow
	}
	if ownerID != orgID {
		return ErrAccessDenied
	}
	return nil
}

// orgIDFromReq extracts the org_id from the request context for logging.
func orgIDFromReq(r *http.Request) int64 {
	orgID, _ := auth.OrgIDFromContext(r.Context())
	return orgID
}

// validateCommand checks a shell command for known-dangerous patterns.
// Returns an error if the command contains patterns that could enable
// network exfiltration, reverse shells, or arbitrary code execution.
func validateCommand(command string) error {
	if dangerousCmdPatterns.MatchString(command) {
		return fmt.Errorf("command contains potentially dangerous pattern: %s", command)
	}
	// Block command substitution and piping to shell
	trimmed := strings.TrimSpace(command)
	if strings.Contains(trimmed, "| sh") || strings.Contains(trimmed, "| bash") {
		return fmt.Errorf("command pipes to shell interpreter")
	}
	return nil
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
