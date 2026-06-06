// Package lamp provides the "Smart Lamp Prototype" hackathon demo: a tiny
// WebSocket relay plus two self-contained phone pages (a luminance sensor and a
// street-lamp screen). It is intentionally infrastructure-only — there is no
// domain or persistence concern, just a real-time fan-out of opaque JSON
// messages between phones sharing a room.
package lamp

import (
	"log/slog"
	"sync"
)

// client is a single connected phone (sensor or screen) in a room.
type client struct {
	id   string
	room string
	role string
	send chan []byte
}

// Hub fans out messages between clients grouped by room. It is safe for
// concurrent use. The relay is deliberately dumb: whatever a client sends is
// forwarded verbatim to every other client in the same room.
type Hub struct {
	log *slog.Logger

	mu    sync.RWMutex
	rooms map[string]map[*client]struct{}
}

// NewHub creates an empty relay hub.
func NewHub(log *slog.Logger) *Hub {
	return &Hub{
		log:   log,
		rooms: make(map[string]map[*client]struct{}),
	}
}

// add registers a client in its room.
func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[c.room] == nil {
		h.rooms[c.room] = make(map[*client]struct{})
	}
	h.rooms[c.room][c] = struct{}{}
	if h.log != nil {
		h.log.Info("lamp client connected", "room", c.room, "role", c.role, "peers", len(h.rooms[c.room]))
	}
}

// remove deregisters a client and closes its send channel.
func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	peers, ok := h.rooms[c.room]
	if !ok {
		return
	}
	if _, ok := peers[c]; ok {
		delete(peers, c)
		close(c.send)
	}
	if len(peers) == 0 {
		delete(h.rooms, c.room)
	}
	if h.log != nil {
		h.log.Info("lamp client disconnected", "room", c.room, "role", c.role)
	}
}

// broadcast forwards payload to every client in room except the sender. A slow
// client that cannot keep up is dropped rather than blocking the whole room —
// fine for a demo where only the latest luminance value matters.
func (h *Hub) broadcast(room string, sender *client, payload []byte) {
	h.mu.RLock()
	peers := h.rooms[room]
	targets := make([]*client, 0, len(peers))
	for c := range peers {
		if c != sender {
			targets = append(targets, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.send <- payload:
		default:
			// Drop the message for this backed-up client.
		}
	}
}
