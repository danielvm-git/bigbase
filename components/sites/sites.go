package sites

import (
	"context"
	"crypto/rand"
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
	Debug(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)   {}
func (noopLogger) Warn(msg string, args ...any)   {}
func (noopLogger) Error(msg string, args ...any)  {}
func (noopLogger) Debug(msg string, args ...any)  {}

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type Deployment struct {
	ID        string `json:"id"`
	RepoID    string `json:"repo_id"`
	Branch    string `json:"branch"`
	CommitSHA string `json:"commit_sha"`
	Status    string `json:"status"`
	URL       string `json:"url"`
	Port      int    `json:"port"`
	AppType   string `json:"app_type"`
	CreatedAt string `json:"created_at"`
}

type Site struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	FullName          string      `json:"full_name"`
	GitRepoID         string      `json:"git_repo_id"`
	ProductionBranch  string      `json:"production_branch"`
	RootPath          string      `json:"root_path"`
	GitHubConnected   bool        `json:"github_connected,omitempty"`
	LatestDeployment  *Deployment `json:"latest_deployment,omitempty"`
}

type DeployTrigger func(ctx context.Context, repoID, branch string) (*Deployment, error)

type Sites struct {
	db            DBer
	logger        Logger
	triggerDeploy DeployTrigger
}

type Options struct {
	DB            DBer
	Logger        Logger
	TriggerDeploy DeployTrigger
}

func New(opts Options) *Sites {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	return &Sites{db: opts.DB, logger: logger, triggerDeploy: opts.TriggerDeploy}
}

func (s *Sites) Name() string                   { return "sites" }
func (s *Sites) Version() string                { return version }
func (s *Sites) Dependencies() []string         { return []string{"db"} }
func (s *Sites) ConfigSchema() json.RawMessage  { return nil }
func (s *Sites) Hooks() []kernel.HookDef        { return nil }

