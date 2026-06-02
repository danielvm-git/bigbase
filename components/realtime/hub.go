package realtime

import (
	"encoding/json"
	"sync"
)

type broadcastMsg struct {
	room string
	data []byte
}

type subRequest struct {
	client *Client
	room   string
}

// HubStats is a snapshot of hub state, returned by Hub.Stats().
type HubStats struct {
	TotalConnections int              `json:"total_connections"`
	TotalRooms       int              `json:"total_rooms"`
	Connections      []ConnectionInfo `json:"connections"`
}

// ConnectionInfo describes a single active WebSocket connection.
type ConnectionInfo struct {
	UserID int64    `json:"user_id"`
	Rooms  []string `json:"rooms"`
}

type statsReq struct {
	result chan HubStats
}

type Hub struct {
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool

	register    chan *Client
	unregister  chan *Client
	subscribe   chan subRequest
	unsubscribe chan subRequest
	broadcast   chan broadcastMsg
	stats       chan statsReq
	stop        chan struct{}
	stopOnce    sync.Once
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		rooms:       make(map[string]map[*Client]bool),
		register:    make(chan *Client, 256),
		unregister:  make(chan *Client, 256),
		subscribe:   make(chan subRequest, 256),
		unsubscribe: make(chan subRequest, 256),
		broadcast:   make(chan broadcastMsg, 256),
		stats:       make(chan statsReq),
		stop:        make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			h.unregisterClient(client)

		case req := <-h.subscribe:
			if h.rooms[req.room] == nil {
				h.rooms[req.room] = make(map[*Client]bool)
			}
			h.rooms[req.room][req.client] = true
			req.client.rooms[req.room] = true

		case req := <-h.unsubscribe:
			if members, ok := h.rooms[req.room]; ok {
				delete(members, req.client)
				if len(members) == 0 {
					delete(h.rooms, req.room)
				}
			}
			delete(req.client.rooms, req.room)

		case msg := <-h.broadcast:
			if members, ok := h.rooms[msg.room]; ok {
				for client := range members {
					select {
					case client.send <- msg.data:
					default:
						h.unregisterClient(client)
					}
				}
			}

		case <-h.stop:
			for client := range h.clients {
				close(client.send)
			}
			h.clients = nil
			h.rooms = nil
			return

		case req := <-h.stats:
			s := HubStats{
				TotalConnections: len(h.clients),
				TotalRooms:       len(h.rooms),
				Connections:      make([]ConnectionInfo, 0, len(h.clients)),
			}
			for client := range h.clients {
				rooms := make([]string, 0, len(client.rooms))
				for room := range client.rooms {
					rooms = append(rooms, room)
				}
				s.Connections = append(s.Connections, ConnectionInfo{
					UserID: client.userID,
					Rooms:  rooms,
				})
			}
			req.result <- s
		}
	}
}

func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.clients[client]; !ok {
		return
	}
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

func (h *Hub) Stop() {
	h.stopOnce.Do(func() {
		close(h.stop)
	})
}

func (h *Hub) Broadcast(channel string, data map[string]any) {
	msg, err := json.Marshal(data)
	if err != nil {
		return
	}
	room := channelPrefix + channel
	select {
	case h.broadcast <- broadcastMsg{room: room, data: msg}:
	default:
	}
}

func (h *Hub) Subscribe(client *Client, room string) {
	h.subscribe <- subRequest{client: client, room: room}
}

func (h *Hub) Unsubscribe(client *Client, room string) {
	h.unsubscribe <- subRequest{client: client, room: room}
}

// Stats returns a snapshot of current hub state: active connections,
// rooms, and per-connection subscription details.
func (h *Hub) Stats() HubStats {
	req := statsReq{result: make(chan HubStats)}
	h.stats <- req
	return <-req.result
}
