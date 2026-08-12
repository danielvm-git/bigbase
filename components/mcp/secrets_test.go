// story: e89s07
package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/danielvm/bigbase/components/mcp"
)

// secrets_test.go covers SC-e89s07-P0-01..05 and SC-e89s07-P1-06 using real
// MCP HTTP sessions (Bearer org keys and Site deploy keys) and stdio sessions
// (in-memory transports carrying an authenticated principal in context).

// --- Test fixtures ----------------------------------------------------------

type mockOrgKey struct {
	orgID  int64
	scopes []string
}

type mockSiteKey struct {
	siteID string
	scopes []string
}

// mockKeyAuth resolves both bb_ org keys and bb_dep_ Site keys, mirroring the
// composition-root auth adapter contract.
type mockKeyAuth struct {
	orgKeys  map[string]mockOrgKey
	siteKeys map[string]mockSiteKey
}

func (m *mockKeyAuth) ResolveOrgKey(token string) (int64, []string, error) {
	if k, ok := m.orgKeys[token]; ok {
		return k.orgID, k.scopes, nil
	}
	return 0, nil, errors.New("api key not found")
}

func (m *mockKeyAuth) ResolveSiteKeyScopes(token string) (string, []string, error) {
	if k, ok := m.siteKeys[token]; ok {
		return k.siteID, k.scopes, nil
	}
	return "", nil, errors.New("site key not found")
}

// mockProjectTargetAuthorizer enforces org ownership for org principals and
// Site→Project binding for Site principals.
type mockProjectTargetAuthorizer struct {
	projectOrg  map[string]int64
	siteProject map[string]string
	called      int
}

func (m *mockProjectTargetAuthorizer) AuthorizeProjectTarget(_ context.Context, projectID string, p mcp.Principal) error {
	m.called++
	switch p.Kind {
	case mcp.PrincipalOrg:
		if m.projectOrg[projectID] != p.OrgID {
			return errors.New("denied")
		}
		return nil
	case mcp.PrincipalSite:
		if m.siteProject[p.SiteID] != projectID {
			return errors.New("denied")
		}
		return nil
	default:
		return errors.New("denied")
	}
}

type mockSecretEntry struct {
	meta  mcp.SecretMetadata
	value string
}

// mockSecretManager is an in-memory ProjectSecretManager. It emits the MCP
// sentinels a composition-root adapter would, so tests exercise the tool layer
// contract rather than storage.
type mockSecretManager struct {
	entries map[string]map[string]mockSecretEntry // folderID -> key -> entry
	nextErr error                                 // injected failure (SC-e89s07-P0-05)
	version int
}

func newMockSecretManager() *mockSecretManager {
	return &mockSecretManager{entries: make(map[string]map[string]mockSecretEntry)}
}

func (m *mockSecretManager) EnsureFolder(_ context.Context, _ mcp.Principal, projectID, environmentID, _ string) (string, error) {
	if m.nextErr != nil {
		return "", m.nextErr
	}
	return "folder-" + projectID + "-" + environmentID, nil
}

func (m *mockSecretManager) ListSecrets(_ context.Context, _ mcp.Principal, projectID, environmentID, folderID string) ([]mcp.SecretMetadata, error) {
	if m.nextErr != nil {
		return nil, m.nextErr
	}
	keys := make([]string, 0, len(m.entries[folderID]))
	for k := range m.entries[folderID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]mcp.SecretMetadata, 0, len(keys))
	for _, k := range keys {
		out = append(out, m.entries[folderID][k].meta)
	}
	return out, nil
}

func (m *mockSecretManager) GetSecretMetadata(_ context.Context, _ mcp.Principal, projectID, environmentID, folderID, key string) (mcp.SecretMetadata, error) {
	if m.nextErr != nil {
		return mcp.SecretMetadata{}, m.nextErr
	}
	e, ok := m.entries[folderID][key]
	if !ok {
		return mcp.SecretMetadata{}, mcp.ErrSecretNotFound
	}
	return e.meta, nil
}

func (m *mockSecretManager) ReadSecretValue(_ context.Context, _ mcp.Principal, projectID, environmentID, folderID, key string) (mcp.SecretValue, error) {
	if m.nextErr != nil {
		return mcp.SecretValue{}, m.nextErr
	}
	e, ok := m.entries[folderID][key]
	if !ok {
		return mcp.SecretValue{}, mcp.ErrSecretNotFound
	}
	return mcp.SecretValue{
		SecretID:  e.meta.ID,
		Key:       key,
		Version:   e.meta.CurrentVersion,
		Value:     e.value,
		KeyID:     "key-1",
		Algorithm: "aes-256-gcm",
	}, nil
}

