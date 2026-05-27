package realtime_test

import (
	"encoding/json"
	"errors"
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
		"action": "mutation",
		"channel": "collection:posts",
		"type":   "create",
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
