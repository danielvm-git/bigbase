package forge

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
	logger Logger
}

type Options struct {
	DB     DBer
	Logger Logger
}

func New(opts Options) *Forge {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	return &Forge{db: opts.DB, logger: logger}
}

func (f *Forge) Name() string                    { return "forge" }
func (f *Forge) Version() string                 { return version }
func (f *Forge) Dependencies() []string          { return []string{"db"} }
func (f *Forge) ConfigSchema() json.RawMessage   { return nil }
func (f *Forge) Hooks() []kernel.HookDef         { return nil }

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
	f.logger.Info("forge component ready")
	return nil
}

func (f *Forge) Stop(ctx *kernel.Context) error {
	return nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
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
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (f *Forge) createIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RepoID      string   `json:"repo_id"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Status      string   `json:"status"`
		Labels      []string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.RepoID == "" || req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id and title are required"})
		return
	}
	if req.Status == "" {
		req.Status = "open"
	}

	id, err := generateID()
	if err != nil {
		f.logger.Error("generate id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	labelsJSON, _ := json.Marshal(req.Labels)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := f.db.ExecContext(r.Context(),
		"INSERT INTO forge_issues (id, repo_id, title, description, status, labels, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, req.RepoID, req.Title, req.Description, req.Status, string(labelsJSON), now, now); err != nil {
		f.logger.Error("insert issue", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, Issue{
		ID: id, RepoID: req.RepoID, Title: req.Title,
		Description: req.Description, Status: req.Status,
		Labels: req.Labels, CreatedAt: now, UpdatedAt: now,
	})
}

func (f *Forge) listIssues(w http.ResponseWriter, r *http.Request) {
	repoID := r.URL.Query().Get("repo_id")
	if repoID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id query parameter required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := f.db.QueryContext(ctx,
		"SELECT id, repo_id, title, description, status, labels, created_at, updated_at FROM forge_issues WHERE repo_id = ? ORDER BY created_at DESC", repoID)
	if err != nil {
		f.logger.Error("list issues", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	issues := make([]Issue, 0)
	for rows.Next() {
		var i Issue
		var labelsStr string
		if err := rows.Scan(&i.ID, &i.RepoID, &i.Title, &i.Description, &i.Status, &labelsStr, &i.CreatedAt, &i.UpdatedAt); err != nil {
			f.logger.Error("scan issue", "error", err)
			continue
		}
		json.Unmarshal([]byte(labelsStr), &i.Labels)
		issues = append(issues, i)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": issues})
}

func (f *Forge) handleIssueByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/forge/issues/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
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
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (f *Forge) getIssue(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var i Issue
	var labelsStr string
	err := f.db.QueryRowContext(ctx,
		"SELECT id, repo_id, title, description, status, labels, created_at, updated_at FROM forge_issues WHERE id = ?", id).
		Scan(&i.ID, &i.RepoID, &i.Title, &i.Description, &i.Status, &labelsStr, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "issue not found"})
		return
	}
	json.Unmarshal([]byte(labelsStr), &i.Labels)
	writeJSON(w, http.StatusOK, i)
}

func (f *Forge) updateIssue(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Status      string   `json:"status"`
		Labels      []string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if req.Labels != nil {
		labelsJSON, _ := json.Marshal(req.Labels)
		_, err := f.db.ExecContext(r.Context(),
			"UPDATE forge_issues SET title=COALESCE(NULLIF(?,''),title), description=COALESCE(NULLIF(?,''),description), status=COALESCE(NULLIF(?,''),status), labels=?, updated_at=? WHERE id=?",
			req.Title, req.Description, req.Status, string(labelsJSON), now, id)
		if err != nil {
			f.logger.Error("update issue", "id", id, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
	} else {
		_, err := f.db.ExecContext(r.Context(),
			"UPDATE forge_issues SET title=COALESCE(NULLIF(?,''),title), description=COALESCE(NULLIF(?,''),description), status=COALESCE(NULLIF(?,''),status), updated_at=? WHERE id=?",
			req.Title, req.Description, req.Status, now, id)
		if err != nil {
			f.logger.Error("update issue", "id", id, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (f *Forge) handleComments(w http.ResponseWriter, r *http.Request, issueID string) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}

	id, err := generateID()
	if err != nil {
		f.logger.Error("generate id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := f.db.ExecContext(r.Context(),
		"INSERT INTO forge_comments (id, issue_id, content, created_at) VALUES (?, ?, ?, ?)",
		id, issueID, req.Content, now); err != nil {
		f.logger.Error("insert comment", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, Comment{
		ID: id, IssueID: issueID, Content: req.Content, CreatedAt: now,
	})
}

func (f *Forge) handleLabels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		f.listLabels(w, r)
	case "POST":
		f.createLabel(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (f *Forge) createLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RepoID string `json:"repo_id"`
		Name   string `json:"name"`
		Color  string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.RepoID == "" || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id and name are required"})
		return
	}
	if req.Color == "" {
		req.Color = "#cccccc"
	}

	id, err := generateID()
	if err != nil {
		f.logger.Error("generate id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if _, err := f.db.ExecContext(r.Context(),
		"INSERT INTO forge_labels (id, repo_id, name, color) VALUES (?, ?, ?, ?)",
		id, req.RepoID, req.Name, req.Color); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "label already exists"})
			return
		}
		f.logger.Error("insert label", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, Label{ID: id, RepoID: req.RepoID, Name: req.Name, Color: req.Color})
}

func (f *Forge) listLabels(w http.ResponseWriter, r *http.Request) {
	repoID := r.URL.Query().Get("repo_id")
	if repoID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := f.db.QueryContext(ctx,
		"SELECT id, repo_id, name, color FROM forge_labels WHERE repo_id = ? ORDER BY name", repoID)
	if err != nil {
		f.logger.Error("list labels", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	labels := make([]Label, 0)
	for rows.Next() {
		var l Label
		if err := rows.Scan(&l.ID, &l.RepoID, &l.Name, &l.Color); err != nil {
			f.logger.Error("scan label", "error", err)
			continue
		}
		labels = append(labels, l)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": labels})
}

func (f *Forge) handleBoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	repoID := r.URL.Query().Get("repo_id")
	if repoID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	statuses := []string{"open", "in_progress", "review", "closed"}
	board := make(map[string][]Issue, len(statuses))

	for _, status := range statuses {
		rows, err := f.db.QueryContext(ctx,
			"SELECT id, repo_id, title, description, status, labels, created_at, updated_at FROM forge_issues WHERE repo_id = ? AND status = ? ORDER BY updated_at DESC",
			repoID, status)
		if err != nil {
			f.logger.Error("query board column", "status", status, "error", err)
			continue
		}

		issues := make([]Issue, 0)
		for rows.Next() {
			var i Issue
			var labelsStr string
			if err := rows.Scan(&i.ID, &i.RepoID, &i.Title, &i.Description, &i.Status, &labelsStr, &i.CreatedAt, &i.UpdatedAt); err != nil {
				f.logger.Error("scan issue", "error", err)
				continue
			}
			json.Unmarshal([]byte(labelsStr), &i.Labels)
			issues = append(issues, i)
		}
		_ = rows.Close()
		board[status] = issues
	}

	writeJSON(w, http.StatusOK, board)
}

func (f *Forge) handleWiki(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimPrefix(r.URL.Path, "/api/forge/wiki/")
	if title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "page title required"})
		return
	}

	switch r.Method {
	case "GET":
		f.getWiki(w, r, title)
	case "PUT":
		f.saveWiki(w, r, title)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (f *Forge) getWiki(w http.ResponseWriter, r *http.Request, title string) {
	repoID := r.URL.Query().Get("repo_id")
	if repoID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var page WikiPage
	err := f.db.QueryRowContext(ctx,
		"SELECT title, repo_id, content, updated_at FROM forge_wiki WHERE title = ? AND repo_id = ?", title, repoID).
		Scan(&page.Title, &page.RepoID, &page.Content, &page.UpdatedAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (f *Forge) saveWiki(w http.ResponseWriter, r *http.Request, title string) {
	var req struct {
		RepoID  string `json:"repo_id"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.RepoID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo_id is required"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := f.db.ExecContext(r.Context(),
		"INSERT INTO forge_wiki (title, repo_id, content, updated_at) VALUES (?, ?, ?, ?) "+
			"ON CONFLICT(title, repo_id) DO UPDATE SET content=excluded.content, updated_at=excluded.updated_at",
		title, req.RepoID, req.Content, now)
	if err != nil {
		f.logger.Error("save wiki", "title", title, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, WikiPage{Title: title, RepoID: req.RepoID, Content: req.Content, UpdatedAt: now})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
