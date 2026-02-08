package room

import (
	"sync"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/game"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

// Room represents a game room with players and state.
type Room struct {
	Code    string            `json:"code"`
	State   game.RoomState    `json:"state"`
	Players map[string]*game.Player `json:"players"`
	HostID  string            `json:"host_id"`

	// Client mapping: player ID -> ws client
	clients map[string]*ws.Client

	mu sync.RWMutex
}

// NewRoom creates a new room with the given code.
func NewRoom(code string) *Room {
	return &Room{
		Code:    code,
		State:   game.StateWaiting,
		Players: make(map[string]*game.Player),
		clients: make(map[string]*ws.Client),
	}
}

// AddPlayer adds a player to the room. Returns false if the room is full.
func (r *Room) AddPlayer(player *game.Player, client *ws.Client) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.Players) >= game.MaxPlayers {
		return false
	}

	r.Players[player.ID] = player
	r.clients[player.ID] = client

	if len(r.Players) == 1 {
		r.HostID = player.ID
	}
	return true
}

// RemovePlayer removes a player from the room.
func (r *Room) RemovePlayer(playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.Players, playerID)
	delete(r.clients, playerID)

	// Transfer host if the host left
	if r.HostID == playerID && len(r.Players) > 0 {
		for id := range r.Players {
			r.HostID = id
			break
		}
	}
}

// PlayerCount returns the number of players.
func (r *Room) PlayerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Players)
}

// PoliceCount returns the number of police players.
func (r *Room) PoliceCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, p := range r.Players {
		if p.Role == game.RolePolice {
			count++
		}
	}
	return count
}

// CanSelectRole checks if a player can select the given role.
func (r *Room) CanSelectRole(role game.Role) bool {
	if role == game.RolePolice {
		return r.PoliceCount() < game.MaxPolice
	}
	return true
}

// AllReady checks if all players are ready and minimum players are met.
func (r *Room) AllReady() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.Players) < game.MinPlayers {
		return false
	}

	for _, p := range r.Players {
		if !p.Ready || p.Role == game.RoleNone {
			return false
		}
	}
	return true
}

// GetPlayerList returns a slice of all players.
func (r *Room) GetPlayerList() []*game.Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	players := make([]*game.Player, 0, len(r.Players))
	for _, p := range r.Players {
		players = append(players, p)
	}
	return players
}

// BroadcastMessage sends a message to all players in the room.
func (r *Room) BroadcastMessage(msg ws.Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, client := range r.clients {
		client.SendMessage(msg)
	}
}

// SendToPlayer sends a message to a specific player.
func (r *Room) SendToPlayer(playerID string, msg ws.Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if client, ok := r.clients[playerID]; ok {
		client.SendMessage(msg)
	}
}

// GetClient returns the WebSocket client for a player.
func (r *Room) GetClient(playerID string) *ws.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clients[playerID]
}

// IsEmpty returns true if the room has no players.
func (r *Room) IsEmpty() bool {
	return r.PlayerCount() == 0
}

// Reset resets the room state to waiting, preserving players and roles.
func (r *Room) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.State = game.StateWaiting
	for _, p := range r.Players {
		p.Reset()
	}
}
