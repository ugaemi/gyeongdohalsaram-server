package handler

import (
	"encoding/json"
	"log/slog"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/game"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/room"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

// LobbyHandler handles lobby-related messages.
type LobbyHandler struct {
	rm     *room.Manager
	router *Router
}

// NewLobbyHandler creates a new lobby handler.
func NewLobbyHandler(rm *room.Manager, router *Router) *LobbyHandler {
	return &LobbyHandler{
		rm:     rm,
		router: router,
	}
}

type createRoomRequest struct {
	Nickname string `json:"nickname"`
}

type createRoomResponse struct {
	Code     string `json:"code"`
	PlayerID string `json:"player_id"`
}

// HandleCreateRoom handles room creation.
func (h *LobbyHandler) HandleCreateRoom(client *ws.Client, msg ws.Message) {
	var req createRoomRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil || req.Nickname == "" {
		client.SendMessage(ws.NewErrorMessage("nickname is required"))
		return
	}

	r := h.rm.CreateRoom()
	player := game.NewPlayer(req.Nickname)
	r.AddPlayer(player, client)
	h.router.RegisterPlayer(client.ID, player.ID)

	resp, _ := ws.NewMessage(ws.TypeCreateRoom, createRoomResponse{
		Code:     r.Code,
		PlayerID: player.ID,
	})
	client.SendMessage(resp)

	slog.Info("player created room", "player", player.Nickname, "room", r.Code)
}

type joinRoomRequest struct {
	Code     string `json:"code"`
	Nickname string `json:"nickname"`
}

// HandleJoinRoom handles joining an existing room.
func (h *LobbyHandler) HandleJoinRoom(client *ws.Client, msg ws.Message) {
	var req joinRoomRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil || req.Code == "" || req.Nickname == "" {
		client.SendMessage(ws.NewErrorMessage("code and nickname are required"))
		return
	}

	r := h.rm.GetRoom(req.Code)
	if r == nil {
		client.SendMessage(ws.NewErrorMessage("room not found"))
		return
	}

	player := game.NewPlayer(req.Nickname)
	if !r.AddPlayer(player, client) {
		client.SendMessage(ws.NewErrorMessage("room is full"))
		return
	}
	h.router.RegisterPlayer(client.ID, player.ID)

	resp, _ := ws.NewMessage(ws.TypeJoinRoom, createRoomResponse{
		Code:     r.Code,
		PlayerID: player.ID,
	})
	client.SendMessage(resp)

	h.broadcastRoomInfo(r)

	slog.Info("player joined room", "player", player.Nickname, "room", r.Code)
}

type selectTeamRequest struct {
	Role string `json:"role"` // "police" or "thief"
}

// HandleSelectTeam handles team selection.
func (h *LobbyHandler) HandleSelectTeam(client *ws.Client, msg ws.Message) {
	var req selectTeamRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		client.SendMessage(ws.NewErrorMessage("invalid team selection"))
		return
	}

	playerID := h.router.GetPlayerID(client.ID)
	r := h.rm.FindRoomByPlayerID(playerID)
	if r == nil {
		client.SendMessage(ws.NewErrorMessage("not in a room"))
		return
	}

	var role game.Role
	switch req.Role {
	case "police":
		role = game.RolePolice
	case "thief":
		role = game.RoleThief
	default:
		client.SendMessage(ws.NewErrorMessage("invalid role"))
		return
	}

	if !r.CanSelectRole(role) {
		client.SendMessage(ws.NewErrorMessage("team is full"))
		return
	}

	r.Players[playerID].SetRole(role)
	h.broadcastRoomInfo(r)

	slog.Info("player selected team", "player", playerID, "role", role.String())
}

// HandlePlayerReady handles player ready status.
func (h *LobbyHandler) HandlePlayerReady(client *ws.Client, msg ws.Message) {
	playerID := h.router.GetPlayerID(client.ID)
	r := h.rm.FindRoomByPlayerID(playerID)
	if r == nil {
		client.SendMessage(ws.NewErrorMessage("not in a room"))
		return
	}

	allReady := r.SetPlayerReady(playerID, true)
	h.broadcastRoomInfo(r)

	slog.Info("player ready", "player", playerID, "room", r.Code)

	// Check if all players are ready to start
	if allReady {
		// Broadcast game_start before starting the loop
		startMsg, _ := ws.NewMessage(ws.TypeGameStart, gameStartResponse{
			Players: r.GetPlayerList(),
		})
		r.BroadcastMessage(startMsg)
		r.StartGame()
		slog.Info("all players ready, game starting", "room", r.Code)
	}
}

// HandleLeaveRoom handles a player leaving a room.
func (h *LobbyHandler) HandleLeaveRoom(client *ws.Client, _ ws.Message) {
	h.removePlayer(client)
}

// HandleDisconnect handles client disconnection.
func (h *LobbyHandler) HandleDisconnect(client *ws.Client) {
	h.removePlayer(client)
}

func (h *LobbyHandler) removePlayer(client *ws.Client) {
	playerID := h.router.GetPlayerID(client.ID)
	if playerID == "" {
		return
	}

	r := h.rm.FindRoomByPlayerID(playerID)
	if r != nil {
		r.RemovePlayer(playerID)
		if r.IsEmpty() {
			h.rm.RemoveRoom(r.Code)
		} else {
			h.broadcastRoomInfo(r)
		}
	}

	h.router.UnregisterPlayer(client.ID)
	slog.Info("player left", "player", playerID)
}

type gameStartResponse struct {
	Players []*game.Player `json:"players"`
}

type roomInfoResponse struct {
	Code    string         `json:"code"`
	State   string         `json:"state"`
	Players []*game.Player `json:"players"`
	HostID  string         `json:"host_id"`
}

func (h *LobbyHandler) broadcastRoomInfo(r *room.Room) {
	resp, _ := ws.NewMessage(ws.TypeRoomInfo, roomInfoResponse{
		Code:    r.Code,
		State:   r.State.String(),
		Players: r.GetPlayerList(),
		HostID:  r.HostID,
	})
	r.BroadcastMessage(resp)
}
