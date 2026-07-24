package sites

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

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

type AuthPolicy struct {
	Default        string   `json:"default"`
	ProtectedPaths []string `json:"protected_paths"`
	PublicPaths    []string `json:"public_paths"`
	Accept         []string `json:"accept"`
}

type Site struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	FullName         string      `json:"full_name"`
	GitRepoID        string      `json:"git_repo_id"`
	ProductionBranch string      `json:"production_branch"`
	RootPath         string      `json:"root_path"`
	GitHubConnected  bool        `json:"github_connected,omitempty"`
	LatestDeployment *Deployment `json:"latest_deployment,omitempty"`
	AuthPolicy       *AuthPolicy `json:"auth_policy,omitempty"`
}

type DeployTrigger func(ctx context.Context, repoID, branch, siteName, siteID string, passthroughPaths []string, appType string) (*Deployment, error)
type DeleteSiteCleanupFunc func(ctx context.Context, siteID, repoID string) error

// CertInfoFunc returns the expiry of the cached TLS certificate for a domain.
// ok is false when no certificate is cached yet (provisioning) or HTTPS is off.
type CertInfoFunc func(ctx context.Context, domain string) (notAfter time.Time, ok bool)

// ActivateDomainFunc registers a freshly verified domain with the proxy so it
// routes immediately, without waiting for the next deployment.
type ActivateDomainFunc func(ctx context.Context, siteID, domain string) error

type Sites struct {
	db                DBer
	logger            kernel.Logger
	gitDir            string
	triggerDeploy     DeployTrigger
	deleteSiteCleanup DeleteSiteCleanupFunc
	encryptionKey     []byte
	certInfo          CertInfoFunc
	unregisterHost    func(domain string)
	activateDomain    ActivateDomainFunc
	updateAuthPolicy  func(siteID string, policyJSON string)
	verifyLim         *verifyLimiter
	validateManifest  func([]byte) error
}

var _ kernel.Component = (*Sites)(nil)

type Options struct {
	DB                DBer
	Logger            kernel.Logger
	GitDir            string
	TriggerDeploy     DeployTrigger
	DeleteSiteCleanup DeleteSiteCleanupFunc
	EnvEncryptionKey  string
	// CertInfo resolves live TLS certificate status for verified domains.
	CertInfo CertInfoFunc
	// UnregisterHost removes a proxy host route when a domain is deleted.
	UnregisterHost func(domain string)
	// ActivateDomain registers a domain with the proxy on successful verification.
	ActivateDomain ActivateDomainFunc
	// UpdateAuthPolicy notifies the proxy when a site's auth policy changes.
	UpdateAuthPolicy func(siteID string, policyJSON string)
	// ValidateManifest validates bigbase.yaml content (injected from deploy; ECC seam).
	ValidateManifest func([]byte) error
}

func New(opts Options) *Sites {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	gitDir := opts.GitDir
	if gitDir == "" {
		gitDir = "data/git"
	}
	key, keyErr := parseEncryptionKey(opts.EnvEncryptionKey)
	s := &Sites{
		db:                opts.DB,
		logger:            logger,
		gitDir:            gitDir,
		triggerDeploy:     opts.TriggerDeploy,
		deleteSiteCleanup: opts.DeleteSiteCleanup,
		encryptionKey:     key,
		certInfo:          opts.CertInfo,
		unregisterHost:    opts.UnregisterHost,
		activateDomain:    opts.ActivateDomain,
		updateAuthPolicy:  opts.UpdateAuthPolicy,
		verifyLim:         newVerifyLimiter(5 * time.Second),
		validateManifest:  opts.ValidateManifest,
	}
	if keyErr != nil {
		// Log immediately — encryption is silently disabled when the key is malformed.
		logger.Warn("env encryption key invalid — env vars will be stored without encryption", "error", keyErr)
	}
	return s
}

