package realtime

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/danielvm/bigbase/kernel"
)

const (
	version        = "0.1.0"
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	sendBufSize    = 256
	channelPrefix  = "collection:"
)

type Options struct {
	Logger          kernel.Logger
	Validate        func(token string) (int64, error)
	AllowedOrigins  []string // CORS-aligned WebSocket origin allowlist
}

type Realtime struct {
	logger          kernel.Logger
	hub             *Hub
	validate        func(token string) (int64, error)
	allowedOrigins  []string
	upgrader        websocket.Upgrader
	unsubscribe     func()
}

var _ kernel.Component = (*Realtime)(nil)

func New(opts Options) *Realtime {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	validate := opts.Validate
	if validate == nil {
		validate = func(token string) (int64, error) {
			return 0, nil
		}
	}
	hub := NewHub()
	go hub.Run()
	rt := &Realtime{
		logger:         logger,
		hub:            hub,
		validate:       validate,
		allowedOrigins: append([]string(nil), opts.AllowedOrigins...),
	}
	rt.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     rt.checkOrigin,
	}
	return rt
}

// checkOrigin allows empty Origin (non-browser), configured allowlist entries,
// and same-origin. Cross-origin is denied when not explicitly allowed.
func (r *Realtime) checkOrigin(req *http.Request) bool {
	origin := req.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, o := range r.allowedOrigins {
		if o == origin {
			return true
		}
	}
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	return origin == scheme+"://"+req.Host
}

func (r *Realtime) Name() string                  { return "realtime" }
func (r *Realtime) Version() string               { return version }
func (r *Realtime) Dependencies() []string        { return []string{"auth", "api"} }

func (r *Realtime) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (r *Realtime) Start(ctx *kernel.Context) error {
	bus := ctx.Kernel.EventBus()
	r.unsubscribe = bus.Subscribe(kernel.HookDef{
		Name:     "mutation",
		Priority: 0,
		Handler: func(_ *kernel.Context, event kernel.Event) error {
			channel, _ := event.Data["collection"].(string)
			mutType, _ := event.Data["type"].(string)
			r.hub.Broadcast(channel, map[string]any{
				"action":  "mutation",
				"channel": channelPrefix + channel,
				"type":    mutType,
			})
			return nil
		},
	})
	r.logger.Info("realtime component ready")
	return nil
}

func (r *Realtime) Stop(ctx *kernel.Context) error {
	r.hub.Stop()
	if r.unsubscribe != nil {
		r.unsubscribe()
	}
	return nil
}

func (r *Realtime) Hub() *Hub {
	return r.hub
}

func (r *Realtime) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/realtime", r.serveWS)
	mux.HandleFunc("/api/realtime/status", r.handleStatus)
	return mux
}

func (r *Realtime) handleStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	stats := r.hub.Stats()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
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

	conn, err := r.upgrader.Upgrade(w, req, nil)
	if err != nil {
		r.logger.Error("websocket upgrade", "error", err)
		return
	}

	client := newClient(r.hub, conn, userID)
	r.hub.register <- client
	go client.writePump()
	go client.readPump()
}

func newClient(hub *Hub, conn *websocket.Conn, userID int64) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, sendBufSize),
		userID: userID,
		rooms:  make(map[string]bool),
	}
}
