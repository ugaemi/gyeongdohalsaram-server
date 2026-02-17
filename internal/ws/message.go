package ws

import "encoding/json"

// Message represents a WebSocket message with type-based routing.
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Message types - Lobby
const (
	TypeCreateRoom = "create_room"
	TypeJoinRoom   = "join_room"
	TypeLeaveRoom  = "leave_room"
	TypeSelectTeam = "select_team"
	TypePlayerReady   = "player_ready"
	TypeReturnToLobby = "return_to_lobby"
)

// Message types - Gameplay
const (
	TypePlayerMove = "player_move"
	TypeGameState  = "game_state"
	TypeGameOver   = "game_over"
	TypeGameStart  = "game_start"
)

// Message types - Auth
const (
	TypeAuthenticate = "authenticate"
	TypeAuthResult   = "auth_result"
)

// Message types - System
const (
	TypeError    = "error"
	TypeRoomInfo = "room_info"
)

// ErrorMessage is sent when an error occurs.
type ErrorMessage struct {
	Message string `json:"message"`
}

// NewErrorMessage creates a Message with an error payload.
func NewErrorMessage(msg string) Message {
	data, _ := json.Marshal(ErrorMessage{Message: msg})
	return Message{Type: TypeError, Data: data}
}

// NewMessage creates a Message with a typed payload.
func NewMessage(msgType string, payload any) (Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	return Message{Type: msgType, Data: data}, nil
}
