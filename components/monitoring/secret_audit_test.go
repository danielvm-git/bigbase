package monitoring_test

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

// secret_audit_test.go is the SC-e89s04-P0-04 regression guard: secret
// create/read/update operations emit audit records carrying actor and scope
// metadata, and no audit row ever contains plaintext or ciphertext. It runs
// the full REST adapter stack against in-memory SQLite because the audit
// contract is only observable end to end.

func auditRootKey(t *testing.T) []byte {
	t.Helper()
	raw := base64.StdEncoding.EncodeToString([]byte("fedcba9876543210fedcba9876543210"))
	key, err := secrets.ParseRootKey(raw)
	if err != nil {
		t.Fatalf("parse root key: %v", err)
	}
	return key
}

// auditSecretFixture boots db + projects + secrets + auth (auth owns the
// orgs/org_members/audit_events tables) and returns the REST adapter handler.
func auditSecretFixture(t *testing.T) (http.Handler, *projects.Projects, *auth.Auth, *db.DB) {
	t.Helper()
	d := db.New(db.Options{Path: ":memory:", Logger: testLogger{}})
	a := auth.New(auth.Options{DB: d, Logger: testLogger{}, Secret: "test-secret-32-chars!!!"})
	s, err := secrets.New(secrets.Options{DB: d, Logger: testLogger{}, RootKey: auditRootKey(t)})
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
		t.Fatalf("start kernel: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	sec, err := api.NewSecretsAPI(api.SecretsAPIOptions{Manager: s, DB: d, Logger: testLogger{}})
	if err != nil {
		t.Fatalf("new secrets api: %v", err)
	}
	return sec.Handler(), p, a, d
}

func TestSecretAuditContainsActorAndScopeButNoValues(t *testing.T) {
	h, p, a, d := auditSecretFixture(t)

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

	const (
		createValue = "audit-plaintext-value"
		updateValue = "audit-updated-value"
	)
	base := "/api/projects/" + proj.ID + "/environments/" + env.ID + "/secrets"
	post := func(path string, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, base+path, strings.NewReader(body)).WithContext(ctx)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	// Create.
	if w := post("", `{"key":"TOKEN","value":"`+createValue+`"}`); w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	// Read value.
	req := httptest.NewRequest(http.MethodGet, base+"/TOKEN/value", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("value read: %d %s", w.Code, w.Body.String())
	}
	// Update.
	req = httptest.NewRequest(http.MethodPut, base+"/TOKEN",
		strings.NewReader(`{"value":"`+updateValue+`"}`)).WithContext(ctx)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}

	// Audit writes are fire-and-forget; poll for all three events.
	deadline := time.Now().Add(2 * time.Second)
	var seen map[string]bool
	for time.Now().Before(deadline) {
		seen = querySecretAuditRows(t, d, createValue, updateValue)
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

// querySecretAuditRows returns the set of secret.* event types and asserts
// every row carries actor/scope metadata and never the given plaintexts.
func querySecretAuditRows(t *testing.T, d *db.DB, plaintexts ...string) map[string]bool {
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
		for _, p := range plaintexts {
			if strings.Contains(meta.String, p) {
				t.Fatalf("audit row %s leaked plaintext %q: %s", eventType, p, meta.String)
			}
		}
		if strings.Contains(meta.String, "ciphertext") || strings.Contains(meta.String, "nonce") {
			t.Fatalf("audit row %s leaked storage internals: %s", eventType, meta.String)
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
		if parsed["secret"] != "TOKEN" {
			t.Fatalf("audit secret reference wrong: %s", meta.String)
		}
		if parsed["actor"] != "user:42" {
			t.Fatalf("audit actor wrong: %s", meta.String)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate audit: %v", err)
	}
	return seen
}
