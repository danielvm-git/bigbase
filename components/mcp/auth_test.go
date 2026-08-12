package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/mcp"
	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(string, ...any)  {}
func (testLogger) Warn(string, ...any)  {}
func (testLogger) Error(string, ...any) {}
func (testLogger) Debug(string, ...any) {}

type mockOrgKeyAuth struct {
}

func (m mockOrgKeyAuth) ResolveOrgKey(token string) (int64, []string, error) {
	switch token {
	case "bb_valid_read", "bb_read_only":
		return 1, nil, nil
	case "bb_valid_provision":
		return 1, []string{mcp.ScopeProvision}, nil
	case "bb_dep_sitekey":
		return 0, nil, errors.New("invalid org api key")
	default:
		return 0, nil, errors.New("api key not found")
	}
}

func toolCallBody(tool string) string {
	params, _ := json.Marshal(map[string]any{
		"name":      tool,
		"arguments": map[string]any{"name": "test"},
	})
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  json.RawMessage(params),
	})
	return string(body)
}

func TestBearerAuthMiddleware(t *testing.T) {
	c := mcp.New(mcp.Options{
		Enabled:             true,
		OrgKeyAuthenticator: mockOrgKeyAuth{},
	})
	handler := c.Handler()

	t.Run("401 on write tool without auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(toolCallBody("create_repo")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("401 on invalid key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(toolCallBody("create_repo")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer bb_invalid")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("403 on write without provision scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(toolCallBody("create_repo")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer bb_valid_read")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("pass on public tool without auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(toolCallBody("get_ci_template")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
			t.Fatalf("public tool blocked with status %d", w.Code)
		}
	})

	t.Run("pass on write with provision scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(toolCallBody("create_repo")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer bb_valid_provision")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
			t.Fatalf("provision key blocked with status %d", w.Code)
		}
	})

	t.Run("pass on lowercase bearer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(toolCallBody("create_repo")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "bearer bb_valid_provision")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
			t.Fatalf("lowercase bearer blocked with status %d", w.Code)
		}
	})
}

func TestBearerAuthContext(t *testing.T) {
	c := mcp.New(mcp.Options{
		Enabled:             true,
		OrgKeyAuthenticator: mockOrgKeyAuth{},
		GitCreator:          &mockGitCreator{id: "r1", name: "test"},
	})

	srv, err := c.NewMCPServer()
	if err != nil {
		t.Fatalf("NewMCPServer: %v", err)
	}

	ctx := mcp.WithOrgAuth(context.Background(), 42, []string{mcp.ScopeProvision})
	t1, t2 := mcpsdk.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "create_repo", Arguments: map[string]any{"name": "proj"}})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("create_repo failed: %v", result.Content)
	}
	orgID, _, ok := mcp.OrgAuthFromContext(ctx)
	if !ok || orgID != 42 {
		t.Fatalf("org context = %d, ok=%v", orgID, ok)
	}
}

func TestRequireScope(t *testing.T) {
	c := mcp.New(mcp.Options{
		Enabled:             true,
		OrgKeyAuthenticator: mockOrgKeyAuth{},
		GitCreator:          &mockGitCreator{id: "r1", name: "x"},
	})
	ctx := mcp.WithOrgAuth(context.Background(), 1, nil)
	text := callToolTextWithCtx(t, c, ctx, "create_repo", map[string]any{"name": "x"})
	if !strings.Contains(text, "insufficient scope") {
		t.Fatalf("expected scope error, got: %s", text)
	}
}

func TestProvisionScopeGate(t *testing.T) {
	c := mcp.New(mcp.Options{
		Enabled:             true,
		OrgKeyAuthenticator: mockOrgKeyAuth{},
		GitCreator:          &mockGitCreator{id: "abc", name: "my-project"},
	})
	ctx := mcp.WithOrgAuth(context.Background(), 1, []string{mcp.ScopeProvision})
	text := callToolTextWithCtx(t, c, ctx, "create_repo", map[string]any{"name": "my-project"})
	if !strings.Contains(text, `"repo_id":"abc"`) {
		t.Fatalf("unexpected: %s", text)
	}
}

func TestReadToolsNoScope(t *testing.T) {
	c := mcp.New(mcp.Options{
		Enabled:             true,
		OrgKeyAuthenticator: mockOrgKeyAuth{},
	})
	ctx := mcp.WithOrgAuth(context.Background(), 1, nil)
	text := callToolTextWithCtx(t, c, ctx, "get_ci_template", nil)
	if strings.Contains(text, "authorization required") || strings.Contains(text, "insufficient scope") {
		t.Fatalf("public/read tool blocked: %s", text)
	}
}

