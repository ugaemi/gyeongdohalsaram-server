package handler

import (
	"encoding/json"
	"log/slog"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/room"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

// Router dispatches incoming messages to the appropriate handler.
type Router struct {
	lobby    *LobbyHandler
	gameplay *GameplayHandler
}

// NewRouter creates a new message router.
func NewRouter(rm *room.Manager) *Router {
	return &Router{
		lobby:    NewLobbyHandler(rm),
		gameplay: NewGameplayHandler(rm),
	}
}

// HandleMessage parses and routes an incoming client message.
func (r *Router) HandleMessage(cm *ws.ClientMessage) {
	var msg ws.Message
	if err := json.Unmarshal(cm.Data, &msg); err != nil {
		slog.Warn("invalid message format", "client", cm.Client.ID, "error", err)
		cm.Client.SendMessage(ws.NewErrorMessage("invalid message format"))
		return
	}

	switch msg.Type {
	// Lobby messages
	case ws.TypeCreateRoom:
		r.lobby.HandleCreateRoom(cm.Client, msg)
	case ws.TypeJoinRoom:
		r.lobby.HandleJoinRoom(cm.Client, msg)
	case ws.TypeLeaveRoom:
		r.lobby.HandleLeaveRoom(cm.Client, msg)
	case ws.TypeSelectTeam:
		r.lobby.HandleSelectTeam(cm.Client, msg)
	case ws.TypePlayerReady:
		r.lobby.HandlePlayerReady(cm.Client, msg)

	// Gameplay messages
	case ws.TypePlayerMove:
		r.gameplay.HandlePlayerMove(cm.Client, msg)

	default:
		slog.Warn("unknown message type", "type", msg.Type, "client", cm.Client.ID)
		cm.Client.SendMessage(ws.NewErrorMessage("unknown message type: " + msg.Type))
	}
}

// HandleDisconnect handles client disconnection.
func (r *Router) HandleDisconnect(client *ws.Client) {
	r.lobby.HandleDisconnect(client)
}