func (m *mockSecretManager) CreateSecret(_ context.Context, _ mcp.Principal, projectID, environmentID, folderID, key, value string) (mcp.SecretMetadata, error) {
	if m.nextErr != nil {
		return mcp.SecretMetadata{}, m.nextErr
	}
	if _, ok := m.entries[folderID][key]; ok {
		return mcp.SecretMetadata{}, mcp.ErrSecretConflict
	}
	return m.put(projectID, environmentID, folderID, key, value), nil
}

func (m *mockSecretManager) UpdateSecret(_ context.Context, _ mcp.Principal, projectID, environmentID, folderID, key, value string) (mcp.SecretMetadata, error) {
	if m.nextErr != nil {
		return mcp.SecretMetadata{}, m.nextErr
	}
	if _, ok := m.entries[folderID][key]; !ok {
		return mcp.SecretMetadata{}, mcp.ErrSecretNotFound
	}
	return m.put(projectID, environmentID, folderID, key, value), nil
}

func (m *mockSecretManager) DeleteSecret(_ context.Context, _ mcp.Principal, projectID, environmentID, folderID, key string) error {
	if m.nextErr != nil {
		return m.nextErr
	}
	if _, ok := m.entries[folderID][key]; !ok {
		return mcp.ErrSecretNotFound
	}
	delete(m.entries[folderID], key)
	return nil
}

func (m *mockSecretManager) put(projectID, environmentID, folderID, key, value string) mcp.SecretMetadata {
	if m.entries[folderID] == nil {
		m.entries[folderID] = make(map[string]mockSecretEntry)
	}
	m.version++
	meta := mcp.SecretMetadata{
		ID:             "sec-" + key,
		ProjectID:      projectID,
		EnvironmentID:  environmentID,
		FolderID:       folderID,
		Key:            key,
		CurrentVersion: m.version,
		ValuePreview:   maskPreview(value),
		CreatedAt:      "2026-08-12T00:00:00Z",
		UpdatedAt:      "2026-08-12T00:00:00Z",
	}
	m.entries[folderID][key] = mockSecretEntry{meta: meta, value: value}
	return meta
}

// maskPreview mirrors the masked preview contract: never the full value.
func maskPreview(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return "••••"
	}
	return "••••" + v[len(v)-4:]
}

func newSecretTestComponent(t *testing.T) (*mcp.Component, *mockKeyAuth, *mockSecretManager, *mockProjectTargetAuthorizer, *mockEnvVarManager) {
	t.Helper()
	authMock := &mockKeyAuth{
		orgKeys: map[string]mockOrgKey{
			"bb_org1_read":        {orgID: 1, scopes: []string{mcp.ScopeSecretsRead}},
			"bb_org1_write":       {orgID: 1, scopes: []string{mcp.ScopeProvision, mcp.ScopeSecretsWrite}},
			"bb_org1_all":         {orgID: 1, scopes: []string{mcp.ScopeProvision, mcp.ScopeSecretsRead, mcp.ScopeSecretsWrite}},
			"bb_org1_noscopes":    {orgID: 1},
			"bb_org1_provision":   {orgID: 1, scopes: []string{mcp.ScopeProvision}},
			"bb_org1_writenoprov": {orgID: 1, scopes: []string{mcp.ScopeSecretsWrite}},
			"bb_org2_read":        {orgID: 2, scopes: []string{mcp.ScopeSecretsRead}},
		},
		siteKeys: map[string]mockSiteKey{
			"bb_dep_sitea": {siteID: "site-a", scopes: []string{"deploy"}},
			"bb_dep_siteb": {siteID: "site-b", scopes: []string{"deploy"}},
		},
	}
	secMock := newMockSecretManager()
	projAuth := &mockProjectTargetAuthorizer{
		projectOrg:  map[string]int64{"proj-1": 1, "proj-2": 2},
		siteProject: map[string]string{"site-a": "proj-1"},
	}
	envMock := newMockEnvVarManager()
	envMock.vars["site-a"] = []mcp.SiteEnvVar{
		{ID: "v1", SiteID: "site-a", Key: "TOKEN", ValuePreview: "••••oken", IsBuildTime: true, IsRuntime: true},
	}
	c := mcp.New(mcp.Options{
		Enabled:                 true,
		OrgKeyAuthenticator:     authMock,
		SiteKeyAuthenticator:    authMock,
		SiteEnvVarManager:       envMock,
		ProjectSecretManager:    secMock,
		ProjectTargetAuthorizer: projAuth,
	})
	return c, authMock, secMock, projAuth, envMock
}

