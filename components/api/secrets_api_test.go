package api_test

import (
	"bytes"
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

// secrets_api_test.go exercises the canonical secret REST surface end to end
// against in-memory SQLite: masked metadata responses, explicit value reads,
// role/action authorization, non-disclosing cross-org errors, bounds, rate
// limits, version listing, and value-free audit events.

// testRootKey is a canonical base64 32-byte root key shared by the fixtures.
func testRootKey(t *testing.T) []byte {
	t.Helper()
	raw := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, 32))
	key, err := secrets.ParseRootKey(raw)
	if err != nil {
		t.Fatalf("parse root key: %v", err)
	}
	return key
}

// secretsAPIFixture boots db, projects, secrets, and auth (for the orgs,
// org_members, and audit_events tables) and returns the mounted adapter
// handler.
func secretsAPIFixture(t *testing.T) (http.Handler, *projects.Projects, *auth.Auth, *db.DB) {
	t.Helper()
	logger := testLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	s, err := secrets.New(secrets.Options{DB: d, Logger: logger, RootKey: testRootKey(t)})
	if err != nil {
		t.Fatalf("new secrets: %v", err)
	}
	p := projects.New(projects.Options{DB: d, Logger: logger})
	k := kernel.New(logger)
	k.Register(d)
	k.Register(p)
	k.Register(s)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("start kernel: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	sec, err := api.NewSecretsAPI(api.SecretsAPIOptions{Manager: s, DB: d, Logger: logger})
	if err != nil {
		t.Fatalf("new secrets api: %v", err)
	}
	return sec.Handler(), p, a, d
}

// seedProject creates an org owned by ownerID, plus a project and its
// production environment.
func seedProject(t *testing.T, a *auth.Auth, p *projects.Projects, orgName string, ownerID int64) (orgID int64, projectID, envID string) {
	t.Helper()
	org, err := a.CreateOrg(context.Background(), orgName, orgName, ownerID)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	ctx := auth.WithOrgID(context.Background(), org.ID)
	proj, err := p.CreateProject(ctx, "Payments")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	env, err := p.CreateEnvironment(ctx, proj.ID, "production", "Production")
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	return org.ID, proj.ID, env.ID
}

func userCtx(orgID, userID int64) context.Context {
	return auth.WithOrgID(auth.WithUserID(context.Background(), userID), orgID)
}

func roleCtx(orgID, userID int64, role auth.ProjectRole) context.Context {
	return auth.WithSecretProjectRole(userCtx(orgID, userID), role)
}

func secretURL(projectID, envID, suffix string) string {
	base := "/api/projects/" + projectID + "/environments/" + envID + "/secrets"
	if suffix == "" {
		return base
	}
	return base + suffix
}

// createSecretAs creates one secret through the API and asserts success.
func createSecretAs(t *testing.T, h http.Handler, projectID, envID, key, value string, ctx context.Context) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, secretURL(projectID, envID, ""),
		bytes.NewReader(mustJSON(t, map[string]string{"key": key, "value": value})))
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %s: status %d body %s", key, rec.Code, rec.Body.String())
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// responseData decodes a {"data": ...} response.
func responseData(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode response %q: %v", string(body), err)
	}
	return out
}

func TestSecretAPIListMetadataOnly(t *testing.T) {
	h, p, a, _ := secretsAPIFixture(t)
	orgID, projectID, envID := seedProject(t, a, p, "acme", 42)
	ctx := userCtx(orgID, 42)
	createSecretAs(t, h, projectID, envID, "TOKEN", "super-secret-value", ctx)
	createSecretAs(t, h, projectID, envID, "API_KEY", "short", ctx)

	req := httptest.NewRequest(http.MethodGet, secretURL(projectID, envID, ""), nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status %d body %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "super-secret-value") || strings.Contains(body, "short") {
		t.Fatalf("list response leaked plaintext: %s", body)
	}
	if strings.Contains(body, "ciphertext") || strings.Contains(body, "nonce") {
		t.Fatalf("list response leaked storage internals: %s", body)
	}
	var list map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	items, ok := list["data"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 secrets, got %#v", list["data"])
	}
	for _, it := range items {
		item := it.(map[string]any)
		for _, field := range []string{"id", "project_id", "environment_id", "folder_id", "key", "value_preview", "current_version", "created_at", "updated_at"} {
			if _, ok := item[field]; !ok {
				t.Fatalf("list item missing %q: %#v", field, item)
			}
		}
		if _, has := item["value"]; has {
			t.Fatalf("list item must not carry a value field: %#v", item)
		}
		if prev, _ := item["value_preview"].(string); prev == "" {
			t.Fatalf("expected a masked preview, got %#v", item)
		}
	}
}

