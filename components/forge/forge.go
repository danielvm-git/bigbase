package forge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type Issue struct {
	ID          string   `json:"id"`
	RepoID      string   `json:"repo_id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
	Labels      []string `json:"labels,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type Label struct {
	ID     string `json:"id"`
	RepoID string `json:"repo_id"`
	Name   string `json:"name"`
	Color  string `json:"color"`
}

type Comment struct {
	ID        string `json:"id"`
	IssueID   string `json:"issue_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type WikiPage struct {
	Title     string `json:"title"`
	RepoID    string `json:"repo_id"`
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}

type Forge struct {
	db     DBer
	logger kernel.Logger
}

var _ kernel.Component = (*Forge)(nil)

type Options struct {
	DB     DBer
	Logger kernel.Logger
}

func New(opts Options) *Forge {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	return &Forge{db: opts.DB, logger: logger}
}

func (f *Forge) Name() string                  { return "forge" }
func (f *Forge) Version() string               { return version }
func (f *Forge) Dependencies() []string        { return []string{"db"} }

func (f *Forge) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (f *Forge) Start(ctx *kernel.Context) error {
	for _, m := range []string{
		`CREATE TABLE IF NOT EXISTS forge_issues (
			id TEXT PRIMARY KEY, repo_id TEXT NOT NULL, title TEXT NOT NULL,
			description TEXT DEFAULT '', status TEXT NOT NULL DEFAULT 'open',
			labels TEXT DEFAULT '[]', created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS forge_labels (
			id TEXT PRIMARY KEY, repo_id TEXT NOT NULL, name TEXT NOT NULL,
			color TEXT NOT NULL DEFAULT '#cccccc',
			UNIQUE(repo_id, name))`,
		`CREATE TABLE IF NOT EXISTS forge_comments (
			id TEXT PRIMARY KEY, issue_id TEXT NOT NULL, content TEXT NOT NULL,
			created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS forge_wiki (
			title TEXT NOT NULL, repo_id TEXT NOT NULL, content TEXT NOT NULL,
			updated_at TEXT NOT NULL, PRIMARY KEY(title, repo_id))`,
	} {
		if err := f.db.Migrate(m); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	for _, idx := range []string{
		"CREATE INDEX IF NOT EXISTS idx_forge_issues_repo_status ON forge_issues(repo_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_forge_labels_repo ON forge_labels(repo_id)",
		"CREATE INDEX IF NOT EXISTS idx_forge_comments_issue ON forge_comments(issue_id)",
	} {
		_ = f.db.Migrate(idx)
	}
	f.logger.Info("forge component ready")
	return nil
}

func (f *Forge) Stop(ctx *kernel.Context) error {
	return nil
}

func (f *Forge) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/forge/issues/", f.handleIssueByID)
	mux.HandleFunc("/api/forge/issues", f.handleIssues)
	mux.HandleFunc("/api/forge/labels", f.handleLabels)
	mux.HandleFunc("/api/forge/board", f.handleBoard)
	mux.HandleFunc("/api/forge/wiki/", f.handleWiki)
	return mux
}

func (f *Forge) handleIssues(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		f.listIssues(w, r)
	case "POST":
		f.createIssue(w, r)
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
func (f *Forge) handleIssueByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/forge/issues/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	if id == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	if len(parts) == 2 && parts[1] == "comments" {
		f.handleComments(w, r, id)
		return
	}

	switch r.Method {
	case "GET":
		f.getIssue(w, r, id)
	case "PATCH":
		f.updateIssue(w, r, id)
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
func (f *Forge) handleLabels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		f.listLabels(w, r)
	case "POST":
		f.createLabel(w, r)
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
func (f *Forge) handleBoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	repoID := r.URL.Query().Get("repo_id")
	if repoID == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id required"})
		return
	}

	statuses := []string{"open", "in_progress", "review", "closed"}
	board := make(map[string][]Issue, len(statuses))
	for _, status := range statuses {
		issues, err := f.fetchIssues(r.Context(), repoID, status)
		if err != nil {
			f.logger.Error("query board column", "status", status, "error", err)
			kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load board"})
			return
		}
		board[status] = issues
	}
	kernel.WriteJSON(w, http.StatusOK, board)
}
func (f *Forge) handleWiki(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimPrefix(r.URL.Path, "/api/forge/wiki/")
	if title == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "page title required"})
		return
	}

	switch r.Method {
	case "GET":
		f.getWiki(w, r, title)
	case "PUT":
		f.saveWiki(w, r, title)
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