// --- Transport helpers -------------------------------------------------------

type toolResult struct {
	status  int
	isError bool
	text    string
}

// httpToolCall drives a real MCP HTTP session: it initializes a Streamable
// HTTP session, then calls the tool with a Bearer token. The response is
// parsed from the JSON-RPC payload (plain JSON or SSE).
func httpToolCall(t *testing.T, handler http.Handler, token, tool string, args map[string]any) toolResult {
	t.Helper()

	// 1. Initialize the MCP session (Streamable HTTP requires it).
	initBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0"},
		},
	})
	initReq := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(initBody))
	initReq.Header.Set("Content-Type", "application/json")
	initReq.Header.Set("Accept", "application/json, text/event-stream")
	initW := httptest.NewRecorder()
	handler.ServeHTTP(initW, initReq)
	sessionID := initW.Header().Get("Mcp-Session-Id")

	// 2. Call the tool within the session.
	params, _ := json.Marshal(map[string]any{"name": tool, "arguments": args})
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params":  json.RawMessage(params),
	})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	out := toolResult{status: w.Code}
	payload := w.Body.Bytes()
	if strings.Contains(w.Header().Get("Content-Type"), "text/event-stream") {
		payload = sseData(payload)
	}
	var resp struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &resp); err == nil {
		if len(resp.Result.Content) > 0 {
			out.text = resp.Result.Content[0].Text
		}
		out.isError = resp.Result.IsError
	}
	return out
}

// sseData extracts the concatenated JSON payload from an SSE message body,
// falling back to the raw body when no data lines are present.
func sseData(body []byte) []byte {
	var data []byte
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "data: ") {
			data = append(data, []byte(strings.TrimPrefix(line, "data: "))...)
		}
	}
	if len(data) == 0 {
		return body
	}
	return data
}

// stdioToolCall drives an MCP session over in-memory transports with the given
// authenticated principal attached to the context, mirroring how a local
// operator process runs the stdio transport.
func stdioToolCall(t *testing.T, c *mcp.Component, p mcp.Principal, tool string, args map[string]any) toolResult {
	t.Helper()
	ctx := mcp.WithPrincipal(context.Background(), p)
	srv, err := c.NewMCPServer()
	if err != nil {
		t.Fatalf("NewMCPServer: %v", err)
	}
	t1, t2 := mcpsdk.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", tool, err)
	}
	out := toolResult{isError: result.IsError}
	if len(result.Content) > 0 {
		if tc, ok := result.Content[0].(*mcpsdk.TextContent); ok {
			out.text = tc.Text
		}
	}
	return out
}

// --- SC-e89s07-P0-01: MCP reads are organization-scoped ----------------------

func TestMCPSecretOrgScopingHTTP(t *testing.T) {
	c, _, _, projAuth, _ := newSecretTestComponent(t)
	handler := c.Handler()

	res := httpToolCall(t, handler, "bb_org1_read", "list_project_secrets", map[string]any{
		"project_id": "proj-2", "environment_id": "env-1",
	})
	if !res.isError {
		t.Fatalf("cross-org list must fail, got: %s", res.text)
	}
	if !strings.Contains(res.text, "project authorization denied") {
		t.Fatalf("expected non-disclosing denial, got %q", res.text)
	}
	if strings.Contains(res.text, "proj-2") || strings.Contains(res.text, "secrets") {
		t.Fatalf("cross-org denial leaked target metadata: %s", res.text)
	}
	if projAuth.called == 0 {
		t.Fatal("expected target authorization to run before any SecretManager call")
	}

	res = httpToolCall(t, handler, "bb_org1_read", "get_project_secret", map[string]any{
		"project_id": "proj-2", "environment_id": "env-1", "key": "API_KEY",
	})
	if !res.isError || !strings.Contains(res.text, "project authorization denied") {
		t.Fatalf("cross-org get must be denied, got %q (err=%v)", res.text, res.isError)
	}
}