func (s *Sites) Name() string                  { return "sites" }
func (s *Sites) DB() DBer                      { return s.db }
func (s *Sites) Version() string               { return version }
func (s *Sites) Dependencies() []string        { return []string{"db"} }

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
	_ = s.db.Migrate(`ALTER TABLE sites ADD COLUMN auth_policy TEXT NOT NULL DEFAULT '{}'`)
	_ = s.db.Migrate(`ALTER TABLE sites ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0`)
	// Reassign orphaned sites (org_id=0 from the DEFAULT) to the admin's
	// default_org_id so they become visible under org-scoped queries.
	if _, err := s.db.Exec(`
		UPDATE sites SET org_id = (
			SELECT default_org_id FROM users WHERE role = 'admin' AND default_org_id > 0 LIMIT 1
		) WHERE org_id = 0 AND EXISTS (
			SELECT 1 FROM users WHERE role = 'admin' AND default_org_id > 0
		)`); err != nil {
		s.logger.Warn("reassign orphaned sites to admin org", "error", err)
	}
	if err := s.migrateDomains(); err != nil {
		return fmt.Errorf("migrate site_domains: %w", err)
	}
	if err := s.migrateEnvVars(); err != nil {
		return fmt.Errorf("migrate site_env_vars: %w", err)
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

func (s *Sites) handleSites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listSites(w, r)
	case http.MethodPost:
		s.createSite(w, r)
	default:
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Sites) handleSiteByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}
	id := parts[0]

	if len(parts) == 2 && parts[1] == "deploy" && r.Method == http.MethodPost {
		s.redeploySite(w, r, id)
		return
	}

	if len(parts) == 2 && parts[1] == "logs" && r.Method == http.MethodGet {
		s.listSiteRequestLogs(w, r, id)
		return
	}

	if len(parts) == 2 && parts[1] == "manifest" {
		switch r.Method {
		case http.MethodGet:
			s.getSiteManifest(w, r, id)
			return
		case http.MethodPost:
			s.saveSiteManifest(w, r, id)
			return
		default:
			kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
	}

	if len(parts) == 2 && parts[1] == "auth-policy" {
		switch r.Method {
		case http.MethodGet:
			s.getSiteAuthPolicy(w, r, id)
			return
		case http.MethodPost:
			s.setSiteAuthPolicy(w, r, id)
			return
		default:
			kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
	}

	// Route domain sub-paths: /api/sites/{id}/domains[/{domain}/verify]
	if len(parts) >= 2 && parts[1] == "domains" {
		s.handleDomains(w, r)
		return
	}

	if len(parts) >= 2 && parts[1] == "env-vars" {
		s.handleEnvVarsRoute(w, r, id, parts)
		return
	}

	if len(parts) == 1 && r.Method == http.MethodDelete {
		s.deleteSite(w, r, id)
		return
	}

	if r.Method != http.MethodGet {
		kernel.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	s.getSite(w, r, id)
}

func (s *Sites) listSites(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Extract org_id for multi-tenant isolation.
	orgID, ok := auth.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "organization required"})
		return
	}

	sites, err := s.ListSites(ctx)
	if err != nil {
		s.logger.Error("list sites", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, map[string]any{"data": sites})
}

