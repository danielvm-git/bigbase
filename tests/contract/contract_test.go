package contract

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/components/monitoring"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

func inMemoryDB(t *testing.T) *db.DB {
	t.Helper()
	return db.New(db.Options{Path: ":memory:", Logger: testLogger{}})
}

func jsonBody(t *testing.T, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	return bytes.NewReader(b)
}

func doRequest(t *testing.T, h http.Handler, method, path string, body io.Reader) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w.Result()
}

func decodeJSON(t *testing.T, res *http.Response, v any) {
	t.Helper()
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("json decode: %v\nbody: %s", err, string(body))
	}
}

func assertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("status code: got %d, want %d", got, want)
	}
}

func assertField(t *testing.T, got map[string]any, field, wantType string) {
	t.Helper()
	v, ok := got[field]
	if !ok {
		t.Errorf("response missing field %q", field)
		return
	}
	switch wantType {
	case "string":
		if _, ok := v.(string); !ok {
			t.Errorf("field %q: want string, got %T (val: %v)", field, v, v)
		}
	case "number":
		if _, ok := v.(float64); !ok {
			t.Errorf("field %q: want number, got %T (val: %v)", field, v, v)
		}
	case "map":
		if _, ok := v.(map[string]any); !ok {
			t.Errorf("field %q: want object, got %T (val: %v)", field, v, v)
		}
	case "array":
		if _, ok := v.([]any); !ok {
			t.Errorf("field %q: want array, got %T (val: %v)", field, v, v)
		}
	}
}

func TestAuthContract(t *testing.T) {
	d := inMemoryDB(t)
	a := auth.New(auth.Options{DB: d, Logger: testLogger{}, Secret: "test-secret-32-chars!!!"})

	k := kernel.New(testLogger{})
	k.Register(d)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	h := a.Handler()

	// POST /api/auth/register — returns {token, user: {id, email}}
	res := doRequest(t, h, "POST", "/api/auth/register",
		jsonBody(t, map[string]string{"email": "ct@test.com", "password": "Secret123!"}))
	assertStatus(t, res.StatusCode, 201)

	var reg map[string]any
	decodeJSON(t, res, &reg)
	assertField(t, reg, "token", "string")
	assertField(t, reg, "user", "map")
	regUser := reg["user"].(map[string]any)
	assertField(t, regUser, "email", "string")
	assertField(t, regUser, "id", "number")

	// POST /api/auth/login — returns {token, user: {id, email}}
	res = doRequest(t, h, "POST", "/api/auth/login",
		jsonBody(t, map[string]string{"email": "ct@test.com", "password": "Secret123!"}))
	assertStatus(t, res.StatusCode, 200)

	var login map[string]any
	decodeJSON(t, res, &login)
	assertField(t, login, "token", "string")
	assertField(t, login, "user", "map")
	loginUser := login["user"].(map[string]any)
	assertField(t, loginUser, "email", "string")
	assertField(t, loginUser, "id", "number")
}

func TestMonitoringContract(t *testing.T) {
	m := monitoring.New(monitoring.Options{})

	k := kernel.New(testLogger{})
	d := inMemoryDB(t)
	k.Register(d)
	k.Register(m)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	h := m.Handler()
	res := doRequest(t, h, "GET", "/api/monitoring/health", nil)
	assertStatus(t, res.StatusCode, 200)

	var health map[string]any
	decodeJSON(t, res, &health)
	assertField(t, health, "status", "string")
}

func TestSitesContract(t *testing.T) {
	d := inMemoryDB(t)
	g := git.New(git.Options{DB: d, Logger: testLogger{}})
	s := sites.New(sites.Options{DB: d, Logger: testLogger{}})

	k := kernel.New(testLogger{})
	k.Register(d)
	k.Register(g)
	k.Register(s)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	h := s.Handler()

	res := doRequest(t, h, "GET", "/api/sites", nil)
	assertStatus(t, res.StatusCode, 200)

	var list map[string]any
	decodeJSON(t, res, &list)
	assertField(t, list, "data", "array")

	res = doRequest(t, h, "POST", "/api/sites",
		jsonBody(t, map[string]string{"name": "contract-test"}))
	assertStatus(t, res.StatusCode, 400)

	var errBody map[string]any
	decodeJSON(t, res, &errBody)
	assertField(t, errBody, "error", "string")
}

func TestDeployContract(t *testing.T) {
	dd := inMemoryDB(t)
	g := git.New(git.Options{DB: dd, Logger: testLogger{}})
	dpl := deploy.New(deploy.Options{DB: dd, Logger: testLogger{}})

	k := kernel.New(testLogger{})
	k.Register(dd)
	k.Register(g)
	k.Register(dpl)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	h := dpl.Handler()

	res := doRequest(t, h, "GET", "/api/deploy", nil)
	assertStatus(t, res.StatusCode, 200)

	var list map[string]any
	decodeJSON(t, res, &list)
	assertField(t, list, "data", "array")

	res = doRequest(t, h, "POST", "/api/deploy",
		jsonBody(t, map[string]string{"branch": "main"}))
	assertStatus(t, res.StatusCode, 400)

	var errBody map[string]any
	decodeJSON(t, res, &errBody)
	assertField(t, errBody, "error", "string")
}
