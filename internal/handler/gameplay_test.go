package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/auth"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/game"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/room"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

func setupGameplayTest() (*Router, *room.Room, *ws.Client, chan sentMessage) {
	store := newMockAccountStore()
	verifier := auth.NewGameCenterVerifier(nil, 0)
	rm := room.NewManager()
	router := NewRouter(rm, verifier, store)

	r := rm.CreateRoom()
	client := &ws.Client{
		ID:            "test-client",
		Send:          make(chan []byte, 256),
		Authenticated: true,
	}

	player := &game.Player{
		ID:       "player1",
		Nickname: "Test",
		Role:     game.RolePolice,
		Ready:    true,
		X:        500,
		Y:        500,
	}
	player.LastMoveTime = time.Now()

	r.AddPlayer(player, client)
	router.RegisterPlayer(client.ID, player.ID)

	// Add a second player so the room can enter playing state
	client2 := &ws.Client{
		ID:            "test-client-2",
		Send:          make(chan []byte, 256),
		Authenticated: true,
	}
	player2 := &game.Player{
		ID:       "player2",
		Nickname: "Test2",
		Role:     game.RoleThief,
		Ready:    true,
		X:        500,
		Y:        3000,
	}
	player2.LastMoveTime = time.Now()
	r.AddPlayer(player2, client2)
	router.RegisterPlayer(client2.ID, player2.ID)

	ch := make(chan sentMessage, 10)
	go func() {
		for data := range client.Send {
			var msg sentMessage
			json.Unmarshal(data, &msg)
			ch <- msg
		}
	}()

	return router, r, client, ch
}

func TestHandlePlayerMove_ValidMove(t *testing.T) {
	router, r, client, ch := setupGameplayTest()
	r.StartGame()
	defer r.StopGame(game.WinNone)

	// Wait a moment to allow some elapsed time for speed validation
	time.Sleep(100 * time.Millisecond)

	// Drain any game_state messages
	drainCh(ch)

	// Get current position after spawn and move slightly
	player := r.Players["player1"]
	newX := player.X + 5
	newY := player.Y + 5

	data, _ := json.Marshal(playerMoveRequest{X: newX, Y: newY})
	rawMsg, _ := json.Marshal(ws.Message{Type: ws.TypePlayerMove, Data: data})
	router.HandleMessage(&ws.ClientMessage{Client: client, Data: rawMsg})

	// Should receive player_move broadcast (may also get game_state)
	var found bool
	for i := 0; i < 10; i++ {
		resp := readResponseWithTimeout(t, ch, 500*time.Millisecond)
		if resp.Type == ws.TypePlayerMove {
			var moveResp playerMoveResponse
			require.NoError(t, json.Unmarshal(resp.Data, &moveResp))
			assert.Equal(t, "player1", moveResp.PlayerID)
			assert.Equal(t, newX, moveResp.X)
			assert.Equal(t, newY, moveResp.Y)
			found = true
			break
		}
	}
	assert.True(t, found, "should have received player_move message")
}

func TestHandlePlayerMove_OutOfBounds_Clamped(t *testing.T) {
	router, r, client, ch := setupGameplayTest()
	r.StartGame()
	defer r.StopGame(game.WinNone)

	// Set player position near edge for speed validation
	r.Players["player1"].X = game.PlayerRadius
	r.Players["player1"].Y = 500
	r.Players["player1"].LastMoveTime = time.Now().Add(-1 * time.Second)

	drainCh(ch)

	// Position out of bounds — should be clamped, not rejected
	data, _ := json.Marshal(playerMoveRequest{X: -100, Y: 500})
	rawMsg, _ := json.Marshal(ws.Message{Type: ws.TypePlayerMove, Data: data})
	router.HandleMessage(&ws.ClientMessage{Client: client, Data: rawMsg})

	var found bool
	for i := 0; i < 10; i++ {
		resp := readResponseWithTimeout(t, ch, 500*time.Millisecond)
		if resp.Type == ws.TypePlayerMove {
			var moveResp playerMoveResponse
			require.NoError(t, json.Unmarshal(resp.Data, &moveResp))
			assert.Equal(t, game.PlayerRadius, moveResp.X, "X should be clamped to PlayerRadius")
			found = true
			break
		}
	}
	assert.True(t, found, "should have received clamped player_move message")
}

func TestHandlePlayerMove_SpeedViolation(t *testing.T) {
	router, r, client, ch := setupGameplayTest()

	// Set up room in playing state manually to control LastMoveTime
	r.StartGame()
	defer r.StopGame(game.WinNone)

	// Set LastMoveTime to now so elapsed time is very small
	r.Players["player1"].LastMoveTime = time.Now()
	r.Players["player1"].X = 100
	r.Players["player1"].Y = 100

	drainCh(ch)

	// Try to move very far in a very short time
	data, _ := json.Marshal(playerMoveRequest{X: 3000, Y: 3000})
	rawMsg, _ := json.Marshal(ws.Message{Type: ws.TypePlayerMove, Data: data})
	router.HandleMessage(&ws.ClientMessage{Client: client, Data: rawMsg})

	resp := readResponseWithTimeout(t, ch, 500*time.Millisecond)
	for resp.Type == ws.TypeGameState {
		resp = readResponseWithTimeout(t, ch, 500*time.Millisecond)
	}
	assert.Equal(t, ws.TypeError, resp.Type)

	var errMsg ws.ErrorMessage
	json.Unmarshal(resp.Data, &errMsg)
	assert.Equal(t, "movement too fast", errMsg.Message)
}

func TestHandlePlayerMove_NotPlaying(t *testing.T) {
	router, _, client, ch := setupGameplayTest()

	drainCh(ch)

	// Room is in waiting state - should reject move
	data, _ := json.Marshal(playerMoveRequest{X: 510, Y: 510})
	rawMsg, _ := json.Marshal(ws.Message{Type: ws.TypePlayerMove, Data: data})
	router.HandleMessage(&ws.ClientMessage{Client: client, Data: rawMsg})

	resp := readResponseWithTimeout(t, ch, 500*time.Millisecond)
	assert.Equal(t, ws.TypeError, resp.Type)

	var errMsg ws.ErrorMessage
	json.Unmarshal(resp.Data, &errMsg)
	assert.Equal(t, "game is not in progress", errMsg.Message)
}

func drainCh(ch chan sentMessage) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func readResponseWithTimeout(t *testing.T, ch chan sentMessage, timeout time.Duration) sentMessage {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(timeout):
		t.Fatal("timeout waiting for response")
		return sentMessage{}
	}
}
