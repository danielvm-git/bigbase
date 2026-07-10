package github

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"


type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)   {}
func (noopLogger) Warn(msg string, args ...any)   {}
func (noopLogger) Error(msg string, args ...any)  {}
func (noopLogger) Debug(msg string, args ...any)  {}

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type GitHub struct {
	db             DBer
	logger kernel.Logger
	gitDir         string
	appID          string
	appSlug        string
	privateKeyPath string
	webhookSecret  string
	client         *http.Client
	bus            *kernel.EventBus
}

type Options struct {
	DB             DBer
	Logger kernel.Logger
	GitDir         string
	AppID          string
	AppSlug        string
	PrivateKeyPath string
	WebhookSecret  string
}

func New(opts Options) *GitHub {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	dir := opts.GitDir
	if dir == "" {
		dir = "data/git"
	}
	return &GitHub{
		db:             opts.DB,
		logger:         logger,
		gitDir:         dir,
		appID:          opts.AppID,
		appSlug:        opts.AppSlug,
		privateKeyPath: opts.PrivateKeyPath,
		webhookSecret:  opts.WebhookSecret,
		client:         &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *GitHub) Name() string                   { return "github" }
func (g *GitHub) Version() string                { return version }
func (g *GitHub) Dependencies() []string         { return []string{"db"} }
func (g *GitHub) ConfigSchema() json.RawMessage  { return nil }
func (g *GitHub) Hooks() []kernel.HookDef        { return nil }

func (g *GitHub) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (g *GitHub) Start(ctx *kernel.Context) error {
	if err := os.MkdirAll(g.gitDir, 0755); err != nil {
		return fmt.Errorf("create git dir: %w", err)
	}
	if err := g.db.Migrate(`CREATE TABLE IF NOT EXISTS github_installations (
		installation_id INTEGER PRIMARY KEY,
		account_login TEXT NOT NULL,
		account_type TEXT NOT NULL DEFAULT 'User',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate github_installations: %w", err)
	}
	if err := g.db.Migrate(`CREATE TABLE IF NOT EXISTS github_repo_links (
		id TEXT PRIMARY KEY,
		installation_id INTEGER NOT NULL,
		github_repo_id INTEGER NOT NULL,
		full_name TEXT NOT NULL UNIQUE,
		git_repo_id TEXT NOT NULL,
		production_branch TEXT NOT NULL DEFAULT 'main',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate github_repo_links: %w", err)
	}
	g.bus = ctx.Kernel.EventBus()
	g.logger.Info("github component ready", "configured", g.configured())
	return nil
}

func (g *GitHub) Stop(ctx *kernel.Context) error {
	return nil
}

func (g *GitHub) configured() bool {
	return g.appID != "" && g.privateKeyPath != "" && g.appSlug != ""
}

// PublicHandler serves routes called by GitHub without a BigBase session (install
// callback redirect and webhooks). Must not be wrapped in auth middleware.
func (g *GitHub) PublicHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/github/callback", g.handleCallback)
	mux.HandleFunc("/api/github/webhook", g.handleWebhook)
	return mux
}

// ProtectedHandler serves GitHub API routes that require an authenticated admin user.
func (g *GitHub) ProtectedHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/github/status", g.handleStatus)
	mux.HandleFunc("/api/github/install", g.handleInstall)
	mux.HandleFunc("/api/github/repos/connect", g.handleConnect)
	mux.HandleFunc("/api/github/repos", g.handleRepos)
	return mux
}

func (g *GitHub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/github/callback", g.handleCallback)
	mux.HandleFunc("/api/github/webhook", g.handleWebhook)
	mux.HandleFunc("/api/github/status", g.handleStatus)
	mux.HandleFunc("/api/github/install", g.handleInstall)
	mux.HandleFunc("/api/github/repos/connect", g.handleConnect)
	mux.HandleFunc("/api/github/repos", g.handleRepos)
	return mux
}

func (g *GitHub) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	resp := map[string]any{
		"connected":  false,
		"configured": g.configured(),
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var login string
	err := g.db.QueryRowContext(ctx,
		"SELECT account_login FROM github_installations ORDER BY created_at DESC LIMIT 1").
		Scan(&login)
	if err == nil {
		resp["connected"] = true
		resp["login"] = login
	}
	writeJSON(w, http.StatusOK, resp)
}

func (g *GitHub) handleInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !g.configured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "GitHub App not configured; set --github-app-id, --github-app-slug, --github-app-private-key-path",
		})
		return
	}
	state, _ := generateID()
	url := fmt.Sprintf("https://github.com/apps/%s/installations/new?state=%s", g.appSlug, state)
	http.Redirect(w, r, url, http.StatusFound)
}

func (g *GitHub) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	installationID := r.URL.Query().Get("installation_id")
	if installationID == "" {
		http.Redirect(w, r, "/admin/#/deploy/new", http.StatusFound)
		return
	}
	id, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid installation_id"})
		return
	}

	login := "github"
	if g.configured() {
		if details, err := g.fetchInstallation(r.Context(), id); err == nil && details != nil {
			login = details.AccountLogin
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = g.db.ExecContext(r.Context(),
		`INSERT INTO github_installations (installation_id, account_login, account_type, created_at)
		 VALUES (?, ?, 'User', ?)
		 ON CONFLICT(installation_id) DO UPDATE SET account_login=excluded.account_login`,
		id, login, now)

	http.Redirect(w, r, "/admin/#/deploy/new", http.StatusFound)
}

type installationDetails struct {
	AccountLogin string
}

func (g *GitHub) handleRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !g.configured() {
		writeJSON(w, http.StatusOK, map[string]any{"data": []Repo{} })
		return
	}

	instID, err := g.latestInstallationID(r.Context())
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "GitHub App is not installed",
			"code":  "github_not_installed",
		})
		return
	}

	repos, err := g.listInstallationRepos(r.Context(), instID)
	if err != nil {
		g.logger.Error("list github repos", "error", err, "installation_id", instID)
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "failed to list repositories from GitHub",
			"code":    "github_api_error",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": repos})
}