func (s *Sites) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (s *Sites) Start(ctx *kernel.Context) error {
	if err := s.db.Migrate(`CREATE TABLE IF NOT EXISTS sites (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		git_repo_id TEXT NOT NULL,
		production_branch TEXT NOT NULL DEFAULT 'main',
		root_path TEXT NOT NULL DEFAULT './',
		github_full_name TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate sites: %w", err)
	}
	if err := s.migrateDomains(); err != nil {
		return fmt.Errorf("migrate site_domains: %w", err)
	}
	s.logger.Info("sites component ready")
	return nil
}

func (s *Sites) Stop(ctx *kernel.Context) error {
	return nil
}

func (s *Sites) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sites", s.handleSites)
	mux.HandleFunc("/api/sites/", s.handleSiteByID)
	return mux
}


func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Sites) handleSites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listSites(w, r)
	case http.MethodPost:
		s.createSite(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Sites) handleSiteByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}
	id := parts[0]

	if len(parts) == 2 && parts[1] == "deploy" && r.Method == http.MethodPost {
		s.redeploySite(w, r, id)
		return
	}

	// Route domain sub-paths: /api/sites/{id}/domains[/{domain}/verify]
	if len(parts) >= 2 && parts[1] == "domains" {
		s.handleDomains(w, r)
		return
	}

	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	s.getSite(w, r, id)
}

func (s *Sites) listSites(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.name, s.git_repo_id, s.production_branch, s.root_path,
			COALESCE(NULLIF(s.github_full_name, ''), g.name, s.name)
		FROM sites s
		LEFT JOIN git_repos g ON g.id = s.git_repo_id
		ORDER BY s.name`)
	if err != nil {
		s.logger.Error("list sites", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	sites := make([]Site, 0)
	for rows.Next() {
		var site Site
		if err := rows.Scan(&site.ID, &site.Name, &site.GitRepoID, &site.ProductionBranch,
			&site.RootPath, &site.FullName); err != nil {
			continue
		}
		if site.FullName != "" {
			site.GitHubConnected = strings.Contains(site.FullName, "/")
		}
		dep, _ := s.latestDeployment(ctx, site.GitRepoID)
		site.LatestDeployment = dep
		sites = append(sites, site)
	}

	if len(sites) == 0 {
		sites = s.listSitesFromRepos(ctx)
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": sites})
}

func (s *Sites) listSitesFromRepos(ctx context.Context) []Site {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, default_branch FROM git_repos ORDER BY name")
	if err != nil {
		return []Site{}
	}
	defer func() { _ = rows.Close() }()

	out := make([]Site, 0)
	for rows.Next() {
		var id, name, branch string
		if err := rows.Scan(&id, &name, &branch); err != nil {
			continue
		}
		site := Site{
			ID:               id,
			Name:             name,
			FullName:         name,
			GitRepoID:        id,
			ProductionBranch: branch,
			RootPath:         "./",
		}
		site.LatestDeployment, _ = s.latestDeployment(ctx, id)
		out = append(out, site)
	}
	return out
}

func (s *Sites) latestDeployment(ctx context.Context, repoID string) (*Deployment, error) {
	var d Deployment
	var appType string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, branch, commit_sha, status, url, port, app_type, created_at
		FROM deployments WHERE repo_id = ? ORDER BY created_at DESC LIMIT 1`, repoID).
		Scan(&d.ID, &d.RepoID, &d.Branch, &d.CommitSHA, &d.Status, &d.URL, &d.Port, &appType, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	d.AppType = appType
	return &d, nil
}

func (s *Sites) getSite(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var site Site
	var repoName string
	err := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.name, s.git_repo_id, s.production_branch, s.root_path, s.github_full_name,
			COALESCE(g.name, '')
		FROM sites s LEFT JOIN git_repos g ON g.id = s.git_repo_id
		WHERE s.id = ? OR s.git_repo_id = ?`, id, id).
		Scan(&site.ID, &site.Name, &site.GitRepoID, &site.ProductionBranch, &site.RootPath, &site.FullName, &repoName)
	if err != nil {
		err = s.db.QueryRowContext(ctx,
			"SELECT id, name, default_branch FROM git_repos WHERE id = ?", id).
			Scan(&site.GitRepoID, &site.Name, &site.ProductionBranch)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
			return
		}
		site.ID = site.GitRepoID
		site.FullName = site.Name
		site.RootPath = "./"
	} else if site.FullName == "" {
		site.FullName = repoName
		if site.FullName == "" {
			site.FullName = site.Name
		}
	}

	site.LatestDeployment, _ = s.latestDeployment(ctx, site.GitRepoID)
	writeJSON(w, http.StatusOK, site)
}

func (s *Sites) createSite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name              string `json:"name"`
		GitRepoID         string `json:"git_repo_id"`
		ProductionBranch  string `json:"production_branch"`
		RootPath          string `json:"root_path"`
		GitHubFullName    string `json:"github_full_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.GitRepoID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "git_repo_id is required"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if req.ProductionBranch == "" {
		req.ProductionBranch = "main"
	}
	if req.RootPath == "" {
		req.RootPath = "./"
	}

	id, err := generateID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(r.Context(),
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, github_full_name, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		id, req.Name, req.GitRepoID, req.ProductionBranch, req.RootPath, req.GitHubFullName, now)
	if err != nil {
		s.logger.Error("insert site", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	site := Site{
		ID:               id,
		Name:             req.Name,
		FullName:         req.GitHubFullName,
		GitRepoID:        req.GitRepoID,
		ProductionBranch: req.ProductionBranch,
		RootPath:         req.RootPath,
		GitHubConnected:  req.GitHubFullName != "",
	}
	if site.FullName == "" {
		var repoName string
		_ = s.db.QueryRowContext(r.Context(), "SELECT name FROM git_repos WHERE id = ?", req.GitRepoID).Scan(&repoName)
		site.FullName = repoName
	}

	if s.triggerDeploy != nil {
		dep, err := s.triggerDeploy(r.Context(), req.GitRepoID, req.ProductionBranch)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		site.LatestDeployment = dep
	}

	writeJSON(w, http.StatusCreated, site)
}

func (s *Sites) redeploySite(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req struct {
		Branch string `json:"branch"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	var gitRepoID, branch string
	err := s.db.QueryRowContext(ctx,
		"SELECT git_repo_id, production_branch FROM sites WHERE id = ? OR git_repo_id = ?", id, id).
		Scan(&gitRepoID, &branch)
	if err != nil {
		gitRepoID = id
		branch = "main"
		_ = s.db.QueryRowContext(ctx, "SELECT default_branch FROM git_repos WHERE id = ?", id).Scan(&branch)
	}
	if req.Branch != "" {
		branch = req.Branch
	}

	if s.triggerDeploy == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "deploy unavailable"})
		return
	}

	dep, err := s.triggerDeploy(ctx, gitRepoID, branch)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, dep)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

var _ kernel.Component = (*Sites)(nil)
