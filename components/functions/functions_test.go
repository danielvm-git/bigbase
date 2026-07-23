package functions_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/functions"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

// toInt converts a numeric value (float64 or int64) to int for test assertions.
func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func setupFunctions(t *testing.T) *functions.Functions {
	t.Helper()
	logger := testLogger{}
	k := kernel.New(logger)

	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	f := functions.New(functions.Options{DB: d, Logger: logger, ScheduleEnabled: true})
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
	bodyBytes, _ := json.Marshal(map[string]string{
		"name": name, "source": "return 42;", "trigger": "http",
	})
	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader(string(bodyBytes)))
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

func TestFunctionsRunTimeout(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	body := `{"name":"inf-loop","source":"while(true){}","trigger":"http","timeout":1}`
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
	if runResp["error"] == nil {
		t.Fatalf("expected timeout error, got: %v", runResp)
	}
}

func TestFunctionsRunJSError(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	body := `{"name":"thrower","source":"throw new Error(\"boom\");","trigger":"http"}`
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
	if runResp["error"] == nil {
		t.Fatalf("expected error, got: %v", runResp)
	}
}

func TestFunctionsRunUnsupportedRuntime(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	body := `{"name":"py","runtime":"python","source":"print('hi')","trigger":"http"}`
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

	if runW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", runW.Code, runW.Body.String())
	}
}

func TestLogs(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	// Create a function that generates console output
	createBody := `{"name":"loggy","source":"console.log('step 1'); console.log('step 2'); console.error('oops'); return 42;","trigger":"http"}`
	createReq := httptest.NewRequest("POST", "/api/functions", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	h.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createW.Code, createW.Body.String())
	}

	var created map[string]any
	_ = json.NewDecoder(createW.Body).Decode(&created)
	fnID := created["id"].(string)

	// Run the function to produce logs
	runReq := httptest.NewRequest("POST", "/api/functions/"+fnID+"/run", nil)
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("run: expected 200, got %d: %s", runW.Code, runW.Body.String())
	}

	// Run it a second time to test multiple executions
	runReq2 := httptest.NewRequest("POST", "/api/functions/"+fnID+"/run", nil)
	runW2 := httptest.NewRecorder()
	h.ServeHTTP(runW2, runReq2)

	// Fetch logs
	logsReq := httptest.NewRequest("GET", "/api/functions/"+fnID+"/logs", nil)
	logsW := httptest.NewRecorder()
	h.ServeHTTP(logsW, logsReq)

	if logsW.Code != http.StatusOK {
		t.Fatalf("logs: expected 200, got %d: %s", logsW.Code, logsW.Body.String())
	}

	var resp struct {
		Data []struct {
			ID        string   `json:"id"`
			Status    string   `json:"status"`
			Logs      []string `json:"logs"`
			CreatedAt string   `json:"created_at"`
		} `json:"data"`
	}
	if err := json.NewDecoder(logsW.Body).Decode(&resp); err != nil {
		t.Fatalf("decode logs: %v", err)
	}

	if len(resp.Data) < 2 {
		t.Fatalf("expected at least 2 executions, got %d", len(resp.Data))
	}

	// Most recent first — check the latest execution
	latest := resp.Data[0]
	if latest.Status != "success" {
		t.Fatalf("expected status 'success', got '%s'", latest.Status)
	}
	if len(latest.Logs) == 0 {
		t.Fatal("expected logs in latest execution, got empty")
	}
	if latest.CreatedAt == "" {
		t.Fatal("expected created_at timestamp")
	}

	// Verify log content
	foundStep1, foundErr := false, false
	for _, log := range latest.Logs {
		if strings.Contains(log, "step 1") {
			foundStep1 = true
		}
		if strings.Contains(log, "oops") {
			foundErr = true
		}
	}
	if !foundStep1 {
		t.Fatal("expected 'step 1' in logs")
	}
	if !foundErr {
		t.Fatal("expected 'oops' in logs")
	}
}

