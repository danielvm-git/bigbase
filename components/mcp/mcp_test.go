package mcp_test

import (
	"context"
	"database/sql"
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

	// Test MCP endpoint requires POST
	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /mcp: expected 405, got %d", w.Code)
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
		`CREATE TABLE git_repos (id TEXT PRIMARY KEY, name TEXT, description TEXT, updated_at TEXT)`,
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

func (m *mockDeployer) Trigger(_ context.Context, repoID, branch, siteName, siteID string) (*deploy.Deployment, error) {
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
