package deploy_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestStreamingLogs(t *testing.T) {
	_, handler, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "stream-logs-repo", gitDir)

	// Create a deployment
	var buf strings.Builder
	_ = json.NewEncoder(&buf).Encode(map[string]string{"repo_id": repoID})
	req := httptest.NewRequest("POST", "/api/deploy", strings.NewReader(buf.String()))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var dep map[string]any
	if err := json.NewDecoder(w.Body).Decode(&dep); err != nil {
		t.Fatalf("decode: %v", err)
	}
	depID := dep["id"].(string)

	// Start a real HTTP server for WebSocket upgrade
	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/deploy/" + depID + "/logs/stream"

	// Connect WebSocket before deployment finishes processing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Wait for deployment to reach terminal state (generates log lines)
	waitForDeploymentTerminal(t, handler, depID, 10*time.Second)

	// Read log lines from WebSocket (with a short timeout for last messages)
	_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	var lines []string
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		lines = append(lines, string(msg))
	}

	if len(lines) == 0 {
		t.Fatal("expected at least one log line via WebSocket, got none")
	}
	t.Logf("received %d log lines via WebSocket", len(lines))

	// Verify we got meaningful content — at minimum a status or clone mention
	foundContent := false
	logText := strings.Join(lines, "\n")
	for _, keyword := range []string{"Status:", "clone", "Clone", "→"} {
		if strings.Contains(logText, keyword) {
			foundContent = true
			break
		}
	}
	if !foundContent {
		t.Fatalf("no expected keywords found in WebSocket log stream. Lines: %v", lines)
	}
}

func TestLogLifecycle(t *testing.T) {
	_, handler, database, gitDir := setupDeploy(t)
	repoID := createTestRepo(t, database, "log-lifecycle-repo", gitDir)

	// Create a deployment
	var buf strings.Builder
	_ = json.NewEncoder(&buf).Encode(map[string]string{"repo_id": repoID})
	req := httptest.NewRequest("POST", "/api/deploy", strings.NewReader(buf.String()))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var dep map[string]any
	if err := json.NewDecoder(w.Body).Decode(&dep); err != nil {
		t.Fatalf("decode: %v", err)
	}
	depID := dep["id"].(string)

	// Start a real HTTP server for WebSocket upgrade
	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/deploy/" + depID + "/logs/stream"

	// Connect WebSocket before deployment finishes processing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Wait for deployment to reach terminal state
	waitForDeploymentTerminal(t, handler, depID, 10*time.Second)

	// Read all streaming log lines
	_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var lines []string
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		lines = append(lines, string(msg))
	}

	if len(lines) == 0 {
		t.Fatal("expected log lines via WebSocket during lifecycle, got none")
	}
	t.Logf("lifecycle log lines (%d):", len(lines))
	for _, line := range lines {
		t.Logf("  %s", line)
	}

	// Verify lifecycle phases appear in the stream
	logText := strings.Join(lines, "\n")
	phases := []string{"building", "Clone", "Deployed at"}
	for _, phase := range phases {
		if !strings.Contains(logText, phase) {
			t.Fatalf("expected lifecycle phase %q in log stream, got: %s", phase, logText)
		}
	}
}
