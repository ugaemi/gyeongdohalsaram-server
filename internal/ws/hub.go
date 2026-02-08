package ws

import (
	"log/slog"
	"sync"
)

// Hub maintains the set of active clients and routes messages.
type Hub struct {
	Clients    map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	Incoming   chan *ClientMessage
	mu         sync.RWMutex

	// OnMessage is called for each incoming client message.
	OnMessage func(cm *ClientMessage)
	// OnDisconnect is called when a client disconnects.
	OnDisconnect func(client *Client)
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Incoming:   make(chan *ClientMessage, 256),
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
			slog.Info("client connected", "client", client.ID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			slog.Info("client disconnected", "client", client.ID)
			if h.OnDisconnect != nil {
				h.OnDisconnect(client)
			}

		case cm := <-h.Incoming:
			if h.OnMessage != nil {
				h.OnMessage(cm)
			}
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.Clients {
		select {
		case client.Send <- data:
		default:
			slog.Warn("broadcast: client send buffer full", "client", client.ID)
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.Clients)
}