func TestMCPSecretOrgScopingStdio(t *testing.T) {
	c, _, _, _, _ := newSecretTestComponent(t)
	org1 := mcp.Principal{Kind: mcp.PrincipalOrg, OrgID: 1, Scopes: []string{mcp.ScopeSecretsRead}}

	res := stdioToolCall(t, c, org1, "list_project_secrets", map[string]any{
		"project_id": "proj-2", "environment_id": "env-1",
	})
	if !res.isError || !strings.Contains(res.text, "project authorization denied") {
		t.Fatalf("cross-org stdio list must be denied, got %q (err=%v)", res.text, res.isError)
	}
	if strings.Contains(res.text, "proj-2") {
		t.Fatalf("cross-org stdio denial leaked target: %s", res.text)
	}

	// Same-org list succeeds and is masked.
	res = stdioToolCall(t, c, org1, "list_project_secrets", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1",
	})
	if res.isError {
		t.Fatalf("same-org stdio list failed: %s", res.text)
	}
	if !strings.Contains(res.text, `"count":0`) {
		t.Fatalf("expected empty masked list, got: %s", res.text)
	}
}

// --- SC-e89s07-P0-02: read/write scopes differ across transports -------------

func TestMCPSecretScopeMatrixHTTP(t *testing.T) {
	c, _, _, _, _ := newSecretTestComponent(t)
	handler := c.Handler()

	args := map[string]any{"project_id": "proj-1", "environment_id": "env-1"}

	// Read-only key: reads succeed, writes denied.
	if res := httpToolCall(t, handler, "bb_org1_read", "list_project_secrets", args); res.isError {
		t.Fatalf("secrets:read key denied list: %s", res.text)
	}
	// A write without mcp:provision is rejected at the HTTP boundary (403).
	if res := httpToolCall(t, handler, "bb_org1_read", "set_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "A", "value": "x",
	}); res.status != http.StatusForbidden {
		t.Fatalf("secrets:read key must not set (want HTTP 403, got %d %q)", res.status, res.text)
	}
	if res := httpToolCall(t, handler, "bb_org1_read", "delete_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "A",
	}); res.status != http.StatusForbidden {
		t.Fatalf("secrets:read key must not delete (want HTTP 403, got %d %q)", res.status, res.text)
	}

	// Provision but no secrets:write: mutations are denied by the narrow scope
	// gate inside the tool (JSON-RPC IsError), not only at the HTTP boundary.
	if res := httpToolCall(t, handler, "bb_org1_provision", "set_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "A", "value": "x",
	}); !res.isError || !strings.Contains(res.text, "insufficient scope") {
		t.Fatalf("provision-only key must not set: %q (err=%v)", res.text, res.isError)
	}

	// Write-only key: mutations succeed, reads denied.
	if res := httpToolCall(t, handler, "bb_org1_write", "set_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY", "value": "v1",
	}); res.isError {
		t.Fatalf("secrets:write key failed set: %s", res.text)
	}
	if res := httpToolCall(t, handler, "bb_org1_write", "list_project_secrets", args); !res.isError || !strings.Contains(res.text, "insufficient scope") {
		t.Fatalf("write-only key must not list: %q (err=%v)", res.text, res.isError)
	}
	if res := httpToolCall(t, handler, "bb_org1_write", "read_project_secret_value", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY",
	}); !res.isError || !strings.Contains(res.text, "insufficient scope") {
		t.Fatalf("write-only key must not read values: %q (err=%v)", res.text, res.isError)
	}
	if res := httpToolCall(t, handler, "bb_org1_write", "delete_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY",
	}); res.isError {
		t.Fatalf("secrets:write key failed delete: %s", res.text)
	}

	// No-scope key: no secret tools at all.
	if res := httpToolCall(t, handler, "bb_org1_noscopes", "list_project_secrets", args); !res.isError || !strings.Contains(res.text, "insufficient scope") {
		t.Fatalf("empty-scope key must not list: %q (err=%v)", res.text, res.isError)
	}
}

