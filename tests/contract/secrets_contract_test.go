package contract

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/api"
	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/projects"
	"github.com/danielvm/bigbase/components/secrets"
	"github.com/danielvm/bigbase/kernel"
)

// secrets_contract_test.go locks the e89s04 HTTP contract: masked metadata
// responses, explicit value reads, the role/action matrix, non-disclosing
// cross-org errors, security bounds, rate limits, and SQL safety — all through
// the canonical route surface.

func contractRootKey(t *testing.T) []byte {
	t.Helper()
	raw := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	key, err := secrets.ParseRootKey(raw)
	if err != nil {
		t.Fatalf("parse root key: %v", err)
	}
	return key
}

// contractSecretsFixture boots the secret stack and returns the adapter
// handler, the projects component, and the auth component for seeding.
func contractSecretsFixture(t *testing.T) (http.Handler, *projects.Projects, *auth.Auth, *db.DB) {
	t.Helper()
	d := inMemoryDB(t)
	a := auth.New(auth.Options{DB: d, Logger: testLogger{}, Secret: "test-secret-32-chars!!!"})
	s, err := secrets.New(secrets.Options{DB: d, Logger: testLogger{}, RootKey: contractRootKey(t)})
	if err != nil {
		t.Fatalf("new secrets: %v", err)
	}
	p := projects.New(projects.Options{DB: d, Logger: testLogger{}})
	k := kernel.New(testLogger{})
	k.Register(d)
	k.Register(p)
	k.Register(s)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	sec, err := api.NewSecretsAPI(api.SecretsAPIOptions{Manager: s, DB: d, Logger: testLogger{}})
	if err != nil {
		t.Fatalf("new secrets api: %v", err)
	}
	return sec.Handler(), p, a, d
}

// contractSeedProject creates an org owned by ownerID with one project and
// production environment, and returns the caller context plus ids.
func contractSeedProject(t *testing.T, a *auth.Auth, p *projects.Projects, orgName string, ownerID int64) (context.Context, string, string) {
	t.Helper()
	org, err := a.CreateOrg(context.Background(), orgName, orgName, ownerID)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	ctx := auth.WithOrgID(auth.WithUserID(context.Background(), ownerID), org.ID)
	proj, err := p.CreateProject(ctx, "Payments")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	env, err := p.CreateEnvironment(ctx, proj.ID, "production", "Production")
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	return ctx, proj.ID, env.ID
}

func contractSecretURL(projectID, envID, suffix string) string {
	base := "/api/projects/" + projectID + "/environments/" + envID + "/secrets"
	return base + suffix
}

func contractDo(t *testing.T, h http.Handler, method, url string, body any, ctx context.Context) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *strings.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rdr = strings.NewReader(string(b))
	} else {
		rdr = strings.NewReader("")
	}
	req := httptest.NewRequest(method, url, rdr)
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestSecretContractListAndCreateMasked(t *testing.T) {
	h, p, a, _ := contractSecretsFixture(t)
	ctx, projectID, envID := contractSeedProject(t, a, p, "acme", 42)

	// Create returns metadata with a masked preview, never the plaintext.
	w := contractDo(t, h, "POST", contractSecretURL(projectID, envID, ""),
		map[string]string{"key": "TOKEN", "value": "top-secret-value"}, ctx)
	assertStatus(t, w.Code, http.StatusCreated)
	if strings.Contains(w.Body.String(), "top-secret-value") {
		t.Fatalf("create response leaked plaintext: %s", w.Body.String())
	}

	// List returns masked previews only.
	w = contractDo(t, h, "GET", contractSecretURL(projectID, envID, ""), nil, ctx)
	assertStatus(t, w.Code, http.StatusOK)
	if strings.Contains(w.Body.String(), "top-secret-value") {
		t.Fatalf("list response leaked plaintext: %s", w.Body.String())
	}
	var list map[string]any
	decodeJSON(t, w.Result(), &list)
	assertField(t, list, "data", "array")
	items := list["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(items))
	}
	item := items[0].(map[string]any)
	if _, has := item["value"]; has {
		t.Fatalf("list item must not carry a value field: %#v", item)
	}
	assertField(t, item, "value_preview", "string")
}

