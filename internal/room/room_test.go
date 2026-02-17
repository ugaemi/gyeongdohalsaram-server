package room

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/game"
)

func TestAllReady_RequiresMinPlayers(t *testing.T) {
	r := NewRoom("TEST")
	c1 := mockClient("client1")
	p1 := &game.Player{ID: "p1", Nickname: "Solo", Role: game.RolePolice, Ready: true}
	r.AddPlayer(p1, c1)

	assert.False(t, r.AllReady(), "should not be ready with less than MinPlayers")
}

func TestAllReady_RequiresBothTeams(t *testing.T) {
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

			p1 := &game.Player{ID: "p1", Nickname: "P1", Role: tt.role1, Ready: true}
			p2 := &game.Player{ID: "p2", Nickname: "P2", Role: tt.role2, Ready: true}
			r.AddPlayer(p1, c1)
			r.AddPlayer(p2, c2)

			assert.Equal(t, tt.want, r.AllReady())
		})
	}
}

func TestAllReady_RequiresAllReady(t *testing.T) {
	r := NewRoom("TEST")
	c1 := mockClient("client1")
	c2 := mockClient("client2")

	p1 := &game.Player{ID: "p1", Nickname: "P1", Role: game.RolePolice, Ready: true}
	p2 := &game.Player{ID: "p2", Nickname: "P2", Role: game.RoleThief, Ready: false}
	r.AddPlayer(p1, c1)
	r.AddPlayer(p2, c2)

	assert.False(t, r.AllReady(), "should not be ready when a player is not ready")
}

func TestAllReady_RequiresRoleSelected(t *testing.T) {
	r := NewRoom("TEST")
	c1 := mockClient("client1")
	c2 := mockClient("client2")

	p1 := &game.Player{ID: "p1", Nickname: "P1", Role: game.RolePolice, Ready: true}
	p2 := &game.Player{ID: "p2", Nickname: "P2", Role: game.RoleNone, Ready: true}
	r.AddPlayer(p1, c1)
	r.AddPlayer(p2, c2)

	assert.False(t, r.AllReady(), "should not be ready when a player has no role")
}
