package git_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupGit(t *testing.T) *git.Git {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	dir := t.TempDir()

	g := git.New(git.Options{DB: d, Logger: logger, Dir: dir})
	k.Register(g)
	k.Register(d)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	return g
}

func TestGitImplementsComponent(t *testing.T) {
	var _ kernel.Component = &git.Git{}
}

func TestGitName(t *testing.T) {
	g := &git.Git{}
	if got := g.Name(); got != "git" {
		t.Fatalf("expected Name()='git', got '%s'", got)
	}
}

func TestGitVersion(t *testing.T) {
	g := &git.Git{}
	if got := g.Version(); got == "" {
		t.Fatal("expected non-empty version")
	}
}

func TestGitDependencies(t *testing.T) {
	g := &git.Git{}
	deps := g.Dependencies()
	if len(deps) != 1 || deps[0] != "db" {
		t.Fatalf("expected dependency on 'db', got %v", deps)
	}
}

func TestGitCreateRepo(t *testing.T) {
	g := setupGit(t)
	handler := g.Handler()

	body := `{"name":"myproject","description":"My first repo"}`
	req := httptest.NewRequest("POST", "/api/git/repos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if name, ok := resp["name"]; !ok || name != "myproject" {
		t.Fatalf("expected name 'myproject', got: %v", resp)
	}
	if _, ok := resp["id"]; !ok || resp["id"] == "" {
		t.Fatalf("expected non-empty id, got: %v", resp)
	}
}

func TestGitCreateDuplicateRepo(t *testing.T) {
	g := setupGit(t)
	handler := g.Handler()

	body := `{"name":"dup"}`
	req := httptest.NewRequest("POST", "/api/git/repos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", w.Code)
	}

	req2 := httptest.NewRequest("POST", "/api/git/repos", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestGitListRepos(t *testing.T) {
	g := setupGit(t)
	handler := g.Handler()

	for _, name := range []string{"repo-a", "repo-b"} {
		body := `{"name":"` + name + `"}`
		req := httptest.NewRequest("POST", "/api/git/repos", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s: expected 201, got %d", name, w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/api/git/repos", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string][]map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"]
	if len(data) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(data))
	}
}