func TestSecretContractValueReadRequiresPermission(t *testing.T) {
	h, p, a, _ := contractSecretsFixture(t)
	ctx, projectID, envID := contractSeedProject(t, a, p, "acme", 42)
	w := contractDo(t, h, "POST", contractSecretURL(projectID, envID, ""),
		map[string]string{"key": "TOKEN", "value": "s3cr3t"}, ctx)
	assertStatus(t, w.Code, http.StatusCreated)

	// A describe-only project member is denied the explicit value read.
	memberCtx := auth.WithSecretProjectRole(ctx, auth.RoleProjectMember)
	w = contractDo(t, h, "GET", contractSecretURL(projectID, envID, "/TOKEN/value"), nil, memberCtx)
	assertStatus(t, w.Code, http.StatusForbidden)
	if strings.Contains(w.Body.String(), "s3cr3t") {
		t.Fatalf("denial leaked plaintext: %s", w.Body.String())
	}

	// The same member can still list versions (matrix: member versions yes).
	w = contractDo(t, h, "GET", contractSecretURL(projectID, envID, "/TOKEN/versions"), nil, memberCtx)
	assertStatus(t, w.Code, http.StatusOK)

	// The org admin reads the value through the explicit route.
	w = contractDo(t, h, "GET", contractSecretURL(projectID, envID, "/TOKEN/value"), nil, ctx)
	assertStatus(t, w.Code, http.StatusOK)
	var val map[string]any
	decodeJSON(t, w.Result(), &val)
	data := val["data"].(map[string]any)
	if data["value"] != "s3cr3t" {
		t.Fatalf("expected plaintext value, got %#v", data["value"])
	}
}

func TestSecretContractCrossOrgNonDisclosing(t *testing.T) {
	h, p, a, _ := contractSecretsFixture(t)
	ctxA, projectA, envA := contractSeedProject(t, a, p, "acme", 42)
	ctxB, _, _ := contractSeedProject(t, a, p, "globex", 43)

	w := contractDo(t, h, "POST", contractSecretURL(projectA, envA, ""),
		map[string]string{"key": "TOKEN", "value": "acme-value"}, ctxA)
	assertStatus(t, w.Code, http.StatusCreated)

	// Every route from org B must be a non-disclosing 404.
	for _, url := range []string{
		contractSecretURL(projectA, envA, ""),
		contractSecretURL(projectA, envA, "/TOKEN"),
		contractSecretURL(projectA, envA, "/TOKEN/versions"),
		contractSecretURL(projectA, envA, "/TOKEN/value"),
	} {
		w := contractDo(t, h, "GET", url, nil, ctxB)
		assertStatus(t, w.Code, http.StatusNotFound)
		if strings.Contains(w.Body.String(), "acme-value") || strings.Contains(w.Body.String(), "TOKEN") {
			t.Fatalf("cross-org response disclosed existence: %s", w.Body.String())
		}
	}
	// Mutation attempts from org B are also non-disclosing 404.
	w = contractDo(t, h, "POST", contractSecretURL(projectA, envA, ""),
		map[string]string{"key": "OTHER", "value": "x"}, ctxB)
	assertStatus(t, w.Code, http.StatusNotFound)
}

