package lobby

import "sync"

// Hub maintains active websocket connections grouped by room.
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[Connection]struct{}
}

// Connection represents the subset of websocket connection capabilities used by the hub.
type Connection interface {
	WriteMessage(opcode int, payload []byte) error
	Close() error
}

// NewHub constructs a ready-to-use Hub.
func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[Connection]struct{}),
	}
}

// Register adds a connection to the given room and returns a cleanup function.
func (h *Hub) Register(roomID string, conn Connection) func() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.rooms[roomID]; !ok {
		h.rooms[roomID] = make(map[Connection]struct{})
	}
	h.rooms[roomID][conn] = struct{}{}

	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		clients, ok := h.rooms[roomID]
		if !ok {
			return
		}
		delete(clients, conn)
		if len(clients) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

// Broadcast sends the payload to all clients in the room except the sender.
func (h *Hub) Broadcast(roomID string, payload []byte, sender Connection) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}

	for conn := range clients {
		if conn == sender {
			continue
		}
		_ = conn.WriteMessage(1, payload)
	}
}
