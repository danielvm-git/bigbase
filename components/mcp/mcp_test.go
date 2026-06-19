package mcp_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielvm/bigbase/components/mcp"
	"github.com/danielvm/bigbase/kernel"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
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
	defer session.Close()

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
	resp.Body.Close()
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
	defer session.Close()

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

func TestDeployTools(t *testing.T) {
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
	defer session.Close()

	// deploy_guide is pure knowledge — should always work
	t.Run("deploy_guide", func(t *testing.T) {
		result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "deploy_guide"})
		if err != nil {
			t.Fatalf("deploy_guide: %v", err)
		}
		tc, _ := result.Content[0].(*mcpsdk.TextContent)
		if tc.Text == "" {
			t.Error("expected non-empty guide")
		}
		t.Logf("deploy_guide: %d chars", len(tc.Text))
	})

	// list_repos, deploy_site, get_deploy_status, get_deploy_logs
	// return friendly messages when DB not connected
	for _, tool := range []string{"list_repos", "get_deploy_status", "get_deploy_logs"} {
		t.Run(tool+"_no_db", func(t *testing.T) {
			result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: tool})
			if err != nil {
				t.Fatalf("%s: %v", tool, err)
			}
			tc, _ := result.Content[0].(*mcpsdk.TextContent)
			// Should return a helpful message, not crash
			if tc.Text == "" {
				t.Errorf("%s: expected message", tool)
			}
		})
	}

	t.Run("deploy_site_no_db", func(t *testing.T) {
		result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
			Name:      "deploy_site",
			Arguments: map[string]interface{}{"repo_id": "test"},
		})
		if err != nil {
			t.Fatalf("deploy_site: %v", err)
		}
		tc, _ := result.Content[0].(*mcpsdk.TextContent)
		if tc.Text == "" {
			t.Error("deploy_site: expected message")
		}
	})
}