func TestSecretContractBoundsAndSQLSafety(t *testing.T) {
	h, p, a, d := contractSecretsFixture(t)
	ctx, projectID, envID := contractSeedProject(t, a, p, "acme", 42)

	// An injection-shaped key is invalid input (rejected before any SQL
	// execution) and cannot alter schema or rows.
	sqliKey := `TOKEN"; DROP TABLE secrets; --`
	w := contractDo(t, h, "POST", contractSecretURL(projectID, envID, ""),
		map[string]string{"key": sqliKey, "value": "x"}, ctx)
	assertStatus(t, w.Code, http.StatusBadRequest)

	// The secrets table still works after the rejected payload.
	w = contractDo(t, h, "POST", contractSecretURL(projectID, envID, ""),
		map[string]string{"key": "TOKEN", "value": "ok"}, ctx)
	assertStatus(t, w.Code, http.StatusCreated)
	var tableCount int
	if err := d.QueryRow("SELECT COUNT(*) FROM secrets").Scan(&tableCount); err != nil {
		t.Fatalf("secrets table unusable after injection attempt: %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("expected exactly 1 secret row, got %d", tableCount)
	}

	// Oversized key (>128 bytes) and oversized value (>64 KiB) are rejected.
	bigKey := strings.Repeat("K", 129)
	w = contractDo(t, h, "POST", contractSecretURL(projectID, envID, ""),
		map[string]string{"key": bigKey, "value": "v"}, ctx)
	assertStatus(t, w.Code, http.StatusBadRequest)
	bigValue := strings.Repeat("v", 64*1024+1)
	w = contractDo(t, h, "POST", contractSecretURL(projectID, envID, ""),
		map[string]string{"key": "BIG", "value": bigValue}, ctx)
	assertStatus(t, w.Code, http.StatusBadRequest)

	// Duplicate create is a 409, not a silent overwrite.
	w = contractDo(t, h, "POST", contractSecretURL(projectID, envID, ""),
		map[string]string{"key": "TOKEN", "value": "again"}, ctx)
	assertStatus(t, w.Code, http.StatusConflict)
}

func TestSecretContractMutationRateLimit(t *testing.T) {
	d := inMemoryDB(t)
	a := auth.New(auth.Options{DB: d, Logger: testLogger{}, Secret: "test-secret-32-chars!!!"})
	s, err := secrets.New(secrets.Options{DB: d, Logger: testLogger{}, RootKey: contractRootKey(t)})
	if err != nil {
		t.Fatalf("new secrets: %v", err)
	}
	p := projects.New(projects.Options{DB: d, Logger: testLogger{}})
	k := kernel.New(testLogger{})
	k.Register(d)
	k.Register(p)
	k.Register(s)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	sec, err := api.NewSecretsAPI(api.SecretsAPIOptions{
		Manager: s, DB: d, Logger: testLogger{},
		MutationLimit: 2, MutationWindow: time.Minute,
	})
	if err != nil {
		t.Fatalf("new secrets api: %v", err)
	}
	h := sec.Handler()
	ctx, projectID, envID := contractSeedProject(t, a, p, "acme", 42)

	for _, key := range []string{"ONE", "TWO"} {
		w := contractDo(t, h, "POST", contractSecretURL(projectID, envID, ""),
			map[string]string{"key": key, "value": "v"}, ctx)
		assertStatus(t, w.Code, http.StatusCreated)
	}
	w := contractDo(t, h, "POST", contractSecretURL(projectID, envID, ""),
		map[string]string{"key": "THREE", "value": "v"}, ctx)
	assertStatus(t, w.Code, http.StatusTooManyRequests)
	if w.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header on 429")
	}
}

func TestSecretContractAuditRowsValueFree(t *testing.T) {
	h, p, a, d := contractSecretsFixture(t)
	ctx, projectID, envID := contractSeedProject(t, a, p, "acme", 42)

	w := contractDo(t, h, "POST", contractSecretURL(projectID, envID, ""),
		map[string]string{"key": "TOKEN", "value": "contract-plaintext"}, ctx)
	assertStatus(t, w.Code, http.StatusCreated)

	deadline := time.Now().Add(2 * time.Second)
	for {
		rows, err := d.Query(`SELECT event_type, metadata FROM audit_events WHERE event_type = 'secret.created'`)
		if err != nil {
			t.Fatalf("query audit: %v", err)
		}
		found := false
		for rows.Next() {
			var eventType string
			var meta sql.NullString
			if err := rows.Scan(&eventType, &meta); err != nil {
				_ = rows.Close()
				t.Fatalf("scan audit: %v", err)
			}
			found = true
			if strings.Contains(meta.String, "contract-plaintext") {
				_ = rows.Close()
				t.Fatalf("audit row leaked plaintext: %s", meta.String)
			}
			if strings.Contains(meta.String, "ciphertext") || strings.Contains(meta.String, "nonce") {
				_ = rows.Close()
				t.Fatalf("audit row leaked storage internals: %s", meta.String)
			}
			var parsed map[string]any
			if err := json.Unmarshal([]byte(meta.String), &parsed); err != nil {
				_ = rows.Close()
				t.Fatalf("audit metadata not json: %v", err)
			}
			for _, field := range []string{"actor", "org_id", "project_id", "environment_id", "secret", "action", "request_id"} {
				if _, ok := parsed[field]; !ok {
					_ = rows.Close()
					t.Fatalf("audit metadata missing %q: %s", field, meta.String)
				}
			}
		}
		_ = rows.Close()
		if found {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("audit row never appeared")
		}
		time.Sleep(25 * time.Millisecond)
	}
}
