package api_test

import (
	"encoding/json"
	"fmt"
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

func withAdminRole(base http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := api.WithUserRole(r.Context(), "admin")
		base.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TestAPIExecuteSQLSelect(t *testing.T) {
	_, handler := setupAPI(t)
	sqlHandler := withAdminRole(handler)

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
	sqlHandler.ServeHTTP(w, req)

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
	sqlHandler := withAdminRole(handler)

	body := `{"query":"DROP TABLE posts"}`
	req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sqlHandler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for DROP, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLMultiStatement(t *testing.T) {
	_, handler := setupAPI(t)
	sqlHandler := withAdminRole(handler)

	body := `{"query":"SELECT 1; DROP TABLE posts"}`
	req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sqlHandler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for multi-statement, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLWith(t *testing.T) {
	_, handler := setupAPI(t)
	sqlHandler := withAdminRole(handler)

	body := `{"query":"WITH t AS (SELECT 1 AS n) SELECT * FROM t"}`
	req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sqlHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for WITH, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLSyntaxError(t *testing.T) {
	_, handler := setupAPI(t)
	sqlHandler := withAdminRole(handler)

	body := `{"query":"SELECTT oops FROM posts"}`
	req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sqlHandler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad syntax, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLEmptyQuery(t *testing.T) {
	_, handler := setupAPI(t)
	sqlHandler := withAdminRole(handler)

	body := `{"query":""}`
	req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sqlHandler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLMissingBody(t *testing.T) {
	_, handler := setupAPI(t)
	sqlHandler := withAdminRole(handler)

	req := httptest.NewRequest("POST", "/api/sql", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	sqlHandler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLWrongMethod(t *testing.T) {
	_, handler := setupAPI(t)
	sqlHandler := withAdminRole(handler)

	req := httptest.NewRequest("GET", "/api/sql", nil)
	w := httptest.NewRecorder()
	sqlHandler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSQLRequiresAdmin(t *testing.T) {
	_, handler := setupAPI(t)

	t.Run("no_role_returns_403", func(t *testing.T) {
		body := `{"query":"SELECT 1"}`
		req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp["error"] != "forbidden" {
			t.Fatalf("expected 'forbidden', got %v", resp["error"])
		}
	})

	t.Run("non_admin_role_returns_403", func(t *testing.T) {
		body := `{"query":"SELECT 1"}`
		req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(api.WithUserRole(req.Context(), "user"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for non-admin role, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("admin_role_allows_query", func(t *testing.T) {
		body := `{"query":"SELECT 1 AS n"}`
		req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(api.WithUserRole(req.Context(), "admin"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for admin role, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestSQLOrgScoped(t *testing.T) {
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

	base := a.Handler()

	// Helper: inject admin role + optional org_id
	adminWithOrg := func(orgID int64, r *http.Request) *http.Request {
		ctx := r.Context()
		ctx = api.WithUserRole(ctx, "admin")
		if orgID > 0 {
			ctx = api.WithOrgID(ctx, orgID)
		}
		return r.WithContext(ctx)
	}

	// Seed: org 1 creates a record
	post1 := httptest.NewRequest("POST", "/api/collections/items",
		strings.NewReader(`{"name":"org1-item"}`))
	post1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	base.ServeHTTP(w1, adminWithOrg(1, post1))
	if w1.Code != http.StatusCreated {
		t.Fatalf("org1 create: expected 201, got %d: %s", w1.Code, w1.Body.String())
	}

	// Seed: org 2 creates a record
	post2 := httptest.NewRequest("POST", "/api/collections/items",
		strings.NewReader(`{"name":"org2-item"}`))
	post2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	base.ServeHTTP(w2, adminWithOrg(2, post2))
	if w2.Code != http.StatusCreated {
		t.Fatalf("org2 create: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	// Org 1 queries — should only see their record
	query1 := httptest.NewRequest("POST", "/api/sql",
		strings.NewReader(`{"query":"SELECT id, data FROM items ORDER BY id"}`))
	query1.Header.Set("Content-Type", "application/json")
	wQ1 := httptest.NewRecorder()
	base.ServeHTTP(wQ1, adminWithOrg(1, query1))
	if wQ1.Code != http.StatusOK {
		t.Fatalf("org1 sql: expected 200, got %d: %s", wQ1.Code, wQ1.Body.String())
	}

	var resp1 map[string]any
	if err := json.NewDecoder(wQ1.Body).Decode(&resp1); err != nil {
		t.Fatalf("decode org1 result: %v", err)
	}
	rows1, _ := resp1["rows"].([]any)
	if len(rows1) != 1 {
		t.Fatalf("org1: expected 1 row, got %d", len(rows1))
	}

	// Org 2 queries — should only see their record
	query2 := httptest.NewRequest("POST", "/api/sql",
		strings.NewReader(`{"query":"SELECT id, data FROM items ORDER BY id"}`))
	query2.Header.Set("Content-Type", "application/json")
	wQ2 := httptest.NewRecorder()
	base.ServeHTTP(wQ2, adminWithOrg(2, query2))
	if wQ2.Code != http.StatusOK {
		t.Fatalf("org2 sql: expected 200, got %d: %s", wQ2.Code, wQ2.Body.String())
	}

	var resp2 map[string]any
	if err := json.NewDecoder(wQ2.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode org2 result: %v", err)
	}
	rows2, _ := resp2["rows"].([]any)
	if len(rows2) != 1 {
		t.Fatalf("org2: expected 1 row, got %d", len(rows2))
	}
}

func TestExtractTableRef(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"SELECT * FROM users", "users"},
		{"SELECT id, name FROM posts WHERE id = 1", "posts"},
		{"SELECT * FROM orders JOIN items ON orders.id = items.order_id", "orders"},
		{"SELECT * FROM users ORDER BY id", "users"},
		{"SELECT * FROM users LIMIT 10", "users"},
		{"WITH t AS (SELECT 1 AS n) SELECT * FROM t", "t"},
		{"EXPLAIN SELECT * FROM users", "users"},
		{"EXPLAIN QUERY PLAN SELECT * FROM users", "users"},
		{"PRAGMA table_info('users')", ""},
		{"SELECT 1", ""},
	}

	for _, tc := range tests {
		t.Run(tc.query[:min(len(tc.query), 40)], func(t *testing.T) {
			got := api.ExtractTableRef(tc.query)
			if got != tc.want {
				t.Errorf("extractTableRef(%q) = %q, want %q", tc.query, got, tc.want)
			}
		})
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
		body := strings.NewReader(fmt.Sprintf(`{"n":%d}`, i))
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

func TestAPIComponentTableExcludedFromCollections(t *testing.T) {
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

	// Simulate another component creating a table with a name matching internalTables
	if err := d.Migrate("CREATE TABLE deployments (id TEXT PRIMARY KEY, repo_id TEXT, status TEXT)"); err != nil {
		t.Fatalf("create component table: %v", err)
	}

	// Create a user collection to verify it still appears
	if err := d.Migrate("CREATE TABLE mydata (id INTEGER PRIMARY KEY AUTOINCREMENT, data TEXT)"); err != nil {
		t.Fatalf("create user collection: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/collections/", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string][]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	for _, name := range resp["data"] {
		if name == "deployments" {
			t.Fatal("component table 'deployments' should be excluded from collections")
		}
	}

	found := false
	for _, name := range resp["data"] {
		if name == "mydata" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("user collection 'mydata' should still appear in collections")
	}
}

func TestAPIComponentTableAccessNoCrash(t *testing.T) {
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

	// Create a component-style table with different schema
	if err := d.Migrate("CREATE TABLE functions (id TEXT PRIMARY KEY, name TEXT, runtime TEXT)"); err != nil {
		t.Fatalf("create component table: %v", err)
	}

	// Accessing a component table directly should not crash with 500
	req := httptest.NewRequest("GET", "/api/collections/functions", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, req)

	if w.Code == http.StatusInternalServerError {
		t.Fatalf("expected no 500 when accessing component table, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIDeleteNonExistentRecord(t *testing.T) {
	_, handler := setupAPI(t)

	// Table must exist first
	post := httptest.NewRequest("POST", "/api/collections/posts", strings.NewReader(`{"a":1}`))
	post.Header.Set("Content-Type", "application/json")
	w0 := httptest.NewRecorder()
	handler.ServeHTTP(w0, post)

	req := httptest.NewRequest("DELETE", "/api/collections/posts/999", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent record, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIExecuteSQLSecurityChecks(t *testing.T) {
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

	base := a.Handler()

	adminWithOrg := func(orgID int64, r *http.Request) *http.Request {
		ctx := r.Context()
		ctx = api.WithUserRole(ctx, "admin")
		if orgID > 0 {
			ctx = api.WithOrgID(ctx, orgID)
		}
		return r.WithContext(ctx)
	}

	// Create collection table to test
	post := httptest.NewRequest("POST", "/api/collections/items", strings.NewReader(`{"name":"test"}`))
	post.Header.Set("Content-Type", "application/json")
	w0 := httptest.NewRecorder()
	base.ServeHTTP(w0, adminWithOrg(1, post))
	if w0.Code != http.StatusCreated {
		t.Fatalf("seed collection: expected 201, got %d", w0.Code)
	}

	t.Run("UNION_query_rejected_on_collections", func(t *testing.T) {
		body := `{"query":"SELECT id FROM items UNION SELECT id FROM items"}`
		req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		base.ServeHTTP(w, adminWithOrg(1, req))

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for UNION query on collection, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("JOIN_query_rejected_on_collections", func(t *testing.T) {
		body := `{"query":"SELECT * FROM items JOIN items AS i2"}`
		req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		base.ServeHTTP(w, adminWithOrg(1, req))

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for JOIN query on collection, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Subquery_rejected_on_collections", func(t *testing.T) {
		body := `{"query":"SELECT * FROM items WHERE id IN (SELECT id FROM items)"}`
		req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		base.ServeHTTP(w, adminWithOrg(1, req))

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for subquery on collection, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Whitespace_normalization_FROM_newline", func(t *testing.T) {
		body := `{"query":"SELECT id\nFROM\nitems"}`
		req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		base.ServeHTTP(w, adminWithOrg(1, req))

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Internal_table_word_boundary_check", func(t *testing.T) {
		// Custom collection containing internal table substring (e.g. "my_users") should succeed
		postUserCol := httptest.NewRequest("POST", "/api/collections/my_users", strings.NewReader(`{"username":"alice"}`))
		postUserCol.Header.Set("Content-Type", "application/json")
		wUserCol := httptest.NewRecorder()
		base.ServeHTTP(wUserCol, adminWithOrg(1, postUserCol))
		if wUserCol.Code != http.StatusCreated {
			t.Fatalf("seed custom collection my_users: expected 201, got %d: %s", wUserCol.Code, wUserCol.Body.String())
		}

		body := `{"query":"SELECT * FROM my_users"}`
		req := httptest.NewRequest("POST", "/api/sql", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		base.ServeHTTP(w, adminWithOrg(1, req))

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 for custom my_users collection query, got %d: %s", w.Code, w.Body.String())
		}

		// Exact internal table "users" should be blocked
		bodyBlocked := `{"query":"SELECT * FROM users"}`
		reqBlocked := httptest.NewRequest("POST", "/api/sql", strings.NewReader(bodyBlocked))
		reqBlocked.Header.Set("Content-Type", "application/json")
		wBlocked := httptest.NewRecorder()
		base.ServeHTTP(wBlocked, adminWithOrg(1, reqBlocked))

		if wBlocked.Code != http.StatusForbidden {
			t.Errorf("expected 403 for blocked internal table users, got %d: %s", wBlocked.Code, wBlocked.Body.String())
		}
	})
}
