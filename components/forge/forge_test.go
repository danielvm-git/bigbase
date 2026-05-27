package forge_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/forge"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupForge(t *testing.T) *forge.Forge {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	f := forge.New(forge.Options{DB: d, Logger: logger})
	k.Register(f)
	k.Register(d)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return f
}

func TestForgeImplementsComponent(t *testing.T) {
	var _ kernel.Component = &forge.Forge{}
}

func TestForgeName(t *testing.T) {
	f := &forge.Forge{}
	if got := f.Name(); got != "forge" {
		t.Fatalf("expected Name()='forge', got '%s'", got)
	}
}

func TestForgeCreateIssue(t *testing.T) {
	f := setupForge(t)
	h := f.Handler()

	body := `{"repo_id":"test-repo","title":"Bug: login fails","description":"Cannot login with valid credentials"}`
	req := httptest.NewRequest("POST", "/api/forge/issues", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if title, ok := resp["title"]; !ok || title != "Bug: login fails" {
		t.Fatalf("expected title, got: %v", resp)
	}
	if _, ok := resp["id"]; !ok || resp["id"] == "" {
		t.Fatalf("expected id, got: %v", resp)
	}
}

func TestForgeListIssues(t *testing.T) {
	f := setupForge(t)
	h := f.Handler()

	for _, title := range []string{"Issue A", "Issue B"} {
		body := `{"repo_id":"r","title":"` + title + `"}`
		req := httptest.NewRequest("POST", "/api/forge/issues", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s: expected 201, got %d", title, w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/api/forge/issues?repo_id=r", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string][]map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"]
	if len(data) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(data))
	}
}

func TestForgeCreateLabel(t *testing.T) {
	f := setupForge(t)
	h := f.Handler()

	body := `{"repo_id":"r","name":"bug","color":"#ff0000"}`
	req := httptest.NewRequest("POST", "/api/forge/labels", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestForgeAddComment(t *testing.T) {
	f := setupForge(t)
	h := f.Handler()

	// Create issue
	createResp := httptest.NewRequest("POST", "/api/forge/issues", strings.NewReader(`{"repo_id":"r","title":"Test"}`))
	createResp.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, createResp)

	var issue map[string]any
	_ = json.NewDecoder(w.Body).Decode(&issue)

	// Add comment
	body := `{"content":"This is a comment"}`
	id := issue["id"].(string)
	req := httptest.NewRequest("POST", "/api/forge/issues/" + id + "/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req)

	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestForgeKanbanBoard(t *testing.T) {
	f := setupForge(t)
	h := f.Handler()

	// Create two issues with different statuses
	for _, s := range []struct{ title, status string }{
		{"To Do", "open"},
		{"In Progress", "in_progress"},
	} {
		body := `{"repo_id":"r","title":"` + s.title + `","status":"` + s.status + `"}`
		req := httptest.NewRequest("POST", "/api/forge/issues", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s: %d", s.title, w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/api/forge/board?repo_id=r", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string][]map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if len(resp["open"]) != 1 || len(resp["in_progress"]) != 1 {
		t.Fatalf("expected 1 open + 1 in_progress, got: open=%d in_progress=%d", len(resp["open"]), len(resp["in_progress"]))
	}
}

func TestForgeWikiPage(t *testing.T) {
	f := setupForge(t)
	h := f.Handler()

	body := `{"content":"# Welcome to the wiki","repo_id":"r"}`
	req := httptest.NewRequest("PUT", "/api/forge/wiki/Home", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	getReq := httptest.NewRequest("GET", "/api/forge/wiki/Home?repo_id=r", nil)
	getW := httptest.NewRecorder()
	h.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var page map[string]any
	_ = json.NewDecoder(getW.Body).Decode(&page)
	if content, ok := page["content"]; !ok || content != "# Welcome to the wiki" {
		t.Fatalf("expected wiki content, got: %v", page)
	}
}
