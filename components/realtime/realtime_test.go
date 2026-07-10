package realtime_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/danielvm/bigbase/components/realtime"
)

func setupRealtime(t *testing.T) (*realtime.Realtime, *httptest.Server) {
	t.Helper()
	rt := realtime.New(realtime.Options{
		Validate: func(token string) (int64, error) {
			if token == "valid" {
				return 1, nil
			}
			return 0, errors.New("invalid token")
		},
	})
	server := httptest.NewServer(rt.Handler())
	t.Cleanup(server.Close)
	return rt, server
}

func dialWS(t *testing.T, server *httptest.Server, token string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/realtime?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readWS(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(msg, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return data
}

func TestRealtimeConnectInvalidToken(t *testing.T) {
	_, server := setupRealtime(t)
	_, _, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/realtime?token=bad",
		nil,
	)
	if err == nil {
		t.Fatal("expected connection error for invalid token")
	}
}

func TestRealtimeConnectNoToken(t *testing.T) {
	_, server := setupRealtime(t)
	_, _, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/realtime",
		nil,
	)
	if err == nil {
		t.Fatal("expected connection error for missing token")
	}
}

func TestRealtimeSubscribeAndBroadcast(t *testing.T) {
	rt, server := setupRealtime(t)
	conn := dialWS(t, server, "valid")

	sub := map[string]string{"action": "subscribe", "channel": "collection:posts"}
	subBytes, _ := json.Marshal(sub)
	if err := conn.WriteMessage(websocket.TextMessage, subBytes); err != nil {
		t.Fatalf("subscribe write: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	rt.Hub().Broadcast("posts", map[string]any{
		"action":  "mutation",
		"channel": "collection:posts",
		"type":    "create",
	})

	data := readWS(t, conn)
	if data["action"] != "mutation" {
		t.Fatalf("expected action 'mutation', got %v", data["action"])
	}
	if data["type"] != "create" {
		t.Fatalf("expected type 'create', got %v", data["type"])
	}
	if data["channel"] != "collection:posts" {
		t.Fatalf("expected channel 'collection:posts', got %v", data["channel"])
	}
}

func TestRealtimeUnsubscribe(t *testing.T) {
	rt, server := setupRealtime(t)
	conn := dialWS(t, server, "valid")

	sub := map[string]string{"action": "subscribe", "channel": "collection:posts"}
	subBytes, _ := json.Marshal(sub)
	_ = conn.WriteMessage(websocket.TextMessage, subBytes)
	time.Sleep(50 * time.Millisecond)

	unsub := map[string]string{"action": "unsubscribe", "channel": "collection:posts"}
	unsubBytes, _ := json.Marshal(unsub)
	_ = conn.WriteMessage(websocket.TextMessage, unsubBytes)
	time.Sleep(50 * time.Millisecond)

	rt.Hub().Broadcast("posts", map[string]any{
		"action": "mutation",
		"type":   "create",
	})

	_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err := conn.ReadMessage()
	if err == nil {
		t.Fatal("expected no message after unsubscribe")
	}
}

func TestRealtimeStop(t *testing.T) {
	rt, server := setupRealtime(t)
	conn := dialWS(t, server, "valid")

	sub := map[string]string{"action": "subscribe", "channel": "collection:posts"}
	subBytes, _ := json.Marshal(sub)
	_ = conn.WriteMessage(websocket.TextMessage, subBytes)
	time.Sleep(50 * time.Millisecond)

	rt.Hub().Broadcast("posts", map[string]any{
		"action": "mutation",
		"type":   "create",
	})
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	_, _, err := conn.ReadMessage()
	if err != nil {
		t.Fatal("expected message before stop")
	}

	_ = rt.Stop(nil)
}

func TestRealtimeStatusEndpoint(t *testing.T) {
	_, server := setupRealtime(t)

	// Connect a client and subscribe to a channel
	conn := dialWS(t, server, "valid")
	sub := map[string]string{"action": "subscribe", "channel": "collection:posts"}
	subBytes, _ := json.Marshal(sub)
	if err := conn.WriteMessage(websocket.TextMessage, subBytes); err != nil {
		t.Fatalf("subscribe write: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Call the status endpoint
	resp, err := server.Client().Get(server.URL + "/api/realtime/status")
	if err != nil {
		t.Fatalf("status request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var status struct {
		TotalConnections int `json:"total_connections"`
		TotalRooms       int `json:"total_rooms"`
		Connections      []struct {
			UserID int64    `json:"user_id"`
			Rooms  []string `json:"rooms"`
		} `json:"connections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if status.TotalConnections < 1 {
		t.Fatalf("expected at least 1 connection, got %d", status.TotalConnections)
	}
	if status.TotalRooms < 1 {
		t.Fatalf("expected at least 1 room, got %d", status.TotalRooms)
	}
	if len(status.Connections) < 1 {
		t.Fatal("expected at least 1 connection in list, got 0")
	}
	found := false
	for _, c := range status.Connections {
		if c.UserID == 1 {
			found = true
			if len(c.Rooms) < 1 {
				t.Fatal("expected at least 1 room for user 1")
			}
		}
	}
	if !found {
		t.Fatal("expected to find userID=1 in connections")
	}
}

func TestRealtimeStatusEmpty(t *testing.T) {
	_, server := setupRealtime(t)

	resp, err := server.Client().Get(server.URL + "/api/realtime/status")
	if err != nil {
		t.Fatalf("status request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var status struct {
		TotalConnections int `json:"total_connections"`
		TotalRooms       int `json:"total_rooms"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if status.TotalConnections != 0 {
		t.Fatalf("expected 0 connections, got %d", status.TotalConnections)
	}
	if status.TotalRooms != 0 {
		t.Fatalf("expected 0 rooms, got %d", status.TotalRooms)
	}
}

func TestRealtimeBroadcastOnlySubscribedChannel(t *testing.T) {
	rt, server := setupRealtime(t)
	conn := dialWS(t, server, "valid")

	sub := map[string]string{"action": "subscribe", "channel": "collection:posts"}
	subBytes, _ := json.Marshal(sub)
	_ = conn.WriteMessage(websocket.TextMessage, subBytes)
	time.Sleep(50 * time.Millisecond)

	rt.Hub().Broadcast("other", map[string]any{
		"action": "mutation",
		"type":   "create",
	})

	_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err := conn.ReadMessage()
	if err == nil {
		t.Fatal("expected no message for unsubscribed channel")
	}
}

func TestRealtimeCheckOrigin(t *testing.T) {
	rt := realtime.New(realtime.Options{
		AllowedOrigins: []string{"https://app.example.com"},
		Validate:       func(token string) (int64, error) { return 1, nil },
	})
	server := httptest.NewServer(rt.Handler())
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/realtime?token=valid"

	t.Run("allowed origin", func(t *testing.T) {
		hdr := http.Header{}
		hdr.Set("Origin", "https://app.example.com")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, hdr)
		if err != nil {
			t.Fatalf("allowed origin dial: %v", err)
		}
		_ = conn.Close()
	})

	t.Run("denied origin", func(t *testing.T) {
		hdr := http.Header{}
		hdr.Set("Origin", "https://evil.com")
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, hdr)
		if err == nil {
			t.Fatal("expected dial failure for evil origin")
		}
		if resp != nil && resp.StatusCode != http.StatusForbidden {
			// gorilla returns 403 on CheckOrigin false
			t.Logf("got status %d (err=%v)", resp.StatusCode, err)
		}
	})
}
