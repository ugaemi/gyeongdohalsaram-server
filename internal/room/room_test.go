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

func TestFindAvailableRoom_WithPreferredRole(t *testing.T) {
	m := NewManager()

	// Police creates a room
	r := m.CreateRoom()
	c1 := mockClient("police-client")
	p1 := &game.Player{ID: "p1", Nickname: "경찰", Role: game.RoleNone}
	r.AddPlayer(p1, c1)

	// Police selects role
	p1.Role = game.RolePolice

	tests := []struct {
		name          string
		preferredRole game.Role
	}{
		{"thief prefers thief", game.RoleThief},
		{"police prefers police", game.RolePolice},
		{"no preference", game.RoleNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := m.FindAvailableRoom(tt.preferredRole)
			assert.NotNil(t, found, "should find available room for role %v", tt.preferredRole)
			assert.Equal(t, r.Code, found.Code)
		})
	}
}

func TestFindAvailableRoom_NoRooms(t *testing.T) {
	m := NewManager()
	found := m.FindAvailableRoom(game.RoleThief)
	assert.Nil(t, found, "should return nil when no rooms exist")
}

func TestFindAvailableRoom_PrefersRoleAvailable(t *testing.T) {
	m := NewManager()

	// Room 1: 2 police (full police cap)
	r1 := m.CreateRoom()
	r1.AddPlayer(&game.Player{ID: "p1", Role: game.RolePolice}, mockClient("c1"))
	r1.AddPlayer(&game.Player{ID: "p2", Role: game.RolePolice}, mockClient("c2"))

	// Room 2: 1 thief (police available)
	r2 := m.CreateRoom()
	r2.AddPlayer(&game.Player{ID: "p3", Role: game.RoleThief}, mockClient("c3"))

	// Prefer police → should pick room 2 (police slot available)
	found := m.FindAvailableRoom(game.RolePolice)
	assert.NotNil(t, found)
	assert.Equal(t, r2.Code, found.Code, "should prefer room where police slot is available")

	// Prefer thief → both rooms work, but both should be found
	found = m.FindAvailableRoom(game.RoleThief)
	assert.NotNil(t, found, "should find room for thief")
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