func TestSecretAPICreateReturnsMaskedResponse(t *testing.T) {
	h, p, a, _ := secretsAPIFixture(t)
	orgID, projectID, envID := seedProject(t, a, p, "acme", 42)
	ctx := userCtx(orgID, 42)

	req := httptest.NewRequest(http.MethodPost, secretURL(projectID, envID, ""),
		bytes.NewReader(mustJSON(t, map[string]string{"key": "TOKEN", "value": "plaintext-value"}))).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status %d body %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "plaintext-value") {
		t.Fatalf("create response leaked plaintext: %s", body)
	}
	data := responseData(t, w.Body.Bytes())
	item := data["data"].(map[string]any)
	if item["key"] != "TOKEN" {
		t.Fatalf("unexpected key %#v", item["key"])
	}
	if _, has := item["value"]; has {
		t.Fatalf("create response must not carry a value field")
	}
}

func TestSecretAPIValueReadRequiresPermission(t *testing.T) {
	h, p, a, _ := secretsAPIFixture(t)
	orgID, projectID, envID := seedProject(t, a, p, "acme", 42)
	owner := userCtx(orgID, 42)
	createSecretAs(t, h, projectID, envID, "TOKEN", "s3cr3t", owner)

	// A project member may list (describe) but must not read values.
	member := roleCtx(orgID, 43, auth.RoleProjectMember)
	req := httptest.NewRequest(http.MethodGet, secretURL(projectID, envID, "/TOKEN/value"), nil).WithContext(member)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("member value read: expected 403, got %d body %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "s3cr3t") {
		t.Fatalf("denial response leaked plaintext: %s", w.Body.String())
	}

	// Describe still works for the member.
	req = httptest.NewRequest(http.MethodGet, secretURL(projectID, envID, ""), nil).WithContext(member)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("member list: expected 200, got %d", w.Code)
	}
}

func TestSecretAPIValueReadSucceedsForOperator(t *testing.T) {
	h, p, a, _ := secretsAPIFixture(t)
	orgID, projectID, envID := seedProject(t, a, p, "acme", 42)
	owner := userCtx(orgID, 42)
	createSecretAs(t, h, projectID, envID, "TOKEN", "s3cr3t-value", owner)

	// org_admin (owner) and project_operator may read the explicit value.
	for _, ctx := range []context.Context{owner, roleCtx(orgID, 43, auth.RoleProjectOperator)} {
		req := httptest.NewRequest(http.MethodGet, secretURL(projectID, envID, "/TOKEN/value"), nil).WithContext(ctx)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("value read: expected 200, got %d body %s", w.Code, w.Body.String())
		}
		data := responseData(t, w.Body.Bytes())
		val := data["data"].(map[string]any)
		if val["value"] != "s3cr3t-value" {
			t.Fatalf("expected plaintext value, got %#v", val["value"])
		}
	}
}

func TestSecretAPICrossOrgNonDisclosing(t *testing.T) {
	h, p, a, _ := secretsAPIFixture(t)
	orgA, projectA, envA := seedProject(t, a, p, "acme", 42)
	orgB, _, _ := seedProject(t, a, p, "globex", 43)
	ownerA := userCtx(orgA, 42)
	createSecretAs(t, h, projectA, envA, "TOKEN", "acme-value", ownerA)

	ctxB := userCtx(orgB, 43)
	for _, path := range []string{
		secretURL(projectA, envA, ""),
		secretURL(projectA, envA, "/TOKEN"),
		secretURL(projectA, envA, "/TOKEN/value"),
		secretURL(projectA, envA, "/TOKEN/versions"),
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil).WithContext(ctxB)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("cross-org %s: expected 404, got %d body %s", path, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "acme-value") || strings.Contains(w.Body.String(), "TOKEN") {
			t.Fatalf("cross-org response disclosed existence: %s", w.Body.String())
		}
	}
}

func TestSecretAPIUnauthenticatedReturns401(t *testing.T) {
	h, p, a, _ := secretsAPIFixture(t)
	_, projectID, envID := seedProject(t, a, p, "acme", 42)

	req := httptest.NewRequest(http.MethodGet, secretURL(projectID, envID, ""), nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: expected 401, got %d body %s", w.Code, w.Body.String())
	}
}