func TestPublicReadTier(t *testing.T) {
	c := mcp.New(mcp.Options{
		Enabled:             true,
		OrgKeyAuthenticator: mockOrgKeyAuth{},
	})
	for _, tool := range []string{"ping", "list_services", "get_ci_template", "deploy_guide"} {
		t.Run(tool, func(t *testing.T) {
			text := callToolText(t, c, tool, nil)
			if strings.Contains(text, "authorization required") {
				t.Fatalf("%s blocked: %s", tool, text)
			}
		})
	}
}

func TestDenyByDefault(t *testing.T) {
	c := mcp.New(mcp.Options{
		Enabled:             true,
		OrgKeyAuthenticator: mockOrgKeyAuth{},
	})
	text := callToolText(t, c, "create_repo", map[string]any{"name": "x"})
	if !strings.Contains(text, "authorization required") {
		t.Fatalf("expected auth error, got: %s", text)
	}
}

func TestResolveOrgKeyRejectsSiteKey(t *testing.T) {
	logger := testLogger{}
	k := kernel.New(logger)
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "test-secret-32-chars!!!"})
	k.Register(d)
	k.Register(a)
	if err := k.Start(); err != nil {
		t.Fatalf("kernel start: %v", err)
	}
	t.Cleanup(func() { _ = k.Stop() })

	_, _, err := a.ResolveOrgKey("bb_dep_" + strings.Repeat("a", 64))
	if err == nil {
		t.Fatal("expected site key rejection")
	}
}

type mockSiteTargetAuthorizer struct {
	allowed string
	called  bool
}

func (m *mockSiteTargetAuthorizer) AuthorizeSiteTarget(_ context.Context, siteID string, orgID int64) error {
	m.called = true
	if orgID != 1 || siteID != m.allowed {
		return errors.New("denied")
	}
	return nil
}

func TestMCPEnvToolsEnforceSiteTargetOwnership(t *testing.T) {
	manager := newMockEnvVarManager()
	manager.vars["site-a"] = []mcp.SiteEnvVar{{SiteID: "site-a", Key: "TOKEN", ValuePreview: "••••oken"}}
	authorizer := &mockSiteTargetAuthorizer{allowed: "site-a"}
	c := mcp.New(mcp.Options{
		Enabled:              true,
		SiteEnvVarManager:    manager,
		OrgKeyAuthenticator:  mockOrgKeyAuth{},
		SiteTargetAuthorizer: authorizer,
	})
	ctx := mcp.WithOrgAuth(context.Background(), 1, nil)

	denied := callToolTextWithCtx(t, c, ctx, "get_site_env_vars", map[string]any{"site_id": "site-b"})
	if !strings.Contains(denied, "site authorization denied") {
		t.Fatalf("expected generic ownership denial, got %q", denied)
	}
	if !authorizer.called {
		t.Fatal("expected target authorizer to run before Site env access")
	}

	authorizer.called = false
	allowed := callToolTextWithCtx(t, c, ctx, "get_site_env_vars", map[string]any{"site_id": "site-a"})
	if !strings.Contains(allowed, "TOKEN") || !authorizer.called {
		t.Fatalf("expected authorized Site env response, got %q", allowed)
	}
}

func callToolTextWithCtx(t *testing.T, c *mcp.Component, ctx context.Context, name string, args map[string]any) string {
	t.Helper()
	srv, err := c.NewMCPServer()
	if err != nil {
		t.Fatalf("NewMCPServer: %v", err)
	}
	t1, t2 := mcpsdk.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer func() { _ = session.Close() }()
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	tc, ok := result.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

func TestGetDeployStatusShortID(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true, OrgKeyAuthenticator: mockOrgKeyAuth{}})
	ctx := mcp.WithOrgAuth(context.Background(), 1, nil)
	// Should not panic on short deployment_id in list mode
	text := callToolTextWithCtx(t, c, ctx, "get_deploy_status", map[string]any{"deployment_id": "x"})
	if text == "" {
		t.Fatal("expected response")
	}
}

func TestHTTPReadToolRequiresAuth(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true, OrgKeyAuthenticator: mockOrgKeyAuth{}})
	handler := c.Handler()

	params, _ := json.Marshal(map[string]any{"name": "list_repos", "arguments": map[string]any{}})
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": json.RawMessage(params)})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("list_repos without auth: status = %d, want 401", w.Code)
	}
}