type Repo struct {
	ID            int64  `json:"id"`
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
	Description   string `json:"description,omitempty"`
}

func (g *GitHub) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		GitHubRepoID int64  `json:"github_repo_id"`
		FullName     string `json:"full_name"`
		Branch       string `json:"branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.FullName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "full_name is required"})
		return
	}
	if req.Branch == "" {
		req.Branch = "main"
	}

	gitRepoID, err := g.mirrorRepository(r.Context(), req.FullName, req.Branch)
	if err != nil {
		g.logger.Error("mirror repo", "repo", req.FullName, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to import repository"})
		return
	}

	instID, _ := g.latestInstallationID(r.Context())
	linkID, _ := generateID()
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = g.db.ExecContext(r.Context(),
		`INSERT INTO github_repo_links (id, installation_id, github_repo_id, full_name, git_repo_id, production_branch, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(full_name) DO UPDATE SET git_repo_id=excluded.git_repo_id, production_branch=excluded.production_branch`,
		linkID, instID, req.GitHubRepoID, req.FullName, gitRepoID, req.Branch, now)

	writeJSON(w, http.StatusOK, map[string]string{
		"git_repo_id": gitRepoID,
		"full_name":   req.FullName,
	})
}

func (g *GitHub) mirrorRepository(ctx context.Context, fullName, branch string) (string, error) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid full_name")
	}
	repoName := strings.ReplaceAll(fullName, "/", "-")
	if strings.Contains(repoName, "..") {
		return "", fmt.Errorf("invalid repo name")
	}

	var existingID string
	err := g.db.QueryRowContext(ctx, "SELECT git_repo_id FROM github_repo_links WHERE full_name = ?", fullName).Scan(&existingID)
	if err == nil && existingID != "" {
		_ = g.fetchMirror(ctx, existingID, fullName)
		return existingID, nil
	}

	id, err := generateID()
	if err != nil {
		return "", err
	}

	repoDir := filepath.Join(g.gitDir, id+".git")
	if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
		return "", err
	}

	cloneURL := fmt.Sprintf("https://github.com/%s.git", fullName)
	if g.configured() {
		if instID, err := g.latestInstallationID(ctx); err == nil {
			if token, err := g.installationToken(ctx, instID); err == nil {
				cloneURL = fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, fullName)
			}
		}
	}

	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		cmd := exec.CommandContext(ctx, "git", "clone", "--bare", cloneURL, repoDir)
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			_ = os.RemoveAll(repoDir)
			return "", fmt.Errorf("git clone --bare: %w", err)
		}
	} else {
		_ = g.fetchMirror(ctx, id, fullName)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = g.db.ExecContext(ctx,
		`INSERT INTO git_repos (id, name, owner_id, private, default_branch, description, created_at)
		 VALUES (?, ?, 0, 0, ?, '', ?)
		 ON CONFLICT(name) DO UPDATE SET default_branch=excluded.default_branch`,
		id, repoName, branch, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			_ = g.db.QueryRowContext(ctx, "SELECT id FROM git_repos WHERE name = ?", repoName).Scan(&id)
			return id, nil
		}
		return "", err
	}

	return id, nil
}

func (g *GitHub) fetchMirror(ctx context.Context, gitRepoID, fullName string) error {
	repoDir := filepath.Join(g.gitDir, gitRepoID+".git")
	if _, err := os.Stat(repoDir); err != nil {
		return err
	}
	remote := fmt.Sprintf("https://github.com/%s.git", fullName)
	if g.configured() {
		if instID, err := g.latestInstallationID(ctx); err == nil {
			if token, err := g.installationToken(ctx, instID); err == nil {
				remote = fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, fullName)
			}
		}
	}
	cmd := exec.CommandContext(ctx, "git", "fetch", "--prune", remote, "+refs/heads/*:refs/heads/*")
	cmd.Dir = repoDir
	return cmd.Run()
}

func (g *GitHub) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 5<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body"})
		return
	}

	if g.webhookSecret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if !verifyWebhookSignature(g.webhookSecret, body, sig) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid signature"})
			return
		}
	}

	var payload struct {
		Ref        string `json:"ref"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event != "push" {
		w.WriteHeader(http.StatusOK)
		return
	}

	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	var gitRepoID, productionBranch string
	err = g.db.QueryRowContext(r.Context(),
		"SELECT git_repo_id, production_branch FROM github_repo_links WHERE full_name = ?",
		payload.Repository.FullName).Scan(&gitRepoID, &productionBranch)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	if branch != productionBranch {
		w.WriteHeader(http.StatusOK)
		return
	}

	_ = g.fetchMirror(r.Context(), gitRepoID, payload.Repository.FullName)

	w.WriteHeader(http.StatusOK)
}

// VerifyWebhookSignatureForTest exposes signature verification for tests.
func VerifyWebhookSignatureForTest(secret string, body []byte, signature string) bool {
	return verifyWebhookSignature(secret, body, signature)
}

func verifyWebhookSignature(secret string, body []byte, signature string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	sigHex := strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	got, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, got)
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (g *GitHub) latestInstallationID(ctx context.Context) (int64, error) {
	var id int64
	err := g.db.QueryRowContext(ctx,
		"SELECT installation_id FROM github_installations ORDER BY created_at DESC LIMIT 1").Scan(&id)
	return id, err
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

var _ kernel.Component = (*GitHub)(nil)
