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
