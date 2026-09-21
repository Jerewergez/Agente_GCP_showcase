// Package ws provides a WebSocket hub for real-time communication
// between the backend and Chrome Extension clients.
package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// QueryRunner is the interface the hub uses to process NL queries.
// Implemented by the query.Service in the internal/query package.
type QueryRunner interface {
	Query(ctx context.Context, rawQuery string) (interface{}, error)
}

// QueryRunnerFunc is an adapter that lets a regular function serve as
// a QueryRunner.
type QueryRunnerFunc func(ctx context.Context, rawQuery string) (interface{}, error)

// Query calls the underlying function.
func (f QueryRunnerFunc) Query(ctx context.Context, rawQuery string) (interface{}, error) {
	return f(ctx, rawQuery)
}

// MessageType defines the type of WebSocket message.
type MessageType string

const (
	// MsgQuery is a query request from a client.
	MsgQuery MessageType = "query"
	// MsgResult is a query result sent to a client.
	MsgResult MessageType = "result"
	// MsgError is an error message sent to a client.
	MsgError MessageType = "error"
	// MsgPing is a client ping (expects pong response).
	MsgPing MessageType = "ping"
	// MsgPong is a server pong response.
	MsgPong MessageType = "pong"
)

// Message represents a structured WebSocket message exchanged
// between the client and server.
type Message struct {
	Type    MessageType     `json:"type"`
	ID      string          `json:"id,omitempty"`
	Room    string          `json:"room,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
	TS      int64           `json:"ts,omitempty"`
}

// Hub manages all active WebSocket connections, per-client channels,
// and broadcast capabilities.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
	upgrader   websocket.Upgrader

	// QueryHandler processes NL→SQL queries from WebSocket clients.
	// When set, incoming MsgQuery messages are forwarded to this handler.
	QueryHandler QueryRunner
}

// NewHub creates and returns a new WebSocket Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan Message, 256),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true }, // CORS handled externally
		},
	}
}

// Run starts the hub's event loop. It must be called as a goroutine.
func (h *Hub) Run() {
	log.Println("ws: hub started")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("ws: client %s connected (%d total)", client.ID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				// Remove from all rooms
				for room := range client.Rooms {
					if roomClients, ok := h.rooms[room]; ok {
						delete(roomClients, client)
						if len(roomClients) == 0 {
							delete(h.rooms, room)
						}
					}
				}
				close(client.Send)
				log.Printf("ws: client %s disconnected (%d total)", client.ID, len(h.clients))
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- msg:
				default:
					// Client send buffer full; skip
					log.Printf("ws: dropping message for slow client %s", client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// HandleConnection upgrades an HTTP connection to WebSocket and
// registers the new client with the hub.
func (h *Hub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade error: %v", err)
		return
	}

	client := NewClient(h, conn)
	h.register <- client

	// Start read/write pumps in goroutines
	go client.WritePump()
	go client.ReadPump()
}

// BroadcastToAll sends a message to every connected client.
func (h *Hub) BroadcastToAll(msg Message) {
	h.broadcast <- msg
}

// BroadcastToRoom sends a message to all clients in a specific room.
func (h *Hub) BroadcastToRoom(room string, msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if roomClients, ok := h.rooms[room]; ok {
		for client := range roomClients {
			select {
			case client.Send <- msg:
			default:
				log.Printf("ws: dropping room message for slow client %s", client.ID)
			}
		}
	}
}

// ClientCount returns the number of currently connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Stats returns basic hub statistics for monitoring.
func (h *Hub) Stats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return map[string]interface{}{
		"clients": len(h.clients),
		"rooms":   len(h.rooms),
	}
}
