package ws

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 65536
)

// Client represents a single WebSocket connection managed by the Hub.
type Client struct {
	Hub   *Hub
	Conn  *websocket.Conn
	Send  chan Message
	ID    string
	Rooms map[string]bool
}

// NewClient creates a new Client with a generated ID.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		Hub:   hub,
		Conn:  conn,
		Send:  make(chan Message, 256),
		ID:    conn.RemoteAddr().String() + "-" + time.Now().Format("150405.000"),
		Rooms: make(map[string]bool),
	}
}

// ReadPump reads messages from the WebSocket connection and processes
// them. It runs in its own goroutine per client.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws: client %s read error: %v", c.ID, err)
			}
			break
		}

		switch msg.Type {
		case MsgPing:
			// Respond with pong
			c.Send <- Message{Type: MsgPong, TS: time.Now().UnixMilli()}
		case MsgQuery:
			// Process query via the hub's QueryHandler (if configured)
			if c.Hub.QueryHandler != nil {
				// Extract the query from payload — it could be a plain string
				// or a JSON object with a "query" field
				queryStr := string(msg.Payload)
				var queryPayload struct {
					Query string `json:"query"`
				}
				if err := json.Unmarshal(msg.Payload, &queryPayload); err == nil && queryPayload.Query != "" {
					queryStr = queryPayload.Query
				}

				// Run in a goroutine so we don't block the read pump
				go func(msgID string, rawQuery string) {
					ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
					defer cancel()

					result, err := c.Hub.QueryHandler.Query(ctx, rawQuery)
					if err != nil {
						c.Send <- NewErrorMessage(msgID, err.Error())
						return
					}
					c.Send <- NewResultMessage(msgID, result)
				}(msg.ID, queryStr)
			} else {
				log.Printf("ws: client %s query: %s (no handler configured)", c.ID, string(msg.Payload))
				// Echo back with a result placeholder
				c.Send <- Message{
					Type: MsgResult,
					ID:   msg.ID,
					TS:   time.Now().UnixMilli(),
				}
			}
		default:
			log.Printf("ws: client %s unknown message type: %s", c.ID, msg.Type)
		}
	}
}

// WritePump writes messages from the Send channel to the WebSocket
// connection. It runs in its own goroutine per client.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(msg); err != nil {
				log.Printf("ws: client %s write error: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
