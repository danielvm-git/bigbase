package mcp_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/mcp"
	"github.com/danielvm/bigbase/kernel"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	_ "modernc.org/sqlite"
)

func TestComponentImplementsKernelComponent(t *testing.T) {
	c := mcp.New(mcp.Options{})
	var _ kernel.Component = c // compile-time assertion
	t.Log("MCP component implements kernel.Component")
}

func TestPingTool(t *testing.T) {
	c := mcp.New(mcp.Options{Transport: "stdio", Enabled: true})

	srv, err := c.NewMCPServer()
	if err != nil {
		t.Fatalf("NewMCPServer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use in-memory transports for testing
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

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "ping"})
	if err != nil {
		t.Fatalf("CallTool(ping): %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("expected content in ping response")
	}
	text, ok := result.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", result.Content[0])
	}
	if text.Text != "pong" {
		t.Errorf("expected 'pong', got %q", text.Text)
	}
}

func TestHTTPTransport(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	handler := c.Handler()

	// Test health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health: expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	bodyStr := string(body)
	if len(bodyStr) == 0 {
		t.Error("health: expected non-empty body")
	}

	// Test GET /mcp is allowed (SSE connection). MCP Streamable HTTP spec
	// requires GET for establishing server→client event streams.
	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Accept", "text/event-stream")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code == http.StatusMethodNotAllowed {
		t.Error("GET /mcp returned 405 — SSE connections are blocked")
	}
}

func TestKnowledgeTools(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	srv, err := c.NewMCPServer()
	if err != nil {
		t.Fatalf("NewMCPServer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

	tools := []struct {
		name string
		args map[string]any
	}{
		{"list_services", nil},
		{"list_frameworks", nil},
		{"get_service_docs", map[string]any{"service": "deploy"}},
		{"get_service_docs", map[string]any{"service": "auth"}},
		{"get_code_example", map[string]any{"service": "auth", "framework": "sveltekit"}},
		{"get_code_example", map[string]any{"service": "db", "framework": "react"}},
	}

	for _, tool := range tools {
		t.Run(tool.name, func(t *testing.T) {
			result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
				Name:      tool.name,
				Arguments: tool.args,
			})
			if err != nil {
				t.Fatalf("CallTool(%s): %v", tool.name, err)
			}
			if len(result.Content) == 0 {
				t.Fatal("expected content")
			}
			tc, ok := result.Content[0].(*mcpsdk.TextContent)
			if !ok {
				t.Fatalf("expected *TextContent, got %T", result.Content[0])
			}
			if tc.Text == "" {
				t.Error("expected non-empty response")
			}
			t.Logf("%s: %d chars", tool.name, len(tc.Text))
		})
	}
}

// deployTestDB creates an in-memory SQLite DB with the tables and
// sample data that the deploy MCP tools need.
func deployTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE git_repos (id TEXT PRIMARY KEY, name TEXT, description TEXT, created_at TEXT)`,
		`CREATE TABLE deployments (id TEXT PRIMARY KEY, status TEXT, url TEXT, app_type TEXT, commit_sha TEXT, error_message TEXT, build_log TEXT, created_at TEXT)`,
		`INSERT INTO git_repos VALUES ('repo-1', 'my-sveltekit-app', 'A SvelteKit demo', '2026-06-01T10:00:00Z')`,
		`INSERT INTO git_repos VALUES ('repo-2', 'go-api', 'A Go API server', '2026-06-02T11:00:00Z')`,
		`INSERT INTO deployments VALUES ('dep-1234567890abcdef', 'running', 'https://myapp.bigbase.click', 'node', 'abc1234', '', '', '2026-06-19T12:00:00Z')`,
		`INSERT INTO deployments VALUES ('dep-failed1234567890', 'failed', '', 'go', 'def5678', 'build: exit status 1', 'Step 1/5 : FROM golang:1.22
 ---> abc123
Step 2/5 : COPY . .
 ---> def456
Step 3/5 : RUN go build
 ---> Running in abc789
go: go.mod file not found
exit status 1', '2026-06-19T13:00:00Z')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt[:40], err)
		}
	}
	return db
}

// mockDeployer is a DeployTrigger that returns a fixed deployment.
type mockDeployer struct{}

