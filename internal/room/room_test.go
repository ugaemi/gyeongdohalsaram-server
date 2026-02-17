package room

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/game"
)

func TestSetPlayerReady_RequiresMinPlayers(t *testing.T) {
	r := NewRoom("TEST")
	c1 := mockClient("client1")
	p1 := &game.Player{ID: "p1", Nickname: "Solo", Role: game.RolePolice}
	r.AddPlayer(p1, c1)

	allReady := r.SetPlayerReady("p1", true)
	assert.False(t, allReady, "should not be ready with less than MinPlayers")
}

func TestSetPlayerReady_RequiresBothTeams(t *testing.T) {
	tests := []struct {
		name  string
		role1 game.Role
		role2 game.Role
		want  bool
	}{
		{"both thieves", game.RoleThief, game.RoleThief, false},
		{"both police", game.RolePolice, game.RolePolice, false},
		{"police and thief", game.RolePolice, game.RoleThief, true},
		{"thief and police", game.RoleThief, game.RolePolice, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRoom("TEST")
			c1 := mockClient("client1")
			c2 := mockClient("client2")

			p1 := &game.Player{ID: "p1", Nickname: "P1", Role: tt.role1}
			p2 := &game.Player{ID: "p2", Nickname: "P2", Role: tt.role2}
			r.AddPlayer(p1, c1)
			r.AddPlayer(p2, c2)

			r.SetPlayerReady("p1", true)
			allReady := r.SetPlayerReady("p2", true)
			assert.Equal(t, tt.want, allReady)
		})
	}
}

func TestSetPlayerReady_RequiresAllReady(t *testing.T) {
	r := NewRoom("TEST")
	c1 := mockClient("client1")
	c2 := mockClient("client2")

	p1 := &game.Player{ID: "p1", Nickname: "P1", Role: game.RolePolice}
	p2 := &game.Player{ID: "p2", Nickname: "P2", Role: game.RoleThief}
	r.AddPlayer(p1, c1)
	r.AddPlayer(p2, c2)

	// Only p1 ready
	allReady := r.SetPlayerReady("p1", true)
	assert.False(t, allReady, "should not be ready when only one player is ready")

	// Now p2 ready too
	allReady = r.SetPlayerReady("p2", true)
	assert.True(t, allReady, "should be ready when both players are ready")
}

func TestSetPlayerReady_RequiresRoleSelected(t *testing.T) {
	r := NewRoom("TEST")
	c1 := mockClient("client1")
	c2 := mockClient("client2")

	p1 := &game.Player{ID: "p1", Nickname: "P1", Role: game.RolePolice}
	p2 := &game.Player{ID: "p2", Nickname: "P2", Role: game.RoleNone}
	r.AddPlayer(p1, c1)
	r.AddPlayer(p2, c2)

	r.SetPlayerReady("p1", true)
	allReady := r.SetPlayerReady("p2", true)
	assert.False(t, allReady, "should not be ready when a player has no role")
}

func TestSetPlayerReady_AtomicReadyAndCheck(t *testing.T) {
	r := NewRoom("TEST")
	c1 := mockClient("client1")
	c2 := mockClient("client2")

	p1 := &game.Player{ID: "p1", Nickname: "P1", Role: game.RolePolice}
	p2 := &game.Player{ID: "p2", Nickname: "P2", Role: game.RoleThief}
	r.AddPlayer(p1, c1)
	r.AddPlayer(p2, c2)

	// Simulate: p1 already ready, p2 sets ready — should return true atomically
	p1.Ready = true
	allReady := r.SetPlayerReady("p2", true)
	assert.True(t, allReady, "SetPlayerReady should atomically set and check")
}
