package sites

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/kernel"
)

type ddTestLogger struct{}

func (ddTestLogger) Info(msg string, args ...any)  {}
func (ddTestLogger) Warn(msg string, args ...any)  {}
func (ddTestLogger) Error(msg string, args ...any) {}
func (ddTestLogger) Debug(msg string, args ...any) {}

func setupDeployDefaultsTest(t *testing.T) (*Sites, func()) {
	t.Helper()
	log := ddTestLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: log})
	g := git.New(git.Options{DB: d, Logger: log})
	s := New(Options{DB: d, Logger: log})

	k := kernel.New(log)
	k.Register(d)
	k.Register(g)
	k.Register(s)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	return s, func() { _ = k.Stop() }
}

func createTestSite(t *testing.T, s *Sites, name string) string {
	t.Helper()
	ctx := auth.WithOrgID(t.Context(), 1)
	// Create a git repo record first
	repoID, _ := kernel.GenerateID()
	_, err := s.db.ExecContext(ctx, "INSERT INTO git_repos (id, name, owner_id, default_branch) VALUES (?, ?, 1, 'main')", repoID, name+"-repo")
	if err != nil {
		t.Fatalf("create git repo: %v", err)
	}
	id, _, err := s.insertSite(ctx, repoID, name, "main", "./", "", "")
	if err != nil {
		t.Fatalf("create test site: %v", err)
	}
	return id
}

func TestDeployDefaultsCRUD(t *testing.T) {
	s, cleanup := setupDeployDefaultsTest(t)
	defer cleanup()

	siteID := createTestSite(t, s, "deploy-defaults-test")

	h := s.Handler()

	// GET deploy-defaults — empty initially
	req := httptest.NewRequest("GET", "/api/sites/"+siteID+"/deploy-defaults", nil)
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.WithOrgID(req.Context(), 1)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("GET deploy-defaults: status %d, want 200", w.Code)
	}

	var dd DeployDefaults
	if err := json.Unmarshal(w.Body.Bytes(), &dd); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// Empty defaults should have zero values
	if dd.AppType != "" {
		t.Errorf("expected empty app_type, got %q", dd.AppType)
	}

	// PUT deploy-defaults
	payload := DeployDefaults{
		AppType:          "node",
		BuildCommand:     "npm run build",
		StartCommand:     "npm start",
		PassthroughPaths: []string{"/mcp"},
		HealthPath:       "/api/health",
	}
	body, _ := json.Marshal(payload)
	req = httptest.NewRequest("PUT", "/api/sites/"+siteID+"/deploy-defaults", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx = auth.WithOrgID(req.Context(), 1)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("PUT deploy-defaults: status %d, want 200\nbody: %s", w.Code, w.Body.String())
	}

	// GET deploy-defaults — should return what we set
	req = httptest.NewRequest("GET", "/api/sites/"+siteID+"/deploy-defaults", nil)
	req.Header.Set("Content-Type", "application/json")
	ctx = auth.WithOrgID(req.Context(), 1)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("GET deploy-defaults after PUT: status %d, want 200", w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &dd); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if dd.AppType != "node" {
		t.Errorf("app_type: got %q, want %q", dd.AppType, "node")
	}
	if dd.BuildCommand != "npm run build" {
		t.Errorf("build_command: got %q, want %q", dd.BuildCommand, "npm run build")
	}
	if dd.StartCommand != "npm start" {
		t.Errorf("start_command: got %q, want %q", dd.StartCommand, "npm start")
	}
	if len(dd.PassthroughPaths) != 1 || dd.PassthroughPaths[0] != "/mcp" {
		t.Errorf("passthrough_paths: got %v, want [/mcp]", dd.PassthroughPaths)
	}
	if dd.HealthPath != "/api/health" {
		t.Errorf("health_path: got %q, want %q", dd.HealthPath, "/api/health")
	}

	// GET site — should include deploy_defaults
	req = httptest.NewRequest("GET", "/api/sites/"+siteID, nil)
	req.Header.Set("Content-Type", "application/json")
	ctx = auth.WithOrgID(req.Context(), 1)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("GET site: status %d, want 200", w.Code)
	}

	var site Site
	if err := json.Unmarshal(w.Body.Bytes(), &site); err != nil {
		t.Fatalf("decode site: %v", err)
	}
	if site.DeployDefaults == nil {
		t.Fatal("site.deploy_defaults is nil")
	}
	if site.DeployDefaults.AppType != "node" {
		t.Errorf("site.deploy_defaults.app_type: got %q, want %q", site.DeployDefaults.AppType, "node")
	}
}

func TestDeployDefaultsValidation(t *testing.T) {
	s, cleanup := setupDeployDefaultsTest(t)
	defer cleanup()

	siteID := createTestSite(t, s, "deploy-defaults-validation-test")
	h := s.Handler()

	// Invalid app_type
	payload := DeployDefaults{AppType: "rust"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", "/api/sites/"+siteID+"/deploy-defaults", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.WithOrgID(req.Context(), 1)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid app_type: status %d, want 400", w.Code)
	}

	// Invalid health_path (no leading /)
	payload = DeployDefaults{HealthPath: "health"}
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest("PUT", "/api/sites/"+siteID+"/deploy-defaults", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx = auth.WithOrgID(req.Context(), 1)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid health_path: status %d, want 400", w.Code)
	}

	// Valid: empty defaults (all zero values)
	payload = DeployDefaults{}
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest("PUT", "/api/sites/"+siteID+"/deploy-defaults", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx = auth.WithOrgID(req.Context(), 1)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("empty defaults: status %d, want 200", w.Code)
	}
}

func TestDeployDefaultsCreateSiteWithDefaults(t *testing.T) {
	s, cleanup := setupDeployDefaultsTest(t)
	defer cleanup()

	h := s.Handler()

	// Create site with deploy_defaults via POST /api/sites
	// First we need a git repo
	ctx := auth.WithOrgID(t.Context(), 1)
	repoID, err := kernel.GenerateID()
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO git_repos (id, name, owner_id, default_branch) VALUES (?, ?, 1, ?)", repoID, "test-repo", "main")
	if err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"name":              "test-with-defaults",
		"git_repo_id":       repoID,
		"production_branch": "main",
		"deploy_defaults": DeployDefaults{
			AppType:      "python",
			BuildCommand: "pip install -r requirements.txt",
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/sites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx = auth.WithOrgID(req.Context(), 1)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/sites: status %d, want 201\nbody: %s", w.Code, w.Body.String())
	}

	var site Site
	if err := json.Unmarshal(w.Body.Bytes(), &site); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if site.DeployDefaults == nil {
		t.Fatal("created site.deploy_defaults is nil")
	}
	if site.DeployDefaults.AppType != "python" {
		t.Errorf("app_type: got %q, want %q", site.DeployDefaults.AppType, "python")
	}
}
