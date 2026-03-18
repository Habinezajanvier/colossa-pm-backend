package chat

import (
	"encoding/json"
	"sync"

	applogger "colossa-pm/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WSMessage is the envelope sent over WebSocket
type WSMessage struct {
	Type    string      `json:"type"` // "message", "reaction", "typing"
	Payload interface{} `json:"payload"`
}

// client represents a single WebSocket connection
type client struct {
	userID uuid.UUID
	conn   *websocket.Conn
	send   chan []byte
}

// Hub manages all active WebSocket connections grouped by conversationID
type Hub struct {
	mu        sync.RWMutex
	rooms     map[uuid.UUID]map[*client]bool // conversationID → set of clients
	join      chan *roomJoin
	leave     chan *roomLeave
	broadcast chan *roomMessage
}

type roomJoin struct {
	conversationID uuid.UUID
	client         *client
}

type roomLeave struct {
	conversationID uuid.UUID
	client         *client
}

type roomMessage struct {
	conversationID uuid.UUID
	payload        []byte
	exclude        *client // optionally exclude the sender
}

var globalHub *Hub
var hubOnce sync.Once

// GetHub returns the singleton Hub
func GetHub() *Hub {
	hubOnce.Do(func() {
		globalHub = &Hub{
			rooms:     make(map[uuid.UUID]map[*client]bool),
			join:      make(chan *roomJoin, 256),
			leave:     make(chan *roomLeave, 256),
			broadcast: make(chan *roomMessage, 256),
		}
		go globalHub.run()
	})
	return globalHub
}

func (h *Hub) run() {
	for {
		select {
		case j := <-h.join:
			h.mu.Lock()
			if _, ok := h.rooms[j.conversationID]; !ok {
				h.rooms[j.conversationID] = make(map[*client]bool)
			}
			h.rooms[j.conversationID][j.client] = true
			h.mu.Unlock()

		case l := <-h.leave:
			h.mu.Lock()
			if room, ok := h.rooms[l.conversationID]; ok {
				delete(room, l.client)
				if len(room) == 0 {
					delete(h.rooms, l.conversationID)
				}
			}
			h.mu.Unlock()
			close(l.client.send)

		case msg := <-h.broadcast:
			h.mu.RLock()
			room := h.rooms[msg.conversationID]
			h.mu.RUnlock()

			for c := range room {
				if c == msg.exclude {
					continue
				}
				select {
				case c.send <- msg.payload:
				default:
					// Client too slow — drop and disconnect
					h.leave <- &roomLeave{conversationID: msg.conversationID, client: c}
				}
			}
		}
	}
}

// Broadcast sends a payload to all clients in a conversation
func (h *Hub) Broadcast(conversationID uuid.UUID, msgType string, payload interface{}, exclude *client) {
	data, err := json.Marshal(WSMessage{Type: msgType, Payload: payload})
	if err != nil {
		applogger.Instance().ErrorMsg("ws broadcast marshal error: " + err.Error())
		return
	}
	h.broadcast <- &roomMessage{conversationID: conversationID, payload: data, exclude: exclude}
}

// writePump pumps messages from the send channel to the WebSocket
func (c *client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
