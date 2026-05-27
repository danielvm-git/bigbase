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