func TestMCPSecretScopeMatrixStdio(t *testing.T) {
	c, _, _, _, _ := newSecretTestComponent(t)

	readOnly := mcp.Principal{Kind: mcp.PrincipalOrg, OrgID: 1, Scopes: []string{mcp.ScopeSecretsRead}}
	writeOnly := mcp.Principal{Kind: mcp.PrincipalOrg, OrgID: 1, Scopes: []string{mcp.ScopeProvision, mcp.ScopeSecretsWrite}}

	args := map[string]any{"project_id": "proj-1", "environment_id": "env-1"}

	if res := stdioToolCall(t, c, readOnly, "list_project_secrets", args); res.isError {
		t.Fatalf("stdio read-only list failed: %s", res.text)
	}
	if res := stdioToolCall(t, c, readOnly, "set_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "A", "value": "x",
	}); !res.isError || !strings.Contains(res.text, "insufficient scope") {
		t.Fatalf("stdio read-only set must be denied: %q (err=%v)", res.text, res.isError)
	}

	if res := stdioToolCall(t, c, writeOnly, "set_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "DB_URL", "value": "postgres://x",
	}); res.isError {
		t.Fatalf("stdio write-only set failed: %s", res.text)
	}
	if res := stdioToolCall(t, c, writeOnly, "list_project_secrets", args); !res.isError || !strings.Contains(res.text, "insufficient scope") {
		t.Fatalf("stdio write-only list must be denied: %q (err=%v)", res.text, res.isError)
	}
	if res := stdioToolCall(t, c, writeOnly, "delete_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "DB_URL",
	}); res.isError {
		t.Fatalf("stdio write-only delete failed: %s", res.text)
	}
}

// --- SC-e89s07-P0-03 / P0-04: Site deploy keys stay Site-bound ----------------

func TestSiteKeyBindingEnvTools(t *testing.T) {
	c, _, _, _, _ := newSecretTestComponent(t)
	handler := c.Handler()

	// Bound to site-a: the same Site succeeds.
	res := httpToolCall(t, handler, "bb_dep_sitea", "get_site_env_vars", map[string]any{"site_id": "site-a"})
	if res.isError {
		t.Fatalf("site key for own site failed: %s", res.text)
	}
	if !strings.Contains(res.text, "TOKEN") {
		t.Fatalf("expected masked env var for own site, got: %s", res.text)
	}

	// Cross-Site target denied without revealing Site B.
	res = httpToolCall(t, handler, "bb_dep_sitea", "get_site_env_vars", map[string]any{"site_id": "site-b"})
	if !res.isError || !strings.Contains(res.text, "site authorization denied") {
		t.Fatalf("site key crossed sites: %q (err=%v)", res.text, res.isError)
	}
	if strings.Contains(res.text, "site-b") {
		t.Fatalf("cross-Site denial leaked target: %s", res.text)
	}

	// Site write tool: own Site ok, other Site denied.
	res = httpToolCall(t, handler, "bb_dep_sitea", "set_site_env_vars", map[string]any{
		"site_id": "site-a", "vars": map[string]any{"NEW": "1"},
	})
	if res.isError {
		t.Fatalf("site key set on own site failed: %s", res.text)
	}
	res = httpToolCall(t, handler, "bb_dep_sitea", "set_site_env_vars", map[string]any{
		"site_id": "site-b", "vars": map[string]any{"NEW": "1"},
	})
	if !res.isError || !strings.Contains(res.text, "site authorization denied") {
		t.Fatalf("site key set crossed sites: %q (err=%v)", res.text, res.isError)
	}

	// Site keys cannot reach Project secrets without an explicit policy.
	res = httpToolCall(t, handler, "bb_dep_sitea", "list_project_secrets", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1",
	})
	if !res.isError || !strings.Contains(res.text, "insufficient scope") {
		t.Fatalf("site key must not read project secrets by default: %q (err=%v)", res.text, res.isError)
	}
}

func TestEnvVarIsolationSiteKey(t *testing.T) {
	c, _, _, _, _ := newSecretTestComponent(t)
	handler := c.Handler()

	// The site-b key cannot see site-a's env vars and vice versa.
	if res := httpToolCall(t, handler, "bb_dep_sitea", "get_site_env_vars", map[string]any{"site_id": "site-b"}); !res.isError {
		t.Fatalf("site-a key must not read site-b: %s", res.text)
	}
	if res := httpToolCall(t, handler, "bb_dep_siteb", "get_site_env_vars", map[string]any{"site_id": "site-a"}); !res.isError {
		t.Fatalf("site-b key must not read site-a: %s", res.text)
	}
	// The site-b key cannot delete on site-a.
	if res := httpToolCall(t, handler, "bb_dep_siteb", "delete_site_env_var", map[string]any{"site_id": "site-a", "key": "TOKEN"}); !res.isError || !strings.Contains(res.text, "site authorization denied") {
		t.Fatalf("site-b key must not delete on site-a: %q (err=%v)", res.text, res.isError)
	}
}

