package lamp

import (
	"embed"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

//go:embed static/*.html
var staticFS embed.FS

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8 * 1024
)

// Handler exposes the lamp demo: a WebSocket relay endpoint and the two phone
// pages. It has no dependency on the database, so the demo works even when the
// rest of the platform runs without Postgres/Redis.
type Handler struct {
	hub      *Hub
	log      *slog.Logger
	upgrader websocket.Upgrader
}

// NewHandler builds the lamp handler with a fresh relay hub.
func NewHandler(log *slog.Logger) *Handler {
	return &Handler{
		hub: NewHub(log),
		log: log,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Demo relay: any origin may connect (phones, tunnels, localhost).
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// ServeWS upgrades the connection and joins the client to a room. Query params:
//
//	room  - logical pairing key (default "demo")
//	role  - "sensor" or "screen" (informational only)
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	if room == "" {
		room = "demo"
	}
	role := r.URL.Query().Get("role")

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade already wrote an error response.
		return
	}

	c := &client{
		id:   uuid.NewString(),
		room: room,
		role: role,
		send: make(chan []byte, 16),
	}
	h.hub.add(c)

	go h.writePump(conn, c)
	h.readPump(conn, c)
}

// readPump reads messages from the phone and fans them out to the room. It owns
// closing/unregistering the client when the socket dies.
func (h *Handler) readPump(conn *websocket.Conn, c *client) {
	defer func() {
		h.hub.remove(c)
		_ = conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		mt, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if mt != websocket.TextMessage {
			continue
		}
		h.hub.broadcast(c.room, c, payload)
	}
}

// writePump drains the client's send channel and keeps the socket alive with
// periodic pings.
func (h *Handler) writePump(conn *websocket.Conn, c *client) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeIndex serves the landing page linking the two phone roles.
func (h *Handler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	h.servePage(w, "static/index.html")
}

// ServeSensor serves Phone A (camera luminance sensor).
func (h *Handler) ServeSensor(w http.ResponseWriter, r *http.Request) {
	h.servePage(w, "static/sensor.html")
}

// ServeScreen serves Phone B (street-lamp screen).
func (h *Handler) ServeScreen(w http.ResponseWriter, r *http.Request) {
	h.servePage(w, "static/screen.html")
}

func (h *Handler) servePage(w http.ResponseWriter, name string) {
	body, err := staticFS.ReadFile(name)
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(body)
}
