package realtime

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/danielvm/bigbase/kernel"
	"github.com/gorilla/websocket"
)

const (
	version          = "0.1.0"
	writeWait        = 10 * time.Second
	pongWait         = 60 * time.Second
	pingPeriod       = (pongWait * 9) / 10
	maxMessageSize   = 4096
	sendBufSize      = 256
	channelPrefix    = "collection:"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}

type Options struct {
	Logger   Logger
	Validate func(token string) (int64, error)
}

type Realtime struct {
	logger   Logger
	hub      *Hub
	validate func(token string) (int64, error)
}

func New(opts Options) *Realtime {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	validate := opts.Validate
	if validate == nil {
		validate = func(token string) (int64, error) {
			return 0, nil
		}
	}
	hub := NewHub()
	go hub.Run()
	return &Realtime{
		logger:   logger,
		hub:      hub,
		validate: validate,
	}
}

func (r *Realtime) Name() string                    { return "realtime" }
func (r *Realtime) Version() string                 { return version }
func (r *Realtime) Dependencies() []string          { return []string{"auth", "api"} }
func (r *Realtime) ConfigSchema() json.RawMessage   { return nil }
func (r *Realtime) Hooks() []kernel.HookDef         { return nil }

func (r *Realtime) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (r *Realtime) Start(ctx *kernel.Context) error {
	bus := ctx.Kernel.EventBus()
	bus.Subscribe(kernel.HookDef{
		Name:     "mutation",
		Priority: 0,
		Handler: func(_ *kernel.Context, event kernel.Event) error {
			channel, _ := event.Data["collection"].(string)
			mutType, _ := event.Data["type"].(string)
			r.hub.Broadcast(channel, map[string]any{
				"action": "mutation",
				"channel": channelPrefix + channel,
				"type":   mutType,
			})
			return nil
		},
	})

	r.logger.Info("realtime component ready")
	return nil
}

func (r *Realtime) Stop(ctx *kernel.Context) error {
	r.hub.Stop()
	return nil
}

func (r *Realtime) Hub() *Hub {
	return r.hub
}

func (r *Realtime) Handler() http.Handler {
	return http.HandlerFunc(r.serveWS)
}

func (r *Realtime) serveWS(w http.ResponseWriter, req *http.Request) {
	token := req.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	userID, err := r.validate(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		r.logger.Error("websocket upgrade", "error", err)
		return
	}

	client := &Client{
		hub:    r.hub,
		conn:   conn,
		send:   make(chan []byte, sendBufSize),
		userID: userID,
		rooms:  make(map[string]bool),
	}

	r.hub.register <- client
	go client.writePump()
	go client.readPump()
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				for room := range client.rooms {
					if members, ok := h.rooms[room]; ok {
						delete(members, client)
						if len(members) == 0 {
							delete(h.rooms, room)
						}
					}
				}
				close(client.send)
			}
			h.mu.Unlock()

		case <-h.stop:
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
			}
			h.clients = nil
			h.rooms = nil
			h.mu.Unlock()
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.stop)
}

func (h *Hub) Broadcast(channel string, data map[string]any) {
	msg, err := json.Marshal(data)
	if err != nil {
		return
	}

	h.mu.RLock()
	room := channelPrefix + channel
	if members, ok := h.rooms[room]; ok {
		for client := range members {
			select {
			case client.send <- msg:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) Subscribe(client *Client, room string) {
	h.mu.Lock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Client]bool)
	}
	h.rooms[room][client] = true
	client.rooms[room] = true
	h.mu.Unlock()
}

func (h *Hub) Unsubscribe(client *Client, room string) {
	h.mu.Lock()
	if members, ok := h.rooms[room]; ok {
		delete(members, client)
		if len(members) == 0 {
			delete(h.rooms, room)
		}
	}
	delete(client.rooms, room)
	h.mu.Unlock()
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID int64
	rooms  map[string]bool
}

type wsMessage struct {
	Action  string `json:"action"`
	Channel string `json:"channel"`
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msgBytes, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg wsMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		switch msg.Action {
		case "subscribe":
			c.hub.Subscribe(c, msg.Channel)
		case "unsubscribe":
			c.hub.Unsubscribe(c, msg.Channel)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