func TestEnvVarAuthSiteBinding(t *testing.T) {
	c, _, _, _, _ := newSecretTestComponent(t)

	// stdio transport: a Site principal is bound to exactly one Site.
	siteA := mcp.Principal{Kind: mcp.PrincipalSite, SiteID: "site-a", Scopes: []string{"deploy"}}

	if res := stdioToolCall(t, c, siteA, "get_site_env_vars", map[string]any{"site_id": "site-a"}); res.isError {
		t.Fatalf("stdio site key for own site failed: %s", res.text)
	}
	if res := stdioToolCall(t, c, siteA, "get_site_env_vars", map[string]any{"site_id": "site-b"}); !res.isError || !strings.Contains(res.text, "site authorization denied") {
		t.Fatalf("stdio site key crossed sites: %q (err=%v)", res.text, res.isError)
	}
}

// --- Native tool flows (t3: TestProjectSecret*) ------------------------------

func TestProjectSecretList(t *testing.T) {
	c, _, secMock, _, _ := newSecretTestComponent(t)
	principal := mcp.Principal{Kind: mcp.PrincipalOrg, OrgID: 1, Scopes: []string{mcp.ScopeProvision, mcp.ScopeSecretsRead, mcp.ScopeSecretsWrite}}

	// Seed one secret through the manager seam (as REST would).
	_, _ = secMock.CreateSecret(context.Background(), principal, "proj-1", "env-1", "folder-proj-1-env-1", "API_KEY", "plaintext-value-1234")

	res := stdioToolCall(t, c, principal, "list_project_secrets", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1",
	})
	if res.isError {
		t.Fatalf("list failed: %s", res.text)
	}
	if !strings.Contains(res.text, `"key":"API_KEY"`) {
		t.Fatalf("expected secret key in list: %s", res.text)
	}
	// Metadata-only: masked preview present, plaintext absent.
	if strings.Contains(res.text, "plaintext-value-1234") {
		t.Fatalf("list leaked plaintext: %s", res.text)
	}
	if !strings.Contains(res.text, "value_preview") {
		t.Fatalf("list must carry masked previews: %s", res.text)
	}
	// Bounded + actor metadata (SC-e89s07-P1-06).
	if !strings.Contains(res.text, `"count":1`) || !strings.Contains(res.text, `"kind":"org"`) {
		t.Fatalf("list must be bounded with actor metadata: %s", res.text)
	}
}

func TestProjectSecretGetMetadataOnly(t *testing.T) {
	c, _, secMock, _, _ := newSecretTestComponent(t)
	principal := mcp.Principal{Kind: mcp.PrincipalOrg, OrgID: 1, Scopes: []string{mcp.ScopeProvision, mcp.ScopeSecretsRead, mcp.ScopeSecretsWrite}}

	_, _ = secMock.CreateSecret(context.Background(), principal, "proj-1", "env-1", "folder-proj-1-env-1", "API_KEY", "never-expose-9876")

	res := stdioToolCall(t, c, principal, "get_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY",
	})
	if res.isError {
		t.Fatalf("get failed: %s", res.text)
	}
	if strings.Contains(res.text, "never-expose-9876") {
		t.Fatalf("get_project_secret must be metadata-only: %s", res.text)
	}
	if strings.Contains(res.text, `"value"`) {
		t.Fatalf("get_project_secret must not carry a value field: %s", res.text)
	}
	if !strings.Contains(res.text, `"value_preview"`) {
		t.Fatalf("get_project_secret must carry a masked preview: %s", res.text)
	}
}

func TestProjectSecretValueRead(t *testing.T) {
	c, _, secMock, _, _ := newSecretTestComponent(t)
	principal := mcp.Principal{Kind: mcp.PrincipalOrg, OrgID: 1, Scopes: []string{mcp.ScopeProvision, mcp.ScopeSecretsRead, mcp.ScopeSecretsWrite}}

	_, _ = secMock.CreateSecret(context.Background(), principal, "proj-1", "env-1", "folder-proj-1-env-1", "API_KEY", "explicit-value-abc")

	res := stdioToolCall(t, c, principal, "read_project_secret_value", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY",
	})
	if res.isError {
		t.Fatalf("explicit value read failed: %s", res.text)
	}
	if !strings.Contains(res.text, "explicit-value-abc") {
		t.Fatalf("explicit value read must return the value: %s", res.text)
	}
	if !strings.Contains(res.text, `"kind":"org"`) {
		t.Fatalf("value read must carry audit actor metadata: %s", res.text)
	}

	// Missing secret is a value-free not-found.
	res = stdioToolCall(t, c, principal, "read_project_secret_value", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "MISSING",
	})
	if !res.isError || !strings.Contains(res.text, "secret not found") {
		t.Fatalf("missing value read must be value-free not-found: %q (err=%v)", res.text, res.isError)
	}
}