func (s *Sites) ListSites(ctx context.Context) ([]Site, error) {
	// Extract org_id for multi-tenant isolation.
	orgID, _ := auth.OrgIDFromContext(ctx)

	var rows *sql.Rows
	var err error
	if orgID > 0 {
		// Filter by org_id when available (JWT/API key auth). Legacy sites
		// created before org_id scoping existed default to org_id=0 and
		// must remain visible — org_id=0 is never a real caller's org_id,
		// so it can never collide with an actual cross-tenant site.
		rows, err = s.db.QueryContext(ctx, `
			SELECT s.id, s.name, s.git_repo_id, s.production_branch, s.root_path,
				COALESCE(NULLIF(s.github_full_name, ''), g.name, s.name),
				s.auth_policy
			FROM sites s
			LEFT JOIN git_repos g ON g.id = s.git_repo_id
			WHERE s.org_id = ? OR s.org_id = 0
			ORDER BY s.name`, orgID)
	} else {
		// Fallback for unauthenticated contexts (tests, site key auth).
		rows, err = s.db.QueryContext(ctx, `
			SELECT s.id, s.name, s.git_repo_id, s.production_branch, s.root_path,
				COALESCE(NULLIF(s.github_full_name, ''), g.name, s.name),
				s.auth_policy
			FROM sites s
			LEFT JOIN git_repos g ON g.id = s.git_repo_id
			ORDER BY s.name`)
	}
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	defer func() { _ = rows.Close() }()

	sites := make([]Site, 0)
	for rows.Next() {
		var site Site
		var authPolicyStr string
		if err := rows.Scan(&site.ID, &site.Name, &site.GitRepoID, &site.ProductionBranch,
			&site.RootPath, &site.FullName, &authPolicyStr); err != nil {
			continue
		}
		if site.FullName != "" {
			site.GitHubConnected = strings.Contains(site.FullName, "/")
		}
		if authPolicyStr != "" {
			_ = json.Unmarshal([]byte(authPolicyStr), &site.AuthPolicy)
		}
		sites = append(sites, site)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sites iteration: %w", err)
	}

	latestByRepo := s.latestDeploymentsByRepo(ctx)
	for i := range sites {
		sites[i].LatestDeployment = latestByRepo[sites[i].GitRepoID]
	}

	if len(sites) == 0 && orgID == 0 {
		sites = s.listSitesFromRepos(ctx)
	}
	return sites, nil
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
		out = append(out, Site{
			ID:               id,
			Name:             name,
			FullName:         name,
			GitRepoID:        id,
			ProductionBranch: branch,
			RootPath:         "./",
		})
	}
	latestByRepo := s.latestDeploymentsByRepo(ctx)
	for i := range out {
		out[i].LatestDeployment = latestByRepo[out[i].GitRepoID]
	}
	return out
}

func (s *Sites) latestDeploymentsByRepo(ctx context.Context) map[string]*Deployment {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.repo_id, d.branch, d.commit_sha, d.status, d.url, d.port, d.app_type, d.created_at
		FROM deployments d
		INNER JOIN (
			SELECT repo_id, MAX(created_at) AS latest_at
			FROM deployments
			GROUP BY repo_id
		) latest ON d.repo_id = latest.repo_id AND d.created_at = latest.latest_at`)
	if err != nil {
		return map[string]*Deployment{}
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]*Deployment)
	for rows.Next() {
		var d Deployment
		var appType string
		if err := rows.Scan(&d.ID, &d.RepoID, &d.Branch, &d.CommitSHA, &d.Status, &d.URL, &d.Port, &appType, &d.CreatedAt); err != nil {
			continue
		}
		d.AppType = appType
		out[d.RepoID] = &d
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

// requireSiteOwnership verifies that the caller's org owns the given site.
// Returns true if the check passes, or writes a 404 error and returns false.
// This prevents IDOR attacks by ensuring cross-org access is denied.
// The siteID parameter can be either a site id or a git_repo_id (legacy alias).
func (s *Sites) requireSiteOwnership(ctx context.Context, w http.ResponseWriter, siteID string) bool {
	orgID, ok := auth.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		kernel.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "organization required"})
		return false
	}

	var dbOrgID int64
	err := s.db.QueryRowContext(ctx, "SELECT org_id FROM sites WHERE id = ? OR git_repo_id = ?", siteID, siteID).Scan(&dbOrgID)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
		return false
	}

	if dbOrgID != orgID && dbOrgID != 0 {
		// Return 404 (not 403) to avoid leaking site existence across orgs.
		// dbOrgID == 0 is a legacy site predating org_id scoping — never a
		// real org, so it's accessible to any authenticated org rather than
		// permanently orphaned.
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
		return false
	}
	return true
}

func (s *Sites) getSite(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !s.requireSiteOwnership(ctx, w, id) {
		return
	}

	site, err := s.GetSite(ctx, id)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
		return
	}
	kernel.WriteJSON(w, http.StatusOK, site)
}

func (s *Sites) GetSite(ctx context.Context, id string) (*Site, error) {
	var site Site
	var repoName string
	var authPolicyStr string
	err := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.name, s.git_repo_id, s.production_branch, s.root_path, s.github_full_name,
			COALESCE(g.name, ''), s.auth_policy
		FROM sites s LEFT JOIN git_repos g ON g.id = s.git_repo_id
		WHERE s.id = ? OR s.git_repo_id = ?`, id, id).
		Scan(&site.ID, &site.Name, &site.GitRepoID, &site.ProductionBranch, &site.RootPath, &site.FullName, &repoName, &authPolicyStr)
	if err != nil {
		err = s.db.QueryRowContext(ctx,
			"SELECT id, name, default_branch FROM git_repos WHERE id = ?", id).
			Scan(&site.GitRepoID, &site.Name, &site.ProductionBranch)
		if err != nil {
			return nil, fmt.Errorf("site %q not found", id)
		}
		site.ID = site.GitRepoID
		site.FullName = site.Name
		site.RootPath = "./"
	} else {
		if authPolicyStr != "" {
			_ = json.Unmarshal([]byte(authPolicyStr), &site.AuthPolicy)
		}
		if site.FullName == "" {
			site.FullName = repoName
			if site.FullName == "" {
				site.FullName = site.Name
			}
		}
	}

	site.LatestDeployment, _ = s.latestDeployment(ctx, site.GitRepoID)
	return &site, nil
}

