package handler

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/game"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/room"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

// GameplayHandler handles in-game messages.
type GameplayHandler struct {
	rm     *room.Manager
	router *Router
}

// NewGameplayHandler creates a new gameplay handler.
func NewGameplayHandler(rm *room.Manager, router *Router) *GameplayHandler {
	return &GameplayHandler{rm: rm, router: router}
}

type playerMoveRequest struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type playerMoveResponse struct {
	PlayerID string  `json:"player_id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

// HandlePlayerMove handles player movement updates.
func (h *GameplayHandler) HandlePlayerMove(client *ws.Client, msg ws.Message) {
	var req playerMoveRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		client.SendMessage(ws.NewErrorMessage("invalid move data"))
		return
	}

	// Clamp position within map bounds (accounting for player radius)
	req.X, req.Y = game.ClampPosition(req.X, req.Y)

	// Resolve player ID via Router
	playerID := h.router.GetPlayerID(client.ID)
	if playerID == "" {
		client.SendMessage(ws.NewErrorMessage("not in a room"))
		return
	}

	// Find the player's room
	r := h.rm.FindRoomByPlayerID(playerID)
	if r == nil {
		client.SendMessage(ws.NewErrorMessage("not in a room"))
		return
	}

	// Only allow movement during playing state
	if r.State != game.StatePlaying {
		client.SendMessage(ws.NewErrorMessage("game is not in progress"))
		return
	}

	player := r.Players[playerID]
	if player == nil {
		return
	}

	// Speed validation: check distance against MoveSpeed * elapsed time
	now := time.Now()
	elapsed := now.Sub(player.LastMoveTime).Seconds()
	if player.LastMoveTime.IsZero() {
		elapsed = float64(game.TickInterval) / float64(time.Second)
	}

	dist := game.Distance(player.X, player.Y, req.X, req.Y)
	maxDist := game.MoveSpeed * elapsed * 1.5 // 50% tolerance for network jitter
	if dist > maxDist {
		slog.Warn("speed violation", "player", playerID, "dist", dist, "maxDist", maxDist)
		client.SendMessage(ws.NewErrorMessage("movement too fast"))
		return
	}

	player.SetPosition(req.X, req.Y)
	player.LastMoveTime = now

	// Broadcast movement to other players in the room
	moveMsg, _ := ws.NewMessage(ws.TypePlayerMove, playerMoveResponse{
		PlayerID: playerID,
		X:        req.X,
		Y:        req.Y,
	})
	r.BroadcastMessage(moveMsg)

	slog.Debug("player moved", "player", playerID, "x", req.X, "y", req.Y)
}