func TestSecretAPIInvalidInputRejected(t *testing.T) {
	h, p, a, _ := secretsAPIFixture(t)
	orgID, projectID, envID := seedProject(t, a, p, "acme", 42)
	ctx := userCtx(orgID, 42)

	badKeys := []string{"", "lowercase", "has space", "DOT.KEY", strings.Repeat("K", 129)}
	for _, key := range badKeys {
		req := httptest.NewRequest(http.MethodPost, secretURL(projectID, envID, ""),
			bytes.NewReader(mustJSON(t, map[string]string{"key": key, "value": "v"}))).WithContext(ctx)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("bad key %q: expected 400, got %d body %s", key, w.Code, w.Body.String())
		}
	}

	// Oversized value (64 KiB + 1).
	big := strings.Repeat("v", 64*1024+1)
	req := httptest.NewRequest(http.MethodPost, secretURL(projectID, envID, ""),
		bytes.NewReader(mustJSON(t, map[string]string{"key": "TOKEN", "value": big}))).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("oversized value: expected 400, got %d body %s", w.Code, w.Body.String())
	}

	// Malformed JSON.
	req = httptest.NewRequest(http.MethodPost, secretURL(projectID, envID, ""),
		strings.NewReader(`{"key": "TOKEN", "value": `)).WithContext(ctx)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("malformed json: expected 400, got %d", w.Code)
	}
}

func TestSecretAPIVersionsBoundedAndOrdered(t *testing.T) {
	h, p, a, _ := secretsAPIFixture(t)
	orgID, projectID, envID := seedProject(t, a, p, "acme", 42)
	owner := userCtx(orgID, 42)
	createSecretAs(t, h, projectID, envID, "TOKEN", "v1", owner)
	req := httptest.NewRequest(http.MethodPut, secretURL(projectID, envID, "/TOKEN"),
		bytes.NewReader(mustJSON(t, map[string]string{"value": "v2"}))).WithContext(owner)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d body %s", w.Code, w.Body.String())
	}

	member := roleCtx(orgID, 43, auth.RoleProjectMember)
	req = httptest.NewRequest(http.MethodGet, secretURL(projectID, envID, "/TOKEN/versions"), nil).WithContext(member)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("member versions: expected 200, got %d body %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "v1") || strings.Contains(body, "v2") {
		t.Fatalf("versions response leaked values: %s", body)
	}
	data := responseData(t, w.Body.Bytes())
	items, ok := data["data"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 versions, got %#v", data["data"])
	}
	for i, it := range items {
		ver := it.(map[string]any)
		if ver["version"] != float64(i+1) {
			t.Fatalf("expected ascending version %d, got %#v", i+1, ver)
		}
		if _, has := ver["value"]; has {
			t.Fatalf("version item must not carry a value: %#v", ver)
		}
	}
}

func TestSecretAPIDeleteSecret(t *testing.T) {
	h, p, a, _ := secretsAPIFixture(t)
	orgID, projectID, envID := seedProject(t, a, p, "acme", 42)
	ctx := userCtx(orgID, 42)
	createSecretAs(t, h, projectID, envID, "TOKEN", "v1", ctx)

	req := httptest.NewRequest(http.MethodDelete, secretURL(projectID, envID, "/TOKEN"), nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d body %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, secretURL(projectID, envID, "/TOKEN"), nil).WithContext(ctx)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d", w.Code)
	}
}

func TestSecretAPIMutationRateLimit(t *testing.T) {
	logger := testLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	s, err := secrets.New(secrets.Options{DB: d, Logger: logger, RootKey: testRootKey(t)})
	if err != nil {
		t.Fatalf("new secrets: %v", err)
	}
	p := projects.New(projects.Options{DB: d, Logger: logger})
	k := kernel.New(logger)
	k.Register(d)
	k.Register(p)
	k.Register(s)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("start kernel: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	// A tiny budget (2 mutations per minute) makes the limit observable.
	sec, err := api.NewSecretsAPI(api.SecretsAPIOptions{
		Manager: s, DB: d, Logger: logger,
		MutationLimit: 2, MutationWindow: time.Minute,
	})
	if err != nil {
		t.Fatalf("new secrets api: %v", err)
	}
	h := sec.Handler()
	org, err := a.CreateOrg(context.Background(), "acme", "acme", 42)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	ctx := auth.WithOrgID(auth.WithUserID(context.Background(), 42), org.ID)
	proj, err := p.CreateProject(ctx, "Payments")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	env, err := p.CreateEnvironment(ctx, proj.ID, "production", "Production")
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}

	// Two mutations within budget.
	for _, key := range []string{"ONE", "TWO"} {
		req := httptest.NewRequest(http.MethodPost, secretURL(proj.ID, env.ID, ""),
			bytes.NewReader(mustJSON(t, map[string]string{"key": key, "value": "v"}))).WithContext(ctx)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s: expected 201, got %d body %s", key, w.Code, w.Body.String())
		}
	}

	// Third mutation is rate-limited before persistence.
	req := httptest.NewRequest(http.MethodPost, secretURL(proj.ID, env.ID, ""),
		bytes.NewReader(mustJSON(t, map[string]string{"key": "THREE", "value": "v"}))).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit: expected 429, got %d body %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header on 429")
	}

	// A different project has its own budget (per-actor+project contract).
	proj2, err := p.CreateProject(ctx, "Orders")
	if err != nil {
		t.Fatalf("create project 2: %v", err)
	}
	env2, err := p.CreateEnvironment(ctx, proj2.ID, "production", "Production")
	if err != nil {
		t.Fatalf("create environment 2: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, secretURL(proj2.ID, env2.ID, ""),
		bytes.NewReader(mustJSON(t, map[string]string{"key": "ONE", "value": "v"}))).WithContext(ctx)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("other project mutation: expected 201, got %d body %s", w.Code, w.Body.String())
	}
}

