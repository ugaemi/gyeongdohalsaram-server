package room

import (
	"log/slog"
	"math/rand"
	"sync"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/game"
)

// Manager manages all active rooms.
type Manager struct {
	rooms map[string]*Room // code -> room
	mu    sync.RWMutex
}

// NewManager creates a new room manager.
func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

// CreateRoom creates a new room and returns it.
func (m *Manager) CreateRoom() *Room {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing := make(map[string]bool, len(m.rooms))
	for code := range m.rooms {
		existing[code] = true
	}

	code := GenerateCode(existing)
	room := NewRoom(code)
	m.rooms[code] = room

	slog.Info("room created", "code", code)
	return room
}

// GetRoom returns a room by its code.
func (m *Manager) GetRoom(code string) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[code]
}

// RemoveRoom removes a room by its code.
func (m *Manager) RemoveRoom(code string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rooms, code)
	slog.Info("room removed", "code", code)
}

// RoomCount returns the number of active rooms.
func (m *Manager) RoomCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rooms)
}

// FindAvailableRoom returns a random room that is waiting and not full.
// If preferredRole is specified, it prefers rooms where that role is available.
func (m *Manager) FindAvailableRoom(preferredRole game.Role) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var available []*Room
	var preferred []*Room
	for _, r := range m.rooms {
		r.mu.RLock()
		isWaiting := r.State == game.StateWaiting
		hasSpace := len(r.Players) < game.MaxPlayers
		canSelect := preferredRole == game.RoleNone || r.canSelectRole(preferredRole)
		r.mu.RUnlock()
		if isWaiting && hasSpace {
			available = append(available, r)
			if canSelect {
				preferred = append(preferred, r)
			}
		}
	}

	if len(preferred) > 0 {
		return preferred[rand.Intn(len(preferred))]
	}
	if len(available) == 0 {
		return nil
	}
	return available[rand.Intn(len(available))]
}

// FindRoomByPlayerID finds the room containing a player.
func (m *Manager) FindRoomByPlayerID(playerID string) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, room := range m.rooms {
		room.mu.RLock()
		_, exists := room.Players[playerID]
		room.mu.RUnlock()
		if exists {
			return room
		}
	}
	return nil
}
