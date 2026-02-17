package room

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/game"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

// mockClient creates a ws.Client with a buffered Send channel for testing.
func mockClient(id string) *ws.Client {
	return &ws.Client{
		ID:   id,
		Send: make(chan []byte, 256),
	}
}

// drainMessages reads all pending messages from a client's send channel.
func drainMessages(client *ws.Client) []ws.Message {
	var msgs []ws.Message
	for {
		select {
		case data := <-client.Send:
			var msg ws.Message
			if err := json.Unmarshal(data, &msg); err == nil {
				msgs = append(msgs, msg)
			}
		default:
			return msgs
		}
	}
}

// findMessageByType finds the first message of a given type.
func findMessageByType(msgs []ws.Message, msgType string) *ws.Message {
	for _, m := range msgs {
		if m.Type == msgType {
			return &m
		}
	}
	return nil
}

func setupTestRoom() (*Room, []*ws.Client) {
	r := NewRoom("TEST")
	c1 := mockClient("client1")
	c2 := mockClient("client2")

	p1 := &game.Player{ID: "p1", Nickname: "Police1", Role: game.RolePolice, Ready: true}
	p2 := &game.Player{ID: "p2", Nickname: "Thief1", Role: game.RoleThief, Ready: true}

	r.AddPlayer(p1, c1)
	r.AddPlayer(p2, c2)

	return r, []*ws.Client{c1, c2}
}

func TestStartGame_SetsState(t *testing.T) {
	r, _ := setupTestRoom()
	r.PrepareGame()
	r.StartGameLoop()
	defer r.StopGame(game.WinNone)

	assert.Equal(t, game.StatePlaying, r.State)
}

func TestStartGame_AssignsSpawnPositions(t *testing.T) {
	r, _ := setupTestRoom()
	r.PrepareGame()
	r.StartGameLoop()
	defer r.StopGame(game.WinNone)

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.Players {
		// All players should have non-zero positions after spawn
		assert.True(t, p.X > 0 || p.Y > 0, "player %s should have a spawn position", p.ID)
	}
}

func TestStartGame_SetsRemainingTime(t *testing.T) {
	r, _ := setupTestRoom()
	r.PrepareGame()
	r.StartGameLoop()
	defer r.StopGame(game.WinNone)

	remaining := r.RemainingTime()
	assert.True(t, remaining > 0, "remaining time should be positive")
	assert.True(t, remaining <= game.GameDuration, "remaining time should not exceed game duration")
}

func TestStopGame_TransitionsToEnded(t *testing.T) {
	r, _ := setupTestRoom()
	r.PrepareGame()
	r.StartGameLoop()

	// Let one tick happen
	time.Sleep(game.TickInterval + 10*time.Millisecond)

	r.StopGame(game.WinPolice)

	assert.Equal(t, game.StateEnded, r.State)
}

func TestStopGame_BroadcastsGameOver(t *testing.T) {
	r, clients := setupTestRoom()
	r.PrepareGame()
	r.StartGameLoop()

	// Let a tick happen
	time.Sleep(game.TickInterval + 10*time.Millisecond)

	// Drain any game_state messages
	for _, c := range clients {
		drainMessages(c)
	}

	r.StopGame(game.WinPolice)

	// Check that game_over was broadcast
	time.Sleep(10 * time.Millisecond)
	for _, c := range clients {
		msgs := drainMessages(c)
		overMsg := findMessageByType(msgs, ws.TypeGameOver)
		require.NotNil(t, overMsg, "should receive game_over message")
	}
}

func TestGameLoop_BroadcastsGameState(t *testing.T) {
	r, clients := setupTestRoom()
	r.PrepareGame()
	r.StartGameLoop()
	defer r.StopGame(game.WinNone)

	// Wait for at least one tick
	time.Sleep(game.TickInterval + 20*time.Millisecond)

	for _, c := range clients {
		msgs := drainMessages(c)
		stateMsg := findMessageByType(msgs, ws.TypeGameState)
		require.NotNil(t, stateMsg, "should receive game_state message")
	}
}

func TestGameLoop_TimerExpiry(t *testing.T) {
	r, clients := setupTestRoom()

	r.mu.Lock()
	r.State = game.StatePlaying
	r.remainingTime = 100 * time.Millisecond // Very short timer
	r.stopCh = make(chan struct{})

	players := make([]*game.Player, 0, len(r.Players))
	for _, p := range r.Players {
		players = append(players, p)
	}
	positions := game.GenerateSpawnPositions(players)
	for id, pos := range positions {
		r.Players[id].SetPosition(pos.X, pos.Y)
	}
	r.mu.Unlock()

	go r.gameLoop()

	// Wait for timer to expire
	time.Sleep(300 * time.Millisecond)

	assert.Equal(t, game.StateEnded, r.State)

	// Check that game_over was sent
	for _, c := range clients {
		msgs := drainMessages(c)
		overMsg := findMessageByType(msgs, ws.TypeGameOver)
		require.NotNil(t, overMsg, "should receive game_over on timer expiry")

		var data gameOverMessage
		json.Unmarshal(overMsg.Data, &data)
		assert.Equal(t, "thief", data.Winner)
	}
}

func TestStopGame_DoubleStopSafe(t *testing.T) {
	r, _ := setupTestRoom()
	r.PrepareGame()
	r.StartGameLoop()

	time.Sleep(game.TickInterval + 10*time.Millisecond)

	// Should not panic on double stop
	r.StopGame(game.WinPolice)
	r.StopGame(game.WinPolice)

	assert.Equal(t, game.StateEnded, r.State)
}