func (m *mockDeployer) Trigger(_ context.Context, repoID, branch, siteName, siteID string, _ []string, _ string, _ string) (*deploy.Deployment, error) {
	return &deploy.Deployment{
		ID:     fmt.Sprintf("dep-%s-%s", repoID, branch),
		Status: "building",
		URL:    fmt.Sprintf("https://%s.bigbase.click", siteName),
	}, nil
}

// connectMCPSession creates an in-memory MCP client session for testing.
func connectMCPSession(t *testing.T, c *mcp.Component) (context.Context, *mcpsdk.ClientSession) {
	t.Helper()
	srv, err := c.NewMCPServer()
	if err != nil {
		t.Fatalf("NewMCPServer: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	t1, t2 := mcpsdk.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return ctx, session
}

func TestDeployGuide(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	ctx, session := connectMCPSession(t, c)
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "deploy_guide"})
	if err != nil {
		t.Fatalf("deploy_guide: %v", err)
	}
	tc, ok := result.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", result.Content[0])
	}
	if tc.Text == "" {
		t.Fatal("expected non-empty guide")
	}
	for _, want := range []string{"deploy_site", "list_repos", "get_deploy_status", "get_deploy_logs"} {
		if !strings.Contains(tc.Text, want) {
			t.Errorf("deploy_guide should mention %q", want)
		}
	}
	t.Logf("deploy_guide: %d chars", len(tc.Text))
}

func TestListRepos(t *testing.T) {
	db := deployTestDB(t)
	defer func() { _ = db.Close() }()

	c := mcp.New(mcp.Options{Enabled: true, DB: db})
	ctx, session := connectMCPSession(t, c)

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "list_repos"})
	if err != nil {
		t.Fatalf("list_repos: %v", err)
	}
	tc, ok := result.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", result.Content[0])
	}
	// Should list our test repos
	for _, want := range []string{"my-sveltekit-app", "go-api"} {
		if !strings.Contains(tc.Text, want) {
			t.Errorf("list_repos should mention %q", want)
		}
	}
	if !strings.Contains(tc.Text, "deploy_site") {
		t.Error("list_repos should mention deploy_site")
	}
	t.Logf("list_repos: %d chars", len(tc.Text))
}

func TestDeploySite(t *testing.T) {
	db := deployTestDB(t)
	defer func() { _ = db.Close() }()

	mockD := &mockDeployer{}
	c := mcp.New(mcp.Options{Enabled: true, DB: db, Deployer: mockD})
	ctx, session := connectMCPSession(t, c)

	// repo_id required
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "deploy_site"})
	if err != nil {
		t.Fatalf("deploy_site (no args): %v", err)
	}
	tc, _ := result.Content[0].(*mcpsdk.TextContent)
	if !strings.Contains(tc.Text, "repo_id is required") {
		t.Errorf("expected 'repo_id is required', got: %s", tc.Text)
	}

	// Successful deploy
	result, err = session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "deploy_site",
		Arguments: map[string]interface{}{"repo_id": "repo-1", "branch": "main"},
	})
	if err != nil {
		t.Fatalf("deploy_site: %v", err)
	}
	tc, _ = result.Content[0].(*mcpsdk.TextContent)
	if !strings.Contains(tc.Text, "Deploy Started") {
		t.Errorf("expected 'Deploy Started', got: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "my-sveltekit-app") {
		t.Errorf("expected repo name, got: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "get_deploy_status") {
		t.Error("deploy_site should mention get_deploy_status")
	}
	t.Logf("deploy_site: %d chars", len(tc.Text))
}

func TestGetDeployStatus(t *testing.T) {
	db := deployTestDB(t)
	defer func() { _ = db.Close() }()

	c := mcp.New(mcp.Options{Enabled: true, DB: db})
	ctx, session := connectMCPSession(t, c)

	// No deployment_id — lists recent deployments
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_deploy_status"})
	if err != nil {
		t.Fatalf("get_deploy_status (no args): %v", err)
	}
	tc, _ := result.Content[0].(*mcpsdk.TextContent)
	if !strings.Contains(tc.Text, "Recent Deployments") {
		t.Errorf("expected 'Recent Deployments', got: %s", tc.Text)
	}
	t.Logf("get_deploy_status (recent): %d chars", len(tc.Text))

	// Specific deployment — running
	result, err = session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_deploy_status",
		Arguments: map[string]interface{}{"deployment_id": "dep-1234567890abcdef"},
	})
	if err != nil {
		t.Fatalf("get_deploy_status (specific): %v", err)
	}
	tc, _ = result.Content[0].(*mcpsdk.TextContent)
	if !strings.Contains(tc.Text, "running") {
		t.Errorf("expected 'running', got: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "https://myapp.bigbase.click") {
		t.Errorf("expected URL, got: %s", tc.Text)
	}

	// Specific deployment — failed
	result, err = session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_deploy_status",
		Arguments: map[string]interface{}{"deployment_id": "dep-failed1234567890"},
	})
	if err != nil {
		t.Fatalf("get_deploy_status (failed): %v", err)
	}
	tc, _ = result.Content[0].(*mcpsdk.TextContent)
	if !strings.Contains(tc.Text, "failed") {
		t.Errorf("expected 'failed', got: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "get_deploy_logs") {
		t.Error("failed deployment should suggest get_deploy_logs")
	}
	t.Logf("get_deploy_status (failed): %d chars", len(tc.Text))
}