func (s *Sites) createSite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name             string `json:"name"`
		GitRepoID        string `json:"git_repo_id"`
		ProductionBranch string `json:"production_branch"`
		RootPath         string `json:"root_path"`
		GitHubFullName   string `json:"github_full_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.GitRepoID == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "git_repo_id is required"})
		return
	}
	if req.Name == "" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if req.ProductionBranch == "" {
		req.ProductionBranch = "main"
	}
	if req.RootPath == "" {
		req.RootPath = "./"
	}

	id, name, err := s.insertSite(r.Context(), req.GitRepoID, req.Name, req.ProductionBranch, req.RootPath, req.GitHubFullName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		s.logger.Error("create site", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	site := Site{
		ID:               id,
		Name:             name,
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
		dep, err := s.triggerDeploy(r.Context(), req.GitRepoID, req.ProductionBranch, name, id, nil, "")
		if err != nil {
			kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		site.LatestDeployment = dep
	}

	kernel.WriteJSON(w, http.StatusCreated, site)
}

// CreateSite provisions a site for a git repository (MCP SiteCreator).
// When name is empty it defaults from the git_repos name.
func (s *Sites) CreateSite(ctx context.Context, gitRepoID, name, branch string) (id, siteName string, err error) {
	return s.insertSite(ctx, gitRepoID, name, branch, "", "")
}

// insertSite writes a site row; rootPath defaults to "./" when empty.
func (s *Sites) insertSite(ctx context.Context, gitRepoID, name, branch, rootPath, githubFullName string) (id, siteName string, err error) {
	if gitRepoID == "" {
		return "", "", fmt.Errorf("git_repo_id is required")
	}
	if branch == "" {
		branch = "main"
	}
	if rootPath == "" {
		rootPath = "./"
	}

	var repoName string
	if err := s.db.QueryRowContext(ctx, "SELECT name FROM git_repos WHERE id = ?", gitRepoID).Scan(&repoName); err != nil {
		return "", "", fmt.Errorf("repository %q not found", gitRepoID)
	}
	if name == "" {
		name = repoName
	}

	id, err = kernel.GenerateID()
	if err != nil {
		return "", "", fmt.Errorf("generate id: %w", err)
	}

	// Extract org_id from context for multi-tenant isolation.
	var orgID int64
	if oid, ok := auth.OrgIDFromContext(ctx); ok {
		orgID = oid
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sites (id, name, git_repo_id, production_branch, root_path, github_full_name, created_at, org_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		id, name, gitRepoID, branch, rootPath, githubFullName, now, orgID)
	if err != nil {
		return "", "", fmt.Errorf("insert site: %w", err)
	}

	return id, name, nil
}

type siteDeleteTarget struct {
	ID        string
	GitRepoID string
}

func (s *Sites) deleteSite(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if !s.requireSiteOwnership(ctx, w, id) {
		return
	}

	target, err := s.siteDeleteTarget(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
			return
		}
		s.logger.Error("resolve site for delete", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	active, err := s.hasActiveDeployment(ctx, target)
	if err != nil {
		s.logger.Error("check active deployments for site delete", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if active && s.deleteSiteCleanup == nil {
		kernel.WriteJSON(w, http.StatusConflict, map[string]string{"error": "site has an active deployment — wait for it to finish or delete deployments first"})
		return
	}

	if s.deleteSiteCleanup != nil {
		if err := s.deleteSiteCleanup(ctx, target.ID, target.GitRepoID); err != nil {
			s.logger.Error("delete site deploy cleanup", "error", err, "site_id", target.ID, "repo_id", target.GitRepoID)
		}
	}

	if err := s.deleteSiteRecords(ctx, target); err != nil {
		s.logger.Error("delete site records", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Sites) siteDeleteTarget(ctx context.Context, id string) (siteDeleteTarget, error) {
	var target siteDeleteTarget
	err := s.db.QueryRowContext(ctx,
		"SELECT id, git_repo_id FROM sites WHERE id = ? OR git_repo_id = ?", id, id).
		Scan(&target.ID, &target.GitRepoID)
	return target, err
}

func (s *Sites) hasActiveDeployment(ctx context.Context, target siteDeleteTarget) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM deployments
		 WHERE (site_id = ? OR (site_id = '' AND repo_id = ?))
		 AND status IN ('pending', 'building', 'running')`,
		target.ID, target.GitRepoID).Scan(&count)
	if isMissingOptionalTable(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Sites) deleteSiteRecords(ctx context.Context, target siteDeleteTarget) error {
	queries := []struct {
		query string
		args  []any
	}{
		{query: "DELETE FROM site_domains WHERE site_id = ?", args: []any{target.ID}},
		{query: "DELETE FROM site_request_logs WHERE site_id = ?", args: []any{target.ID}},
		{query: "DELETE FROM deployments WHERE site_id = ? OR (site_id = '' AND repo_id = ?)", args: []any{target.ID, target.GitRepoID}},
		{query: "DELETE FROM sites WHERE id = ?", args: []any{target.ID}},
	}

	for _, q := range queries {
		if _, err := s.db.ExecContext(ctx, q.query, q.args...); err != nil && !isMissingOptionalTable(err) {
			return err
		}
	}
	return nil
}

func isMissingOptionalTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such table: deployments") ||
		strings.Contains(msg, "no such table: site_request_logs") ||
		strings.Contains(msg, `relation "deployments" does not exist`) ||
		strings.Contains(msg, `relation "site_request_logs" does not exist`) ||
		strings.Contains(msg, "no such column") ||
		strings.Contains(msg, "column") && strings.Contains(msg, "does not exist")
}

func (s *Sites) listSiteRequestLogs(w http.ResponseWriter, r *http.Request, siteID string) {
	if !s.requireSiteOwnership(r.Context(), w, siteID) {
		return
	}

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 500 {
		limit = 100
	}
	statusClass := q.Get("status_class") // 2xx, 4xx, 5xx
	pathPrefix := q.Get("path")

	where := "site_id = ?"
	args := []any{siteID}

	if statusClass != "" {
		switch statusClass {
		case "2xx":
			where += " AND status >= 200 AND status < 300"
		case "4xx":
			where += " AND status >= 400 AND status < 500"
		case "5xx":
			where += " AND status >= 500 AND status < 600"
		}
	}
	if pathPrefix != "" {
		where += " AND path LIKE ?"
		args = append(args, pathPrefix+"%")
	}

	var total int
	_ = s.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM site_request_logs WHERE "+where, args...).Scan(&total)

	rows, err := s.db.QueryContext(r.Context(),
		"SELECT id, method, path, status, duration_ms, created_at FROM site_request_logs WHERE "+where+" ORDER BY created_at DESC LIMIT ?",
		append(args, limit)...)
	if err != nil {
		s.logger.Error("list request logs", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	defer func() { _ = rows.Close() }()

	type LogEntry struct {
		ID         string `json:"id"`
		Method     string `json:"method"`
		Path       string `json:"path"`
		Status     int    `json:"status"`
		DurationMS int    `json:"duration_ms"`
		CreatedAt  string `json:"created_at"`
	}

	data := make([]LogEntry, 0)
	for rows.Next() {
		var l LogEntry
		if err := rows.Scan(&l.ID, &l.Method, &l.Path, &l.Status, &l.DurationMS, &l.CreatedAt); err != nil {
			continue
		}
		data = append(data, l)
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{
		"data":    data,
		"total":   total,
		"site_id": siteID,
	})
}

func (s *Sites) redeploySite(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !s.requireSiteOwnership(ctx, w, id) {
		return
	}

	var req struct {
		Branch  string `json:"branch"`
		AppType string `json:"app_type"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	var gitRepoID, branch, siteName string
	err := s.db.QueryRowContext(ctx,
		"SELECT git_repo_id, production_branch, name FROM sites WHERE id = ? OR git_repo_id = ?", id, id).
		Scan(&gitRepoID, &branch, &siteName)
	if err != nil {
		gitRepoID = id
		branch = "main"
		_ = s.db.QueryRowContext(ctx, "SELECT name, default_branch FROM git_repos WHERE id = ?", id).Scan(&siteName, &branch)
	}
	if req.Branch != "" {
		branch = req.Branch
	}

	if s.triggerDeploy == nil {
		kernel.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "deploy unavailable"})
		return
	}

	dep, err := s.triggerDeploy(ctx, gitRepoID, branch, siteName, id, nil, req.AppType)
	if err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	kernel.WriteJSON(w, http.StatusCreated, dep)
}

func (s *Sites) getSiteManifest(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if !s.requireSiteOwnership(ctx, w, id) {
		return
	}

	repoID, branch, err := s.resolveRepoBranch(ctx, id)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "site or repository not found"})
		return
	}

	repoPath := filepath.Join(s.gitDir, repoID+".git")
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "git repository directory not found"})
		return
	}

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "show", branch+":bigbase.yaml")
	out, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := string(out)
		if strings.Contains(errMsg, "exists but is not") || strings.Contains(errMsg, "does not exist") || strings.Contains(errMsg, "Invalid object name") || strings.Contains(errMsg, "fatal: path") {
			kernel.WriteJSON(w, http.StatusOK, map[string]any{
				"exists":  false,
				"content": "",
			})
			return
		}
		s.logger.Error("git show failed", "error", err, "output", errMsg)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read manifest file"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{
		"exists":  true,
		"content": string(out),
	})
}

func (s *Sites) saveSiteManifest(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if s.validateManifest != nil {
		if err := s.validateManifest([]byte(req.Content)); err != nil {
			kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid manifest: %v", err)})
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if !s.requireSiteOwnership(ctx, w, id) {
		return
	}

	repoID, branch, err := s.resolveRepoBranch(ctx, id)
	if err != nil {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "site or repository not found"})
		return
	}

	repoPath := filepath.Join(s.gitDir, repoID+".git")
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "git repository directory not found"})
		return
	}

	if err := s.commitManifestToRepo(ctx, repoPath, branch, req.Content); err != nil {
		s.logger.Error("failed to commit manifest", "error", err)
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save manifest"})
		return
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// commitManifestToRepo clones the repository, writes bigbase.yaml,
// and commits/pushes the change to the given branch.
func (s *Sites) commitManifestToRepo(ctx context.Context, repoPath, branch, content string) error {
	tempDir, err := os.MkdirTemp("", "bigbase-manifest-edit-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cloneCmd := exec.CommandContext(ctx, "git", "clone", repoPath, ".")
	cloneCmd.Dir = tempDir
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clone: %w (output: %s)", err, string(out))
	}

	// Checkout branch — create it if it doesn't exist yet
	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", branch)
	checkoutCmd.Dir = tempDir
	if out, err := checkoutCmd.CombinedOutput(); err != nil {
		checkoutNewCmd := exec.CommandContext(ctx, "git", "checkout", "-b", branch)
		checkoutNewCmd.Dir = tempDir
		if out2, err2 := checkoutNewCmd.CombinedOutput(); err2 != nil {
			return fmt.Errorf("checkout: %w (out: %s), create-branch: %v (out: %s)", err, string(out), err2, string(out2))
		}
	}

	manifestFile := filepath.Join(tempDir, "bigbase.yaml")
	if err := os.WriteFile(manifestFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	configUserCmd := exec.CommandContext(ctx, "git", "config", "user.name", "BigBase Admin")
	configUserCmd.Dir = tempDir
	_ = configUserCmd.Run()
	configEmailCmd := exec.CommandContext(ctx, "git", "config", "user.email", "admin@bigbase.local")
	configEmailCmd.Dir = tempDir
	_ = configEmailCmd.Run()

	addCmd := exec.CommandContext(ctx, "git", "add", "bigbase.yaml")
	addCmd.Dir = tempDir
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("add: %w (output: %s)", err, string(out))
	}

	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	statusCmd.Dir = tempDir
	statusOut, _ := statusCmd.Output()
	if len(bytes.TrimSpace(statusOut)) == 0 {
		return nil // no changes to commit
	}

	commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", "Update bigbase.yaml")
	commitCmd.Dir = tempDir
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("commit: %w (output: %s)", err, string(out))
	}

	pushCmd := exec.CommandContext(ctx, "git", "push", "origin", branch)
	pushCmd.Dir = tempDir
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("push: %w (output: %s)", err, string(out))
	}

	return nil
}

// resolveRepoBranch looks up a site's git_repo_id and production_branch.
// Falls back to querying git_repos directly if the sites lookup fails.
func (s *Sites) resolveRepoBranch(ctx context.Context, id string) (repoID, branch string, err error) {
	err = s.db.QueryRowContext(ctx,
		"SELECT git_repo_id, production_branch FROM sites WHERE id = ?", id).
		Scan(&repoID, &branch)
	if err != nil {
		// Fallback: check if id is a git_repos id
		err = s.db.QueryRowContext(ctx,
			"SELECT id, default_branch FROM git_repos WHERE id = ?", id).
			Scan(&repoID, &branch)
		if err != nil {
			return "", "", err
		}
	}
	if branch == "" {
		branch = "main"
	}
	return repoID, branch, nil
}

func (s *Sites) getSiteAuthPolicy(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !s.requireSiteOwnership(ctx, w, id) {
		return
	}

	var policyStr sql.NullString
	err := s.db.QueryRowContext(ctx, "SELECT auth_policy FROM sites WHERE id = ?", id).Scan(&policyStr)
	if err != nil {
		if err == sql.ErrNoRows {
			kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
		} else {
			kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return
	}

	var policy AuthPolicy
	if policyStr.Valid && policyStr.String != "" {
		_ = json.Unmarshal([]byte(policyStr.String), &policy)
	}
	if policy.Default == "" {
		policy.Default = "public"
	}
	kernel.WriteJSON(w, http.StatusOK, policy)
}

func (s *Sites) setSiteAuthPolicy(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !s.requireSiteOwnership(ctx, w, id) {
		return
	}

	// Verify site exists first
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT count(1) FROM sites WHERE id = ?", id).Scan(&exists)
	if err != nil || exists == 0 {
		kernel.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
		return
	}

	var policy AuthPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Basic validation / defaults
	if policy.Default == "" {
		policy.Default = "public"
	}
	if policy.Default != "public" && policy.Default != "protected" {
		kernel.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid default policy value"})
		return
	}

	policyBytes, err := json.Marshal(policy)
	if err != nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal policy failed"})
		return
	}
	policyStr := string(policyBytes)

	_, err = s.db.ExecContext(ctx, "UPDATE sites SET auth_policy = ? WHERE id = ?", policyStr, id)
	if err != nil {
		kernel.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Trigger callback to update proxy immediately
	if s.updateAuthPolicy != nil {
		s.updateAuthPolicy(id, policyStr)
	}

	kernel.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "policy": policy})
}
