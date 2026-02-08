package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistance(t *testing.T) {
	tests := []struct {
		name     string
		x1, y1   float64
		x2, y2   float64
		expected float64
	}{
		{"same point", 0, 0, 0, 0, 0},
		{"horizontal", 0, 0, 3, 0, 3},
		{"vertical", 0, 0, 0, 4, 4},
		{"diagonal 3-4-5", 0, 0, 3, 4, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Distance(tt.x1, tt.y1, tt.x2, tt.y2)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestInArrestRange(t *testing.T) {
	police := &Player{X: 100, Y: 100, Role: RolePolice}

	tests := []struct {
		name     string
		thief    *Player
		expected bool
	}{
		{"within range", &Player{X: 150, Y: 100, Role: RoleThief}, true},
		{"at boundary", &Player{X: 200, Y: 100, Role: RoleThief}, true},
		{"out of range", &Player{X: 300, Y: 100, Role: RoleThief}, false},
		{"same position", &Player{X: 100, Y: 100, Role: RoleThief}, true},
		{"just outside", &Player{X: 100 + ArrestRange + 1, Y: 100, Role: RoleThief}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InArrestRange(police, tt.thief)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInJailRange(t *testing.T) {
	jailX, jailY := JailX, JailY

	tests := []struct {
		name     string
		thief    *Player
		expected bool
	}{
		{"at jail", &Player{X: jailX, Y: jailY, Role: RoleThief}, true},
		{"within range", &Player{X: jailX + 50, Y: jailY, Role: RoleThief}, true},
		{"at boundary", &Player{X: jailX + JailRange, Y: jailY, Role: RoleThief}, true},
		{"out of range", &Player{X: jailX + JailRange + 50, Y: jailY, Role: RoleThief}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InJailRange(tt.thief, jailX, jailY)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindArrestPairs(t *testing.T) {
	t.Run("police near free thief", func(t *testing.T) {
		players := []*Player{
			{ID: "p1", X: 100, Y: 100, Role: RolePolice},
			{ID: "t1", X: 150, Y: 100, Role: RoleThief, State: StateFree},
		}
		pairs := FindArrestPairs(players)
		assert.Len(t, pairs, 1)
		assert.Equal(t, "p1", pairs[0][0].ID)
		assert.Equal(t, "t1", pairs[0][1].ID)
	})

	t.Run("police far from thief", func(t *testing.T) {
		players := []*Player{
			{ID: "p1", X: 100, Y: 100, Role: RolePolice},
			{ID: "t1", X: 500, Y: 500, Role: RoleThief, State: StateFree},
		}
		pairs := FindArrestPairs(players)
		assert.Len(t, pairs, 0)
	})

	t.Run("ignores arrested thief", func(t *testing.T) {
		players := []*Player{
			{ID: "p1", X: 100, Y: 100, Role: RolePolice},
			{ID: "t1", X: 110, Y: 100, Role: RoleThief, State: StateArrested},
		}
		pairs := FindArrestPairs(players)
		assert.Len(t, pairs, 0)
	})

	t.Run("ignores invincible thief", func(t *testing.T) {
		players := []*Player{
			{ID: "p1", X: 100, Y: 100, Role: RolePolice},
			{ID: "t1", X: 110, Y: 100, Role: RoleThief, State: StateInvincible},
		}
		pairs := FindArrestPairs(players)
		assert.Len(t, pairs, 0)
	})

	t.Run("multiple pairs", func(t *testing.T) {
		players := []*Player{
			{ID: "p1", X: 100, Y: 100, Role: RolePolice},
			{ID: "p2", X: 500, Y: 500, Role: RolePolice},
			{ID: "t1", X: 110, Y: 100, Role: RoleThief, State: StateFree},
			{ID: "t2", X: 510, Y: 500, Role: RoleThief, State: StateFree},
			{ID: "t3", X: 1000, Y: 1000, Role: RoleThief, State: StateFree},
		}
		pairs := FindArrestPairs(players)
		assert.Len(t, pairs, 2)
	})

	t.Run("empty players", func(t *testing.T) {
		pairs := FindArrestPairs(nil)
		assert.Len(t, pairs, 0)
	})
}

func TestFindJailRescueCandidates(t *testing.T) {
	jailX, jailY := JailX, JailY

	t.Run("free thief near jail", func(t *testing.T) {
		players := []*Player{
			{ID: "t1", X: jailX + 50, Y: jailY, Role: RoleThief, State: StateFree},
		}
		candidates := FindJailRescueCandidates(players, jailX, jailY)
		assert.Len(t, candidates, 1)
		assert.Equal(t, "t1", candidates[0].ID)
	})

	t.Run("arrested thief near jail excluded", func(t *testing.T) {
		players := []*Player{
			{ID: "t1", X: jailX, Y: jailY, Role: RoleThief, State: StateArrested},
		}
		candidates := FindJailRescueCandidates(players, jailX, jailY)
		assert.Len(t, candidates, 0)
	})

	t.Run("free thief far from jail", func(t *testing.T) {
		players := []*Player{
			{ID: "t1", X: 0, Y: 0, Role: RoleThief, State: StateFree},
		}
		candidates := FindJailRescueCandidates(players, jailX, jailY)
		assert.Len(t, candidates, 0)
	})

	t.Run("police near jail excluded", func(t *testing.T) {
		players := []*Player{
			{ID: "p1", X: jailX, Y: jailY, Role: RolePolice, State: StateFree},
		}
		candidates := FindJailRescueCandidates(players, jailX, jailY)
		assert.Len(t, candidates, 0)
	})

	t.Run("multiple candidates", func(t *testing.T) {
		players := []*Player{
			{ID: "t1", X: jailX + 10, Y: jailY, Role: RoleThief, State: StateFree},
			{ID: "t2", X: jailX, Y: jailY + 10, Role: RoleThief, State: StateFree},
			{ID: "t3", X: 0, Y: 0, Role: RoleThief, State: StateFree},           // too far
			{ID: "t4", X: jailX, Y: jailY, Role: RoleThief, State: StateArrested}, // arrested
		}
		candidates := FindJailRescueCandidates(players, jailX, jailY)
		assert.Len(t, candidates, 2)
	})

	t.Run("empty players", func(t *testing.T) {
		candidates := FindJailRescueCandidates(nil, jailX, jailY)
		assert.Len(t, candidates, 0)
	})
}