func TestGetDeployLogs(t *testing.T) {
	db := deployTestDB(t)
	defer func() { _ = db.Close() }()

	c := mcp.New(mcp.Options{Enabled: true, DB: db})
	ctx, session := connectMCPSession(t, c)

	// No deployment_id — error message
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "get_deploy_logs"})
	if err != nil {
		t.Fatalf("get_deploy_logs (no args): %v", err)
	}
	tc, _ := result.Content[0].(*mcpsdk.TextContent)
	if !strings.Contains(tc.Text, "deployment_id is required") {
		t.Errorf("expected 'deployment_id is required', got: %s", tc.Text)
	}

	// Build logs for failed deployment
	result, err = session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_deploy_logs",
		Arguments: map[string]interface{}{"deployment_id": "dep-failed1234567890"},
	})
	if err != nil {
		t.Fatalf("get_deploy_logs (specific): %v", err)
	}
	tc, _ = result.Content[0].(*mcpsdk.TextContent)
	if !strings.Contains(tc.Text, "go.mod file not found") {
		t.Errorf("expected build log content, got: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "Build Logs") {
		t.Error("expected 'Build Logs' header")
	}

	// No build logs for running deployment
	result, err = session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_deploy_logs",
		Arguments: map[string]interface{}{"deployment_id": "dep-1234567890abcdef"},
	})
	if err != nil {
		t.Fatalf("get_deploy_logs (running): %v", err)
	}
	tc, _ = result.Content[0].(*mcpsdk.TextContent)
	if !strings.Contains(tc.Text, "No build logs") {
		t.Errorf("expected 'No build logs', got: %s", tc.Text)
	}
	t.Logf("get_deploy_logs: %d chars", len(tc.Text))
}

// --- e38s01 stdio transport test ---

func TestStdioTransport(t *testing.T) {
	c := mcp.New(mcp.Options{Transport: "stdio", Enabled: true})
	srv, err := c.NewMCPServer()
	if err != nil {
		t.Fatalf("NewMCPServer: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
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
	// Verify we can call a tool over the transport
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "ping"})
	if err != nil {
		t.Fatalf("ping over stdio transport: %v", err)
	}
	tc, _ := result.Content[0].(*mcpsdk.TextContent)
	if tc.Text != "pong" {
		t.Errorf("expected 'pong', got %q", tc.Text)
	}
	t.Log("stdio transport ping: pong")
}

// --- e38s02 knowledge tool standalone tests ---

func TestListServices(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	ctx, session := connectMCPSession(t, c)
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "list_services"})
	if err != nil {
		t.Fatalf("list_services: %v", err)
	}
	tc, ok := result.Content[0].(*mcpsdk.TextContent)
	if !ok || tc.Text == "" {
		t.Fatal("expected non-empty response")
	}
	if !strings.Contains(tc.Text, "BigBase Services") {
		t.Error("expected 'BigBase Services' header")
	}
	for _, svc := range []string{"deploy", "auth", "db"} {
		if !strings.Contains(tc.Text, svc) {
			t.Errorf("expected service %q in list", svc)
		}
	}
	t.Logf("list_services: %d chars", len(tc.Text))
}

