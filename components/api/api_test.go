package api_test

import (
	"encoding/json"
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

func parseResponse(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	return resp
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

	req := httptest.NewRequest("GET", "/api/collections/posts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w.Body.Bytes())
	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got: %v", resp)
	}
	if len(data) != 0 {
		t.Fatalf("expected empty data, got %d items", len(data))
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

	resp := parseResponse(t, w.Body.Bytes())
	id, ok := resp["id"].(float64)
	if !ok || id != 1 {
		t.Fatalf("expected id=1, got: %v", resp)
	}
}

func TestAPIGetRecord(t *testing.T) {
	_, handler := setupAPI(t)

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

	resp := parseResponse(t, w.Body.Bytes())
	if title, ok := resp["title"].(string); !ok || title != "hello" {
		t.Fatalf("expected title 'hello', got: %v", resp)
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

	resp := parseResponse(t, w2.Body.Bytes())
	if title, ok := resp["title"].(string); !ok || title != "updated" {
		t.Fatalf("expected title 'updated', got: %v", resp)
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

func TestAPIListCollections(t *testing.T) {
	_, handler := setupAPI(t)

	create := func(name string) {
		req := httptest.NewRequest("POST", "/api/collections/"+name, strings.NewReader(`{"a":1}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s: expected 201, got %d", name, w.Code)
		}
	}
	create("posts")
	create("comments")

	req := httptest.NewRequest("GET", "/api/collections/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string][]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp["data"]) != 2 {
		t.Fatalf("expected 2 collections (posts, comments), got %v", resp["data"])
	}
}

func TestAPIExecuteSQLSelect(t *testing.T) {
	_, handler := setupAPI(t)

	// Seed a record
	post := httptest.NewRequest("POST", "/api/collections/posts", strings.NewReader(`{"title":"hello"}`))
	post.Header.Set("Content-Type", "application/json")
	w0 := httptest.NewRecorder()
	handler.ServeHTTP(w0, post)
	if w0.Code != http.StatusCreated {
		t.Fatalf("seed: expected 201, got %d", w0.Code)
	}

	body := `{"query":"SELECT id, data FROM posts ORDER BY id"}`
	req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["columns"]; !ok {
		t.Fatalf("expected columns in response, got: %v", resp)
	}
	if _, ok := resp["rows"]; !ok {
		t.Fatalf("expected rows in response, got: %v", resp)
	}
	cols, _ := resp["columns"].([]any)
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d: %v", len(cols), cols)
	}
	rows, _ := resp["rows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
}

func TestAPIExecuteSQLInvalidQuery(t *testing.T) {
	_, handler := setupAPI(t)

	body := `{"query":"DROP TABLE posts"}`
	req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for DROP, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLMultiStatement(t *testing.T) {
	_, handler := setupAPI(t)

	body := `{"query":"SELECT 1; DROP TABLE posts"}`
	req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for multi-statement, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLWith(t *testing.T) {
	_, handler := setupAPI(t)

	body := `{"query":"WITH t AS (SELECT 1 AS n) SELECT * FROM t"}`
	req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for WITH, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLSyntaxError(t *testing.T) {
	_, handler := setupAPI(t)

	body := `{"query":"SELECTT oops FROM posts"}`
	req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad syntax, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLEmptyQuery(t *testing.T) {
	_, handler := setupAPI(t)

	body := `{"query":""}`
	req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLMissingBody(t *testing.T) {
	_, handler := setupAPI(t)

	req := httptest.NewRequest("POST", "/api/sql", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLWrongMethod(t *testing.T) {
	_, handler := setupAPI(t)

	req := httptest.NewRequest("GET", "/api/sql", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIMissingRecord(t *testing.T) {
	_, handler := setupAPI(t)

	req := httptest.NewRequest("GET", "/api/collections/posts/999", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIMalformedJSON(t *testing.T) {
	_, handler := setupAPI(t)

	req := httptest.NewRequest("POST", "/api/collections/posts", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIMethodNotAllowed(t *testing.T) {
	_, handler := setupAPI(t)

	req := httptest.NewRequest("PUT", "/api/collections/posts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIPatchWithoutID(t *testing.T) {
	_, handler := setupAPI(t)

	req := httptest.NewRequest("PATCH", "/api/collections/posts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIDeleteWithoutID(t *testing.T) {
	_, handler := setupAPI(t)

	req := httptest.NewRequest("DELETE", "/api/collections/posts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIAccessMultipleCollections(t *testing.T) {
	_, handler := setupAPI(t)

	post1 := httptest.NewRequest("POST", "/api/collections/posts", strings.NewReader(`{"title":"post-1"}`))
	post1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, post1)

	post2 := httptest.NewRequest("POST", "/api/collections/comments", strings.NewReader(`{"text":"comment-1"}`))
	post2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, post2)

	if w1.Code != http.StatusCreated || w2.Code != http.StatusCreated {
		t.Fatalf("expected both 201, got posts=%d comments=%d", w1.Code, w2.Code)
	}
}

func TestAPIListRecordsPagination(t *testing.T) {
	_, handler := setupAPI(t)

	for i := 0; i < 5; i++ {
		body := strings.NewReader(`{"n":` + string(rune('0'+i)) + `}`)
		req := httptest.NewRequest("POST", "/api/collections/posts", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to seed record %d: %d", i, w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/api/collections/posts?limit=2&offset=1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := parseResponse(t, w.Body.Bytes())
	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got: %v", resp)
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 records with limit=2, got %d", len(data))
	}
}
