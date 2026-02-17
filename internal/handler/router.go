package handler

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/auth"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/room"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/store"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

// Router dispatches incoming messages to the appropriate handler.
type Router struct {
	authH    *AuthHandler
	lobby    *LobbyHandler
	gameplay *GameplayHandler

	// playerMap tracks client ID -> player ID mapping, shared across handlers.
	playerMap map[string]string
	mu        sync.RWMutex
}

// NewRouter creates a new message router.
func NewRouter(rm *room.Manager, verifier *auth.GameCenterVerifier, accountStore store.AccountStore) *Router {
	r := &Router{
		playerMap: make(map[string]string),
	}
	r.authH = NewAuthHandler(verifier, accountStore)
	r.lobby = NewLobbyHandler(rm, r)
	r.gameplay = NewGameplayHandler(rm, r)
	return r
}

// RegisterPlayer maps a client ID to a player ID.
func (r *Router) RegisterPlayer(clientID, playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.playerMap[clientID] = playerID
}

// UnregisterPlayer removes a client's player mapping.
func (r *Router) UnregisterPlayer(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.playerMap, clientID)
}

// GetPlayerID returns the player ID for a client, or empty string if not found.
func (r *Router) GetPlayerID(clientID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.playerMap[clientID]
}

// HandleMessage parses and routes an incoming client message.
func (r *Router) HandleMessage(cm *ws.ClientMessage) {
	var msg ws.Message
	if err := json.Unmarshal(cm.Data, &msg); err != nil {
		slog.Warn("invalid message format", "client", cm.Client.ID, "error", err)
		cm.Client.SendMessage(ws.NewErrorMessage("invalid message format"))
		return
	}

	// Auth messages are always allowed
	if msg.Type == ws.TypeAuthenticate {
		r.authH.HandleAuthenticate(cm.Client, msg)
		return
	}

	// Auth guard: block unauthenticated clients
	if !cm.Client.Authenticated {
		cm.Client.SendMessage(ws.NewErrorMessage("authentication required"))
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
	case ws.TypeReturnToLobby:
		r.lobby.HandleReturnToLobby(cm.Client, msg)

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

// StartAuthTimeout starts the authentication timeout for a new client.
func (r *Router) StartAuthTimeout(client *ws.Client) {
	r.authH.StartAuthTimeout(client)
}