func TestSecretAPIAuditContainsNoValues(t *testing.T) {
	h, p, a, d := secretsAPIFixture(t)
	orgID, projectID, envID := seedProject(t, a, p, "acme", 42)
	ctx := userCtx(orgID, 42)

	createSecretAs(t, h, projectID, envID, "TOKEN", "plaintext-token", ctx)
	req := httptest.NewRequest(http.MethodGet, secretURL(projectID, envID, "/TOKEN/value"), nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("value read: %d %s", w.Code, w.Body.String())
	}
	req = httptest.NewRequest(http.MethodPut, secretURL(projectID, envID, "/TOKEN"),
		bytes.NewReader(mustJSON(t, map[string]string{"value": "updated-token"}))).WithContext(ctx)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}

	// Audit writes are fire-and-forget; poll briefly for the expected events.
	deadline := time.Now().Add(2 * time.Second)
	var seen map[string]bool
	for time.Now().Before(deadline) {
		seen = querySecretAuditEvents(t, d)
		if seen["secret.created"] && seen["secret.value_read"] && seen["secret.updated"] {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	for _, want := range []string{"secret.created", "secret.value_read", "secret.updated"} {
		if !seen[want] {
			t.Fatalf("missing audit event %q; saw %v", want, seen)
		}
	}
}

// querySecretAuditEvents returns the set of secret.* event types present in
// the audit table and asserts every row is value-free.
func querySecretAuditEvents(t *testing.T, d *db.DB) map[string]bool {
	t.Helper()
	rows, err := d.Query(`SELECT event_type, metadata FROM audit_events WHERE event_type LIKE 'secret.%' ORDER BY id`)
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	defer func() { _ = rows.Close() }()
	seen := map[string]bool{}
	for rows.Next() {
		var eventType string
		var meta sql.NullString
		if err := rows.Scan(&eventType, &meta); err != nil {
			t.Fatalf("scan audit: %v", err)
		}
		seen[eventType] = true
		if !meta.Valid {
			t.Fatalf("audit row %s has no metadata", eventType)
		}
		if strings.Contains(meta.String, "plaintext-token") || strings.Contains(meta.String, "updated-token") {
			t.Fatalf("audit row leaked plaintext: %s", meta.String)
		}
		if strings.Contains(meta.String, "ciphertext") || strings.Contains(meta.String, "nonce") {
			t.Fatalf("audit row leaked storage internals: %s", meta.String)
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(meta.String), &parsed); err != nil {
			t.Fatalf("audit metadata not json: %v", err)
		}
		for _, field := range []string{"actor", "org_id", "project_id", "environment_id", "secret", "action", "request_id"} {
			if _, ok := parsed[field]; !ok {
				t.Fatalf("audit metadata missing %q: %s", field, meta.String)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate audit: %v", err)
	}
	return seen
}

func TestSecretAPIUnknownRouteMethod(t *testing.T) {
	h, p, a, _ := secretsAPIFixture(t)
	orgID, projectID, envID := seedProject(t, a, p, "acme", 42)
	ctx := userCtx(orgID, 42)

	// PATCH is not part of the frozen contract.
	req := httptest.NewRequest(http.MethodPatch, secretURL(projectID, envID, "/TOKEN"), nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("patch: expected 405, got %d", w.Code)
	}
}
