package git

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
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

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type Repo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	OwnerID       int64  `json:"owner_id"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	Description   string `json:"description"`
	CreatedAt     string `json:"created_at"`
}

type Git struct {
	db      DBer
	logger  Logger
	dir     string
}

type Options struct {
	DB     DBer
	Logger Logger
	Dir    string
}

func New(opts Options) *Git {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	dir := opts.Dir
	if dir == "" {
		dir = "data/git"
	}
	return &Git{db: opts.DB, logger: logger, dir: dir}
}

func (g *Git) Name() string                    { return "git" }
func (g *Git) Version() string                 { return version }
func (g *Git) Dependencies() []string          { return []string{"db"} }
func (g *Git) ConfigSchema() json.RawMessage   { return nil }
func (g *Git) Hooks() []kernel.HookDef         { return nil }

func (g *Git) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (g *Git) Start(ctx *kernel.Context) error {
	if err := os.MkdirAll(g.dir, 0755); err != nil {
		return fmt.Errorf("create git dir: %w", err)
	}
	if err := g.db.Migrate(`CREATE TABLE IF NOT EXISTS git_repos (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		owner_id INTEGER NOT NULL,
		private INTEGER NOT NULL DEFAULT 1,
		default_branch TEXT NOT NULL DEFAULT 'main',
		description TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate git_repos table: %w", err)
	}
	if err := g.db.Migrate(`CREATE TABLE IF NOT EXISTS git_ssh_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		public_key TEXT NOT NULL,
		fingerprint TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate git_ssh_keys table: %w", err)
	}
	g.logger.Info("git component ready", "dir", g.dir)
	return nil
}

func (g *Git) Stop(ctx *kernel.Context) error {
	return nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (g *Git) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/git/repos", g.handleRepos)
	mux.HandleFunc("/api/git/repos/", g.handleRepoByID)
	return mux
}

func (g *Git) handleRepos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		g.listRepos(w, r)
	case "POST":
		g.createRepo(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (g *Git) createRepo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Private     bool   `json:"private"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	id, err := generateID()
	if err != nil {
		g.logger.Error("generate id", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	repoDir := filepath.Join(g.dir, id+".git")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		g.logger.Error("create repo dir", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if _, err := gogit.PlainInit(repoDir, true); err != nil {
		_ = os.RemoveAll(repoDir)
		g.logger.Error("init bare repo", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	private := 0
	if req.Private {
		private = 1
	}
	if _, err := g.db.ExecContext(r.Context(),
		"INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, req.Name, 0, private, "main", req.Description, now); err != nil {
		_ = os.RemoveAll(repoDir)
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "repo name already exists"})
			return
		}
		g.logger.Error("insert repo", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, Repo{
		ID:            id,
		Name:          req.Name,
		OwnerID:       0,
		Private:       req.Private,
		DefaultBranch: "main",
		Description:   req.Description,
		CreatedAt:     now,
	})
}

func (g *Git) listRepos(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := g.db.QueryContext(ctx, "SELECT id, name, owner_id, private, default_branch, description, created_at FROM git_repos ORDER BY name")
	if err != nil {
		g.logger.Error("list repos", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	repos := make([]Repo, 0)
	for rows.Next() {
		var r Repo
		var privateInt int
		if err := rows.Scan(&r.ID, &r.Name, &r.OwnerID, &privateInt, &r.DefaultBranch, &r.Description, &r.CreatedAt); err != nil {
			g.logger.Error("scan repo", "error", err)
			continue
		}
		r.Private = privateInt != 0
		repos = append(repos, r)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": repos})
}

func (g *Git) handleRepoByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/git/repos/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	switch r.Method {
	case "GET":
		g.getRepo(w, r, id)
	case "DELETE":
		g.deleteRepo(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (g *Git) getRepo(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var repo Repo
	var privateInt int
	err := g.db.QueryRowContext(ctx,
		"SELECT id, name, owner_id, private, default_branch, description, created_at FROM git_repos WHERE id = ?", id).
		Scan(&repo.ID, &repo.Name, &repo.OwnerID, &privateInt, &repo.DefaultBranch, &repo.Description, &repo.CreatedAt)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repo not found"})
		return
	}
	repo.Private = privateInt != 0
	writeJSON(w, http.StatusOK, repo)
}

func (g *Git) deleteRepo(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := g.db.ExecContext(ctx, "DELETE FROM git_repos WHERE id = ?", id)
	if err != nil {
		g.logger.Error("delete repo", "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repo not found"})
		return
	}

	_ = os.RemoveAll(filepath.Join(g.dir, id+".git"))
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
