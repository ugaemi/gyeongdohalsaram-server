package room

import (
	"log/slog"
	"sync"
	"time"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/game"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

// Room represents a game room with players and state.
type Room struct {
	Code    string                  `json:"code"`
	State   game.RoomState          `json:"state"`
	Players map[string]*game.Player `json:"players"`
	HostID  string                  `json:"host_id"`

	// Client mapping: player ID -> ws client
	clients map[string]*ws.Client

	// Map objects generated at game start
	MapObjects []game.MapObject `json:"-"`

	// Game loop control
	stopCh        chan struct{}
	remainingTime time.Duration

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

// SetPlayerReady sets a player's ready status and returns whether all players are ready.
// This must be used instead of setting Ready directly to avoid race conditions.
func (r *Room) SetPlayerReady(playerID string, ready bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if p, ok := r.Players[playerID]; ok {
		p.Ready = ready
	}

	return r.allReady()
}

// allReady checks if all players are ready and team composition is valid.
// Caller must hold r.mu.
func (r *Room) allReady() bool {
	if len(r.Players) < game.MinPlayers {
		return false
	}

	policeCount := 0
	thiefCount := 0
	for _, p := range r.Players {
		if !p.Ready || p.Role == game.RoleNone {
			return false
		}
		switch p.Role {
		case game.RolePolice:
			policeCount++
		case game.RoleThief:
			thiefCount++
		}
	}

	return policeCount >= 1 && thiefCount >= 1
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

// PrepareGame assigns spawn positions and transitions to playing state.
// Must be called before broadcasting game_start so clients receive correct positions.
func (r *Room) PrepareGame() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.State = game.StatePlaying
	r.remainingTime = game.GameDuration
	r.stopCh = make(chan struct{})

	// Generate and apply spawn positions
	players := make([]*game.Player, 0, len(r.Players))
	for _, p := range r.Players {
		players = append(players, p)
	}
	positions := game.GenerateSpawnPositions(players)
	for id, pos := range positions {
		r.Players[id].SetPosition(pos.X, pos.Y)
	}

	// Generate map objects so all clients see the same map
	r.MapObjects = game.GenerateMapObjects()

	slog.Info("game prepared", "room", r.Code, "players", len(r.Players), "objects", len(r.MapObjects))
}

// StartGameLoop starts the game tick loop. Must be called after PrepareGame and broadcasting game_start.
func (r *Room) StartGameLoop() {
	go r.gameLoop()
}

// StopGame stops the game loop and transitions to ended state.
func (r *Room) StopGame(result game.WinResult) {
	r.mu.Lock()

	if r.State != game.StatePlaying {
		r.mu.Unlock()
		return
	}

	r.State = game.StateEnded

	// Signal the game loop to stop
	select {
	case <-r.stopCh:
		// Already closed
	default:
		close(r.stopCh)
	}

	r.mu.Unlock()

	// Broadcast game over
	msg, _ := ws.NewMessage(ws.TypeGameOver, gameOverMessage{
		Winner: result.String(),
	})
	r.BroadcastMessage(msg)

	slog.Info("game ended", "room", r.Code, "winner", result.String())
}

// RemainingTime returns the remaining game time.
func (r *Room) RemainingTime() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.remainingTime
}

type gameOverMessage struct {
	Winner string `json:"winner"`
}

type gameStateMessage struct {
	RemainingTime float64            `json:"remaining_time"`
	Players       []playerStateEntry `json:"players"`
}

type playerStateEntry struct {
	ID          string  `json:"id"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	State       string  `json:"state"`
	Role        string  `json:"role"`
	ArrestGauge float64 `json:"arrest_gauge"`
	RescueGauge float64 `json:"rescue_gauge"`
}

// gameLoop runs the game tick loop at TickRate frequency.
func (r *Room) gameLoop() {
	ticker := time.NewTicker(game.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.mu.Lock()
			r.remainingTime -= game.TickInterval
			timerExpired := r.remainingTime <= 0
			dt := game.TickInterval.Seconds()

			playerList := make([]*game.Player, 0, len(r.Players))
			for _, p := range r.Players {
				playerList = append(playerList, p)
			}

			// --- Invincibility timer ---
			for _, p := range playerList {
				if p.IsInvincible() {
					p.InvincibleTimer -= game.TickInterval
					if p.InvincibleTimer <= 0 {
						p.State = game.StateFree
						p.InvincibleTimer = 0
					}
				}
			}

			// --- Arrest mechanics (cumulative gauge) ---
			arrestPairs := game.FindArrestPairs(playerList)
			// Track which thieves are being chased this tick
			chasedThieves := make(map[string]bool)
			for _, pair := range arrestPairs {
				thief := pair[1]
				chasedThieves[thief.ID] = true
				thief.ArrestGauge += dt
				if thief.ArrestGauge >= game.ArrestDuration {
					thief.Arrest()
					slog.Info("thief arrested", "thief", thief.ID, "room", r.Code)
				}
			}
			// Gauge is cumulative — do NOT reset when out of range

			// --- Rescue mechanics (continuous gauge) ---
			rescueCandidates := game.FindJailRescueCandidates(playerList, game.JailX, game.JailY)
			rescuingThieves := make(map[string]bool)
			for _, thief := range rescueCandidates {
				rescuingThieves[thief.ID] = true
				thief.RescueGauge += dt
				if thief.RescueGauge >= game.RescueDuration {
					// Release all arrested thieves
					for _, p := range playerList {
						if p.Role == game.RoleThief && p.IsArrested() {
							p.Release()
							slog.Info("thief rescued", "thief", p.ID, "room", r.Code)
						}
					}
					thief.RescueGauge = 0
				}
			}
			// Reset rescue gauge for thieves NOT near jail (continuous requirement)
			for _, p := range playerList {
				if p.Role == game.RoleThief && p.IsFree() && !rescuingThieves[p.ID] {
					p.RescueGauge = 0
				}
			}

			// Build game state snapshot (after processing mechanics)
			players := make([]playerStateEntry, 0, len(r.Players))
			for _, p := range playerList {
				players = append(players, playerStateEntry{
					ID:          p.ID,
					X:           p.X,
					Y:           p.Y,
					State:       p.State.String(),
					Role:        p.Role.String(),
					ArrestGauge: p.ArrestGauge,
					RescueGauge: p.RescueGauge,
				})
			}

			remaining := r.remainingTime.Seconds()
			if remaining < 0 {
				remaining = 0
			}
			r.mu.Unlock()

			// Broadcast game state
			msg, _ := ws.NewMessage(ws.TypeGameState, gameStateMessage{
				RemainingTime: remaining,
				Players:       players,
			})
			r.BroadcastMessage(msg)

			// Check win conditions
			if game.CheckPoliceWin(playerList) {
				r.StopGame(game.WinPolice)
				return
			}
			if timerExpired {
				r.StopGame(game.WinThief)
				return
			}
		}
	}
}
