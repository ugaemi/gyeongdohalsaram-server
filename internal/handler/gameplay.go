package handler

import (
	"encoding/json"
	"log/slog"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/game"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/room"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

// GameplayHandler handles in-game messages.
type GameplayHandler struct {
	rm *room.Manager
}

// NewGameplayHandler creates a new gameplay handler.
func NewGameplayHandler(rm *room.Manager) *GameplayHandler {
	return &GameplayHandler{rm: rm}
}

type playerMoveRequest struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// HandlePlayerMove handles player movement updates.
func (h *GameplayHandler) HandlePlayerMove(client *ws.Client, msg ws.Message) {
	var req playerMoveRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		client.SendMessage(ws.NewErrorMessage("invalid move data"))
		return
	}

	// Validate position is within map bounds
	if req.X < 0 || req.X > game.MapWidth || req.Y < 0 || req.Y > game.MapHeight {
		client.SendMessage(ws.NewErrorMessage("position out of bounds"))
		return
	}

	// Find the player's room
	r := h.findRoomByClient(client)
	if r == nil {
		return
	}

	// Find player ID from room's clients
	playerID := h.findPlayerID(r, client)
	if playerID == "" {
		return
	}

	player := r.Players[playerID]
	if player == nil {
		return
	}

	player.SetPosition(req.X, req.Y)

	slog.Debug("player moved", "player", playerID, "x", req.X, "y", req.Y)
}

func (h *GameplayHandler) findRoomByClient(client *ws.Client) *room.Room {
	// Search through all rooms for this client
	// This is a simple implementation; can be optimized with a reverse lookup map
	return nil // Will be connected via the lobby handler's playerMap in later phases
}

func (h *GameplayHandler) findPlayerID(r *room.Room, client *ws.Client) string {
	for _, p := range r.GetPlayerList() {
		if r.GetClient(p.ID) == client {
			return p.ID
		}
	}
	return ""
}