func TestLogsNonexistentFunction(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	req := httptest.NewRequest("GET", "/api/functions/nonexistent/logs", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogsEmptyHistory(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	// Create a function but don't run it
	createBody := `{"name":"unused","source":"return 1;","trigger":"http"}`
	createReq := httptest.NewRequest("POST", "/api/functions", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	h.ServeHTTP(createW, createReq)

	var created map[string]any
	_ = json.NewDecoder(createW.Body).Decode(&created)
	fnID := created["id"].(string)

	// Fetch logs for function with no executions
	logsReq := httptest.NewRequest("GET", "/api/functions/"+fnID+"/logs", nil)
	logsW := httptest.NewRecorder()
	h.ServeHTTP(logsW, logsReq)

	if logsW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", logsW.Code, logsW.Body.String())
	}

	var resp struct {
		Data []any `json:"data"`
	}
	_ = json.NewDecoder(logsW.Body).Decode(&resp)

	if len(resp.Data) != 0 {
		t.Fatalf("expected empty history, got %d entries", len(resp.Data))
	}
}

func TestRuntimeFetch(t *testing.T) {
	// Start a test HTTP server that returns known JSON
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"hello from server"}`))
	}))
	defer srv.Close()

	f := setupFunctions(t)
	h := f.Handler()

	// Extract host:port from test server URL for the allowlist
	srvHost := strings.TrimPrefix(srv.URL, "http://")

	// Create a function that uses fetch to call the test server
	createBody := fmt.Sprintf(`{"name":"fetcher","source":"var resp = fetch('http://'+env.TEST_HOST+'/api'); return resp;","trigger":"http","env":{"ALLOWED_HOSTS":"%s","TEST_HOST":"%s"}}`, srvHost, srvHost)
	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	fnID := created["id"].(string)

	// Run the function
	runReq := httptest.NewRequest("POST", "/api/functions/"+fnID+"/run", nil)
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("run: expected 200, got %d: %s", runW.Code, runW.Body.String())
	}

	var runResp map[string]any
	_ = json.NewDecoder(runW.Body).Decode(&runResp)

	// Should have result (not error)
	if runResp["error"] != nil {
		t.Fatalf("expected no error, got: %v", runResp["error"])
	}

	result, ok := runResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result to be map, got: %T %v", runResp["result"], runResp["result"])
	}

	// Check fetch response fields
	if status, _ := result["status"].(float64); int(status) != 200 {
		t.Fatalf("expected status 200, got: %v", result["status"])
	}
	if body, _ := result["body"].(string); !strings.Contains(body, "hello from server") {
		t.Fatalf("expected body to contain 'hello from server', got: %v", result["body"])
	}
}

func TestFetchAllowlist(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	// Create a function that tries to fetch a blocked host
	// No ALLOWED_HOSTS set, so only localhost is allowed — 127.0.0.1:9999 is NOT localhost
	createBody := `{"name":"blocked-fetcher","source":"var resp = fetch('http://169.254.169.254/latest/meta-data'); return resp;","trigger":"http","env":{}}`
	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	fnID := created["id"].(string)

	// Run — should fail with host-not-in-allowlist
	runReq := httptest.NewRequest("POST", "/api/functions/"+fnID+"/run", nil)
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("run: expected 200, got %d: %s", runW.Code, runW.Body.String())
	}

	var runResp map[string]any
	_ = json.NewDecoder(runW.Body).Decode(&runResp)

	// Should have an error about allowlist
	errMsg, _ := runResp["error"].(string)
	if errMsg == "" {
		t.Fatal("expected allowlist error, got no error")
	}
	if !strings.Contains(errMsg, "allowlist") && !strings.Contains(errMsg, "not in allowlist") {
		t.Fatalf("expected allowlist error message, got: %s", errMsg)
	}
}

func TestRuntimeEnv(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	// Create a function that reads env vars
	createBody := `{"name":"env-reader","source":"return { key: env.API_KEY, region: env.REGION };","trigger":"http","env":{"API_KEY":"sk-abc123","REGION":"us-east"}}`
	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	fnID := created["id"].(string)

	runReq := httptest.NewRequest("POST", "/api/functions/"+fnID+"/run", nil)
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("run: expected 200, got %d: %s", runW.Code, runW.Body.String())
	}

	var runResp map[string]any
	_ = json.NewDecoder(runW.Body).Decode(&runResp)

	if runResp["error"] != nil {
		t.Fatalf("expected no error, got: %v", runResp["error"])
	}

	result, ok := runResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result map, got: %T %v", runResp["result"], runResp["result"])
	}

	if key, _ := result["key"].(string); key != "sk-abc123" {
		t.Fatalf("expected env.API_KEY='sk-abc123', got: %v", result["key"])
	}
	if region, _ := result["region"].(string); region != "us-east" {
		t.Fatalf("expected env.REGION='us-east', got: %v", result["region"])
	}
}

func TestRuntimeDB(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	// Create a function that uses db.collection() to create and list records
	source := `
		var col = db.collection('messages');
		var r1 = col.create({text: 'hello', from: 'alice'});
		var r2 = col.create({text: 'world', from: 'bob'});
		var items = col.list();
		return { count: items.length, first_text: items[0].text };
	`
	createBody := fmt.Sprintf(`{"name":"db-test","source":%q,"trigger":"http"}`, source)
	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	fnID := created["id"].(string)

	runReq := httptest.NewRequest("POST", "/api/functions/"+fnID+"/run", nil)
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("run: expected 200, got %d: %s", runW.Code, runW.Body.String())
	}

	var runResp map[string]any
	_ = json.NewDecoder(runW.Body).Decode(&runResp)

	if runResp["error"] != nil {
		t.Fatalf("expected no error, got: %v", runResp["error"])
	}

	result, ok := runResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result map, got: %T %v", runResp["result"], runResp["result"])
	}

	count, _ := result["count"].(float64)
	if int(count) < 2 {
		t.Fatalf("expected at least 2 records, got count=%v", count)
	}
	first, _ := result["first_text"].(string)
	if first != "hello" {
		t.Fatalf("expected first_text='hello', got: %v", first)
	}
}

func TestRunHTTPContext(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	// Create a function that reads request context
	source := `
		return {
			method: request.method,
			header_x: request.headers['x-custom'],
			body_name: request.body.name,
			query_page: request.query.page
		};
	`
	createBody := fmt.Sprintf(`{"name":"ctx-reader","source":%q,"trigger":"http"}`, source)
	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	fnID := created["id"].(string)

	// Run with custom headers, JSON body, and query params
	runBody := `{"name":"alice","role":"admin"}`
	runReq := httptest.NewRequest("POST", "/api/functions/"+fnID+"/run?page=3", strings.NewReader(runBody))
	runReq.Header.Set("Content-Type", "application/json")
	runReq.Header.Set("X-Custom", "hello-world")
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("run: expected 200, got %d: %s", runW.Code, runW.Body.String())
	}

	var runResp map[string]any
	_ = json.NewDecoder(runW.Body).Decode(&runResp)

	if runResp["error"] != nil {
		t.Fatalf("expected no error, got: %v", runResp["error"])
	}

	result, ok := runResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result map, got: %T %v", runResp["result"], runResp["result"])
	}

	if method, _ := result["method"].(string); method != "POST" {
		t.Fatalf("expected method='POST', got: %v", result["method"])
	}
	if hdr, _ := result["header_x"].(string); hdr != "hello-world" {
		t.Fatalf("expected header_x='hello-world', got: %v", result["header_x"])
	}
	if bodyName, _ := result["body_name"].(string); bodyName != "alice" {
		t.Fatalf("expected body_name='alice', got: %v", result["body_name"])
	}
	if queryPage, _ := result["query_page"].(string); queryPage != "3" {
		t.Fatalf("expected query_page='3', got: %v", result["query_page"])
	}
}

func TestSchedule(t *testing.T) {
	f := setupFunctions(t)
	h := f.Handler()

	// Create a scheduled function that logs a message
	source := `console.log('scheduled run at ' + new Date().toISOString()); return {ok: true};`
	createBody := fmt.Sprintf(`{"name":"cron-job","source":%q,"trigger":"schedule","schedule":"@every 1s"}`, source)
	req := httptest.NewRequest("POST", "/api/functions", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	fnID := created["id"].(string)

	// Wait for at least one scheduled execution
	time.Sleep(2500 * time.Millisecond)

	// Check execution logs
	logsReq := httptest.NewRequest("GET", "/api/functions/"+fnID+"/logs", nil)
	logsW := httptest.NewRecorder()
	h.ServeHTTP(logsW, logsReq)

	if logsW.Code != http.StatusOK {
		t.Fatalf("logs: expected 200, got %d: %s", logsW.Code, logsW.Body.String())
	}

	var resp struct {
		Data []struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.NewDecoder(logsW.Body).Decode(&resp)

	if len(resp.Data) == 0 {
		t.Fatal("expected at least one scheduled execution, got none")
	}

	// At least one execution should have status 'success'
	found := false
	for _, exec := range resp.Data {
		if exec.Status == "success" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected at least one successful scheduled execution")
	}
}

func TestDBCrossTenantIsolation(t *testing.T) {
	// Verifies that db.collection() queries are scoped by org_id.
	// Two different orgs should not see each other's data.
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	f := functions.New(functions.Options{DB: d, Logger: logger})
	k.Register(f)
	k.Register(d)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	rt := functions.NewJSRuntime()

	createSrc := `
		var col = db.collection('items');
		col.create({value: env.ITEM_VALUE});
	`
	listSrc := `
		var col = db.collection('items');
		var items = col.list();
		return {count: items.length};
	`

	// Org 1 creates a record
	_, err := rt.Execute(createSrc, 30, functions.RunContext{
		DB:    d,
		OrgID: 1,
		Env:   map[string]string{"ITEM_VALUE": "org1-secret"},
	})
	if err != nil {
		t.Fatalf("org1 create: %v", err)
	}

	// Org 2 creates a record
	_, err = rt.Execute(createSrc, 30, functions.RunContext{
		DB:    d,
		OrgID: 2,
		Env:   map[string]string{"ITEM_VALUE": "org2-secret"},
	})
	if err != nil {
		t.Fatalf("org2 create: %v", err)
	}

	// Org 1 should see only its own record
	out1, err := rt.Execute(listSrc, 30, functions.RunContext{
		DB:    d,
		OrgID: 1,
	})
	if err != nil {
		t.Fatalf("org1 list: %v", err)
	}
	res1, ok := out1.Result.(map[string]any)
	if !ok {
		t.Fatalf("org1 list type: %T", out1.Result)
	}
	if count := toInt(res1["count"]); count != 1 {
		t.Fatalf("org1: expected 1 item (isolation broken!), got %d", count)
	}

	// Org 2 should see only its own record
	out2, err := rt.Execute(listSrc, 30, functions.RunContext{
		DB:    d,
		OrgID: 2,
	})
	if err != nil {
		t.Fatalf("org2 list: %v", err)
	}
	res2, ok := out2.Result.(map[string]any)
	if !ok {
		t.Fatalf("org2 list type: %T", out2.Result)
	}
	if count := toInt(res2["count"]); count != 1 {
		t.Fatalf("org2: expected 1 item (isolation broken!), got %d", count)
	}

	// Cross-tenant get should return null
	getSrc := `
		var col = db.collection('items');
		return col.get(OTHER_ID);
	`
	getSrcOrg2 := strings.Replace(getSrc, "OTHER_ID", "1", 1)
	outGet, err := rt.Execute(getSrcOrg2, 30, functions.RunContext{
		DB:    d,
		OrgID: 2,
	})
	if err != nil {
		t.Fatalf("cross-tenant get: %v", err)
	}
	if outGet.Result != nil {
		t.Fatalf("cross-tenant get: expected null, got %v", outGet.Result)
	}
}
