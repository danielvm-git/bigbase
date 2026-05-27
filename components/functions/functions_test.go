package functions_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/functions"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func setupFunctions(t *testing.T) *functions.Functions {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	f := functions.New(functions.Options{DB: d, Logger: logger})
	k.Register(f)
	k.Register(d)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })
	return f
}

func TestFunctionsImplementsComponent(t *testing.T) {
	var _ kernel.Component = &functions.Functions{}
}

func TestFunctionsName(t *testing.T) {
	f := &functions.Functions{}
	if got := f.Name(); got != "functions" {
		t.Fatalf("expected Name()='functions', got '%s'", got)
	}
}

func TestFunctionsCreateFunction(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	body := `{"name":"hello","runtime":"javascript","source":"return {greeting: \"Hello World\"};","trigger":"http"}`
	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if name, ok := resp["name"]; !ok || name != "hello" {
		t.Fatalf("expected name 'hello', got: %v", resp)
	}
	if _, ok := resp["id"]; !ok || resp["id"] == "" {
		t.Fatalf("expected id, got: %v", resp)
	}
}

func createTestFunction(t *testing.T, h http.Handler, name string) string {
	t.Helper()
	body := `{"name":"` + name + `","source":"return 42;","trigger":"http"}`
	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %s: expected 201, got %d: %s", name, w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	id, _ := resp["id"].(string)
	return id
}

func TestFunctionsListFunctions(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	createTestFunction(t, h, "fn-a")
	createTestFunction(t, h, "fn-b")

	req := httptest.NewRequest("GET", "/api/functions", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string][]map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"]
	if len(data) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(data))
	}
}

func TestFunctionsGetFunction(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	id := createTestFunction(t, h, "get-test")

	req := httptest.NewRequest("GET", "/api/functions/"+id, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["id"] != id {
		t.Fatalf("expected id %s, got %v", id, resp)
	}
	if resp["name"] != "get-test" {
		t.Fatalf("expected name 'get-test', got %v", resp["name"])
	}
}

func TestFunctionsUpdateFunction(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	id := createTestFunction(t, h, "old-name")

	body := `{"name":"new-name","source":"return 99;","trigger":"http"}`
	req := httptest.NewRequest("PUT", "/api/functions/"+id, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	getReq := httptest.NewRequest("GET", "/api/functions/"+id, nil)
	getW := httptest.NewRecorder()
	h.ServeHTTP(getW, getReq)

	var resp map[string]any
	_ = json.NewDecoder(getW.Body).Decode(&resp)
	if resp["name"] != "new-name" {
		t.Fatalf("expected name 'new-name', got %v", resp["name"])
	}
}

func TestFunctionsDeleteFunction(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	id := createTestFunction(t, h, "delete-me")

	req := httptest.NewRequest("DELETE", "/api/functions/"+id, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	getReq := httptest.NewRequest("GET", "/api/functions/"+id, nil)
	getW := httptest.NewRecorder()
	h.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getW.Code)
	}
}

func TestFunctionsDeleteNotFound(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	req := httptest.NewRequest("DELETE", "/api/functions/nonexistent", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestFunctionsCreateInvalidJSON(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFunctionsRunFunction(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	id := createTestFunction(t, h, "runner")

	req := httptest.NewRequest("POST", "/api/functions/"+id+"/run", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["result"] == nil {
		t.Fatalf("expected result, got: %v", resp)
	}
}

func TestFunctionsRunWithLogs(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	body := `{"name":"loggy","source":"console.log(\"hello from js\"); return 99;","trigger":"http"}`
	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	id := created["id"].(string)

	runReq := httptest.NewRequest("POST", "/api/functions/"+id+"/run", nil)
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", runW.Code, runW.Body.String())
	}

	var runResp map[string]any
	_ = json.NewDecoder(runW.Body).Decode(&runResp)

	logs, _ := runResp["logs"].([]any)
	if len(logs) == 0 {
		t.Fatalf("expected logs, got: %v", runResp)
	}
	if logs[0] != "hello from js" {
		t.Fatalf("expected 'hello from js', got: %v", logs[0])
	}
}

func TestFunctionsRunNotFound(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	req := httptest.NewRequest("POST", "/api/functions/nonexistent/run", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestFunctionsCreateMissingName(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader(`{"source":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFunctionsMethodNotAllowed(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	req := httptest.NewRequest("PATCH", "/api/functions", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFunctionsEmptyID(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	req := httptest.NewRequest("GET", "/api/functions/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFunctionsUpdateMethodNotAllowed(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	req := httptest.NewRequest("PATCH", "/api/functions/some-id", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", w.Code, w.Body.String())
	}
}