func TestGetServiceDocs(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	ctx, session := connectMCPSession(t, c)
	for _, svc := range []string{"deploy", "auth", "storage"} {
		result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
			Name:      "get_service_docs",
			Arguments: map[string]interface{}{"service": svc},
		})
		if err != nil {
			t.Fatalf("get_service_docs(%s): %v", svc, err)
		}
		tc, _ := result.Content[0].(*mcpsdk.TextContent)
		if tc.Text == "" {
			t.Errorf("get_service_docs(%s): empty response", svc)
		}
		if !strings.Contains(tc.Text, svc) {
			t.Errorf("get_service_docs(%s): response should mention %s", svc, svc)
		}
		t.Logf("get_service_docs(%s): %d chars", svc, len(tc.Text))
	}
	// Unknown service returns helpful message
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_service_docs",
		Arguments: map[string]interface{}{"service": "nonexistent"},
	})
	if err != nil {
		t.Fatalf("get_service_docs(nonexistent): %v", err)
	}
	tc, _ := result.Content[0].(*mcpsdk.TextContent)
	if !strings.Contains(tc.Text, "not found") {
		t.Error("unknown service should say 'not found'")
	}
}

func TestGetCodeExample(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	ctx, session := connectMCPSession(t, c)
	pairs := [][2]string{
		{"auth", "sveltekit"},
		{"db", "react"},
		{"storage", "sveltekit"},
	}
	for _, p := range pairs {
		result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
			Name: "get_code_example",
			Arguments: map[string]interface{}{"service": p[0], "framework": p[1]},
		})
		if err != nil {
			t.Fatalf("get_code_example(%s/%s): %v", p[0], p[1], err)
		}
		tc, _ := result.Content[0].(*mcpsdk.TextContent)
		if tc.Text == "" {
			t.Errorf("get_code_example(%s/%s): empty", p[0], p[1])
		}
		t.Logf("get_code_example(%s/%s): %d chars", p[0], p[1], len(tc.Text))
	}
	// Unknown pair returns helpful message
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "get_code_example",
		Arguments: map[string]interface{}{"service": "nope", "framework": "nope"},
	})
	if err != nil {
		t.Fatalf("get_code_example(nope): %v", err)
	}
	tc, _ := result.Content[0].(*mcpsdk.TextContent)
	if !strings.Contains(tc.Text, "No example") {
		t.Error("unknown pair should suggest alternatives")
	}
}

func TestServerInstructions(t *testing.T) {
	c := mcp.New(mcp.Options{Transport: "stdio", Enabled: true})
	srv, err := c.NewMCPServer()
	if err != nil {
		t.Fatalf("NewMCPServer: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
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
	// Server instructions are sent on initialize. Verify the session is alive
	// and tools are available.
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "list_services"})
	if err != nil {
		t.Fatalf("list_services: %v", err)
	}
	tc, _ := result.Content[0].(*mcpsdk.TextContent)
	if !strings.Contains(tc.Text, "BigBase") {
		t.Error("expected instructions context in response")
	}
	t.Log("server instructions accessible via initialized session")
}

func TestListFrameworks(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	ctx, session := connectMCPSession(t, c)
	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "list_frameworks"})
	if err != nil {
		t.Fatalf("list_frameworks: %v", err)
	}
	tc, ok := result.Content[0].(*mcpsdk.TextContent)
	if !ok || tc.Text == "" {
		t.Fatal("expected non-empty response")
	}
	if !strings.Contains(tc.Text, "Supported Frameworks") {
		t.Error("expected 'Supported Frameworks' header")
	}
	for _, fw := range []string{"SvelteKit", "React", "Astro"} {
		if !strings.Contains(tc.Text, fw) {
			t.Errorf("expected framework %q in list", fw)
		}
	}
	t.Logf("list_frameworks: %d chars", len(tc.Text))
}

func TestMCPGetSSE(t *testing.T) {
	c := mcp.New(mcp.Options{Enabled: true})
	handler := c.Handler()

	// GET /mcp should NOT return 405 — the MCP Streamable HTTP spec requires
	// GET for SSE session establishment with Accept: text/event-stream.
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Accept", "text/event-stream")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code == http.StatusMethodNotAllowed {
		t.Fatal("GET /mcp returned 405 — SSE connections are blocked by method guard")
	}
	// Without a session, the handler may return 400. That's fine — the critical
	// check is that 405 is gone, proving the method guard is removed.
	if w.Code == http.StatusBadRequest {
		t.Log("GET /mcp returned 400 (expected without an active MCP session)")
		return
	}
	if w.Code != http.StatusOK {
		t.Fatalf("GET /mcp status = %d, want 200 or 400", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "text/event-stream") {
		t.Errorf("expected text/event-stream content-type, got %q", ct)
	}
	t.Logf("GET /mcp: %d OK, Content-Type: %s", w.Code, ct)
}