func TestProjectSecretSetDelete(t *testing.T) {
	c, _, secMock, _, _ := newSecretTestComponent(t)
	principal := mcp.Principal{Kind: mcp.PrincipalOrg, OrgID: 1, Scopes: []string{mcp.ScopeProvision, mcp.ScopeSecretsRead, mcp.ScopeSecretsWrite}}

	// set = create-or-update: first call creates, response is masked metadata.
	res := stdioToolCall(t, c, principal, "set_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY", "value": "rotate-me-0000",
	})
	if res.isError {
		t.Fatalf("set failed: %s", res.text)
	}
	if strings.Contains(res.text, "rotate-me-0000") {
		t.Fatalf("set must not echo the value: %s", res.text)
	}
	if !strings.Contains(res.text, `"action":"set"`) || !strings.Contains(res.text, `"kind":"org"`) {
		t.Fatalf("set must carry action and actor metadata: %s", res.text)
	}

	// Second set updates (immutable version append), still masked.
	res = stdioToolCall(t, c, principal, "set_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY", "value": "rotate-me-1111",
	})
	if res.isError {
		t.Fatalf("update failed: %s", res.text)
	}
	if strings.Contains(res.text, "rotate-me-1111") {
		t.Fatalf("update must not echo the value: %s", res.text)
	}

	// Delete removes it; second delete is a value-free not-found.
	res = stdioToolCall(t, c, principal, "delete_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY",
	})
	if res.isError {
		t.Fatalf("delete failed: %s", res.text)
	}
	if !strings.Contains(res.text, `"status":"deleted"`) {
		t.Fatalf("delete must return status: %s", res.text)
	}
	res = stdioToolCall(t, c, principal, "delete_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY",
	})
	if !res.isError || !strings.Contains(res.text, "secret not found") {
		t.Fatalf("second delete must be not-found: %q (err=%v)", res.text, res.isError)
	}

	// Upsert path landed in the manager seam.
	if _, ok := secMock.entries["folder-proj-1-env-1"]["API_KEY"]; ok {
		t.Fatal("secret must be deleted from the manager seam")
	}
}

// --- SC-e89s07-P1-06: bounded and masked responses ----------------------------

func TestProjectSecretBoundedArgs(t *testing.T) {
	c, _, _, _, _ := newSecretTestComponent(t)
	handler := c.Handler()

	// Missing required identifiers.
	if res := httpToolCall(t, handler, "bb_org1_all", "list_project_secrets", map[string]any{}); !res.isError || !strings.Contains(res.text, "project_id is required") {
		t.Fatalf("missing project_id: %q (err=%v)", res.text, res.isError)
	}
	if res := httpToolCall(t, handler, "bb_org1_all", "list_project_secrets", map[string]any{"project_id": "proj-1"}); !res.isError || !strings.Contains(res.text, "environment_id is required") {
		t.Fatalf("missing environment_id: %q (err=%v)", res.text, res.isError)
	}

	// Invalid secret keys are rejected before any storage call.
	if res := httpToolCall(t, handler, "bb_org1_all", "set_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "bad-key!", "value": "x",
	}); !res.isError || !strings.Contains(res.text, "invalid secret key") {
		t.Fatalf("invalid key: %q (err=%v)", res.text, res.isError)
	}

	// Oversized values are rejected without echoing the value.
	big := strings.Repeat("v", 70*1024)
	res := httpToolCall(t, handler, "bb_org1_all", "set_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY", "value": big,
	})
	if !res.isError || !strings.Contains(res.text, "secret value too large") {
		t.Fatalf("oversized value: %q (err=%v)", res.text, res.isError)
	}
	if strings.Contains(res.text, strings.Repeat("v", 16)) {
		t.Fatalf("oversized value echoed in error: %.80s", res.text)
	}
}

