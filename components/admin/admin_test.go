package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/admin"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}
func (testLogger) Warn(msg string, args ...any)  {}

func TestAdminImplementsComponent(t *testing.T) {
	var _ kernel.Component = &admin.Admin{}
}

func TestAdminName(t *testing.T) {
	a := &admin.Admin{}
	if got := a.Name(); got != "admin" {
		t.Fatalf("expected Name()='admin', got '%s'", got)
	}
}

func TestAdminVersion(t *testing.T) {
	a := &admin.Admin{}
	if got := a.Version(); got != "0.1.0" {
		t.Fatalf("expected Version()='0.1.0', got '%s'", got)
	}
}

func TestAdminDependencies(t *testing.T) {
	a := &admin.Admin{}
	if got := a.Dependencies(); got != nil {
		t.Fatalf("expected no dependencies, got %v", got)
	}
}

func TestAdminServesIndexHTML(t *testing.T) {
	a := admin.New(admin.Options{Logger: testLogger{}})
	handler := a.Handler()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Fatalf("expected text/html, got %q", ct)
	}
	if !contains(w.Body.String(), "<!doctype html>") {
		t.Fatalf("expected HTML, got: %s", w.Body.String()[:100])
	}
}

func TestAdminServesAssets(t *testing.T) {
	a := admin.New(admin.Options{Logger: testLogger{}})
	handler := a.Handler()

	req := httptest.NewRequest("GET", "/favicon.svg", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for favicon, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Fatalf("expected image/svg+xml, got %q", ct)
	}
}

func TestAdminReturns404(t *testing.T) {
	a := admin.New(admin.Options{Logger: testLogger{}})
	handler := a.Handler()

	req := httptest.NewRequest("GET", "/nonexistent.txt", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAdminHandler(t *testing.T) {
	a := admin.New(admin.Options{Logger: testLogger{}})
	h := a.Handler()
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
