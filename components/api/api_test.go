package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/api"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupAPI(t *testing.T) (*api.API, http.Handler) {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := api.New(api.Options{DB: d, Logger: logger})

	k.Register(a)
	k.Register(d)

	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	return a, a.Handler()
}

func TestAPIImplementsComponent(t *testing.T) {
	var _ kernel.Component = &api.API{}
}

func TestAPIName(t *testing.T) {
	a := &api.API{}
	if got := a.Name(); got != "api" {
		t.Fatalf("expected Name()='api', got '%s'", got)
	}
}

func TestAPICreateCollection(t *testing.T) {
	_, handler := setupAPI(t)

	// list should return empty array
	req := httptest.NewRequest("GET", "/api/collections/posts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"data":[]`) {
		t.Fatalf("expected empty data array, got: %s", w.Body.String())
	}
}

func TestAPICreateRecord(t *testing.T) {
	_, handler := setupAPI(t)

	req := httptest.NewRequest("POST", "/api/collections/posts", strings.NewReader(`{"title":"hello","body":"world"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"id":1`) {
		t.Fatalf("expected created record with id=1, got: %s", w.Body.String())
	}
}

func TestAPIGetRecord(t *testing.T) {
	_, handler := setupAPI(t)

	// create
	post := httptest.NewRequest("POST", "/api/collections/posts", strings.NewReader(`{"title":"hello","body":"world"}`))
	post.Header.Set("Content-Type", "application/json")
	w0 := httptest.NewRecorder()
	handler.ServeHTTP(w0, post)

	get := httptest.NewRequest("GET", "/api/collections/posts/1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, get)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"title":"hello"`) {
		t.Fatalf("expected title in response, got: %s", w.Body.String())
	}
}

func TestAPIUpdateRecord(t *testing.T) {
	_, handler := setupAPI(t)

	post := httptest.NewRequest("POST", "/api/collections/posts", strings.NewReader(`{"title":"hello","body":"world"}`))
	post.Header.Set("Content-Type", "application/json")
	w0 := httptest.NewRecorder()
	handler.ServeHTTP(w0, post)

	patch := httptest.NewRequest("PATCH", "/api/collections/posts/1", strings.NewReader(`{"title":"updated"}`))
	patch.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, patch)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	get := httptest.NewRequest("GET", "/api/collections/posts/1", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, get)
	if !strings.Contains(w2.Body.String(), `"title":"updated"`) {
		t.Fatalf("expected updated title, got: %s", w2.Body.String())
	}
}

func TestAPIDeleteRecord(t *testing.T) {
	_, handler := setupAPI(t)

	post := httptest.NewRequest("POST", "/api/collections/posts", strings.NewReader(`{"title":"hello","body":"world"}`))
	post.Header.Set("Content-Type", "application/json")
	w0 := httptest.NewRecorder()
	handler.ServeHTTP(w0, post)

	del := httptest.NewRequest("DELETE", "/api/collections/posts/1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, del)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	get := httptest.NewRequest("GET", "/api/collections/posts/1", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, get)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w2.Code)
	}
}