func TestMCPSecretMaskedResults(t *testing.T) {
	c, _, _, _, _ := newSecretTestComponent(t)
	principal := mcp.Principal{Kind: mcp.PrincipalOrg, OrgID: 1, Scopes: []string{mcp.ScopeProvision, mcp.ScopeSecretsRead, mcp.ScopeSecretsWrite}}

	const plaintext = "super-secret-1234"
	if res := stdioToolCall(t, c, principal, "set_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY", "value": plaintext,
	}); res.isError {
		t.Fatalf("set failed: %s", res.text)
	}
	if res := stdioToolCall(t, c, principal, "list_project_secrets", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1",
	}); res.isError {
		t.Fatalf("list failed: %s", res.text)
	} else if strings.Contains(res.text, plaintext) {
		t.Fatalf("list leaked plaintext: %s", res.text)
	}

	// The explicit value read is the only path to plaintext.
	if res := stdioToolCall(t, c, principal, "read_project_secret_value", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY",
	}); res.isError || !strings.Contains(res.text, plaintext) {
		t.Fatalf("explicit value read must return plaintext: %q (err=%v)", res.text, res.isError)
	}

	// Mutation responses never contain the submitted value.
	if res := stdioToolCall(t, c, principal, "set_project_secret", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY", "value": "rotate-9999",
	}); strings.Contains(res.text, "rotate-9999") {
		t.Fatalf("set echoed the value: %s", res.text)
	}

	// Actor metadata is present and safe (never a raw key).
	if res := stdioToolCall(t, c, principal, "list_project_secrets", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1",
	}); !strings.Contains(res.text, `"kind":"org"`) || !strings.Contains(res.text, `"org_id":1`) {
		t.Fatalf("list must carry safe actor metadata: %s", res.text)
	}
}

// --- SC-e89s07-P0-05: tool errors are safe ------------------------------------

func TestMCPSecretSafeErrors(t *testing.T) {
	// HTTP transport: an internal storage failure surfaces as a generic
	// message with no SQL, key material, or plaintext.
	c, _, secMock, _, _ := newSecretTestComponent(t)
	handler := c.Handler()

	secMock.nextErr = errors.New(`sql: no such table: secret_versions (db: sqlite)`)

	res := httpToolCall(t, handler, "bb_org1_all", "list_project_secrets", map[string]any{
		"project_id": "proj-1", "environment_id": "env-1",
	})
	if !res.isError || !strings.Contains(res.text, "internal error") {
		t.Fatalf("internal failure must be generic: %q (err=%v)", res.text, res.isError)
	}
	for _, leak := range []string{"sql:", "secret_versions", "no such table"} {
		if strings.Contains(res.text, leak) {
			t.Fatalf("internal failure leaked %q: %s", leak, res.text)
		}
	}

	// stdio transport: same contract for a mutation failure.
	secMock.nextErr = errors.New(`update secret_versions: no such table`)
	res = stdioToolCall(t, c, mcp.Principal{Kind: mcp.PrincipalOrg, OrgID: 1, Scopes: []string{mcp.ScopeProvision, mcp.ScopeSecretsRead, mcp.ScopeSecretsWrite}},
		"set_project_secret", map[string]any{
			"project_id": "proj-1", "environment_id": "env-1", "key": "API_KEY", "value": "x",
		})
	if !res.isError || !strings.Contains(res.text, "internal error") {
		t.Fatalf("stdio internal failure must be generic: %q (err=%v)", res.text, res.isError)
	}
	if strings.Contains(res.text, "secret_versions") {
		t.Fatalf("stdio internal failure leaked SQL: %s", res.text)
	}
}

// --- HTTP auth surface for the new secret tools --------------------------------

func TestMCPSecretHTTPRequiresAuth(t *testing.T) {
	c, _, _, _, _ := newSecretTestComponent(t)
	handler := c.Handler()

	// No token: 401.
	res := httpToolCall(t, handler, "", "list_project_secrets", map[string]any{"project_id": "proj-1", "environment_id": "env-1"})
	if res.status != http.StatusUnauthorized {
		t.Fatalf("no-token list status = %d, want 401", res.status)
	}
	// Invalid token: 401.
	res = httpToolCall(t, handler, "bb_invalid", "list_project_secrets", map[string]any{"project_id": "proj-1", "environment_id": "env-1"})
	if res.status != http.StatusUnauthorized {
		t.Fatalf("invalid-token list status = %d, want 401", res.status)
	}
	// Write tool without mcp:provision: 403 at the HTTP boundary.
	res = httpToolCall(t, handler, "bb_org1_writenoprov", "set_project_secret", map[string]any{"project_id": "proj-1", "environment_id": "env-1", "key": "A", "value": "x"})
	if res.status != http.StatusForbidden {
		t.Fatalf("write without provision status = %d, want 403", res.status)
	}
}