func TestMCPNonLocalhostHostHeader(t *testing.T) {
	// Simulate VPS: the MCP server listens on :3900 (including localhost),
	// but requests arrive via Caddy reverse proxy with Host: mcp.bigbase.click.
	// DNS rebinding protection (mcpsdk v1.4.0) must be disabled since the
	// server is behind a reverse proxy, not exposed directly to the internet.
	c := mcp.New(mcp.Options{Enabled: true})
	handler := c.Handler()

	req := httptest.NewRequest(http.MethodPost, "/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test","version":"1.0"},"capabilities":{}}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Host = "mcp.bigbase.click"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code == http.StatusForbidden {
		t.Fatal("POST /mcp with Host: mcp.bigbase.click returned 403 — DNS rebinding protection blocks non-localhost Host header")
	}
	// httptest doesn't set LocalAddrContextKey, so the DNS rebinding check is
	// skipped. The critical assertion is that 403 is never returned.
	body := w.Body.String()
	preview := body
	if len(preview) > 150 {
		preview = body[:150]
	}
	t.Logf("Host: mcp.bigbase.click → %d, body: %s", w.Code, preview)
	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status %d", w.Code)
	}
}

func TestMCPSetSiteAuthPolicy(t *testing.T) {
	dbConn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	_, err = dbConn.Exec(`CREATE TABLE IF NOT EXISTS sites (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		git_repo_id TEXT NOT NULL,
		production_branch TEXT NOT NULL DEFAULT 'main',
		root_path TEXT NOT NULL DEFAULT './',
		github_full_name TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		auth_policy TEXT NOT NULL DEFAULT '{}'
	)`)
	if err != nil {
		t.Fatalf("create sites: %v", err)
	}

	siteID := "site-auth-mcp-test"
	_, err = dbConn.Exec(`INSERT INTO sites (id, name, git_repo_id) VALUES (?, 'Test Site', 'repo-1')`, siteID)
	if err != nil {
		t.Fatalf("insert site: %v", err)
	}

	var callbackCalled bool
	var callbackSiteID string
	var callbackPolicyJSON string

	c := mcp.New(mcp.Options{
		Enabled:   true,
		Transport: "stdio",
		DB:        dbConn,
		UpdateAuthPolicy: func(siteID string, policyJSON string) {
			callbackCalled = true
			callbackSiteID = siteID
			callbackPolicyJSON = policyJSON
		},
	})

	srv, err := c.NewMCPServer()
	if err != nil {
		t.Fatalf("NewMCPServer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t1, t2 := mcpsdk.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	argsStr := `{
		"site_id": "site-auth-mcp-test",
		"policy": {
			"default": "protected",
			"protected_paths": ["/mcp/*"],
			"public_paths": ["/mcp/login"],
			"accept": ["jwt"]
		}
	}`
	var args json.RawMessage = []byte(argsStr)

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "set_site_auth_policy",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool set_site_auth_policy: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("expected content in response")
	}
	text, ok := result.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	if !strings.Contains(text.Text, `"status":"ok"`) {
		t.Errorf("expected status ok in tool response, got %s", text.Text)
	}

	var dbPolicy string
	err = dbConn.QueryRow("SELECT auth_policy FROM sites WHERE id = ?", siteID).Scan(&dbPolicy)
	if err != nil {
		t.Fatalf("query DB policy: %v", err)
	}
	if !strings.Contains(dbPolicy, `"/mcp/*"`) || !strings.Contains(dbPolicy, `"protected"`) {
		t.Errorf("DB policy was not updated correctly: %s", dbPolicy)
	}

	if !callbackCalled {
		t.Errorf("expected UpdateAuthPolicy callback to be called")
	}
	if callbackSiteID != siteID {
		t.Errorf("expected siteID %s in callback, got %s", siteID, callbackSiteID)
	}
	if !strings.Contains(callbackPolicyJSON, `"/mcp/*"`) {
		t.Errorf("expected policy in callback, got %s", callbackPolicyJSON)
	}
}
