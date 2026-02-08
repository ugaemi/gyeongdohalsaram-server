package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSpawnPositions_ZoneSeparation(t *testing.T) {
	players := []*Player{
		{ID: "p1", Role: RolePolice},
		{ID: "p2", Role: RolePolice},
		{ID: "t1", Role: RoleThief},
		{ID: "t2", Role: RoleThief},
		{ID: "t3", Role: RoleThief},
	}

	positions := GenerateSpawnPositions(players)
	require.Len(t, positions, 5)

	halfY := float64(MapHeight) / 2

	// Police should be in upper half
	for _, id := range []string{"p1", "p2"} {
		pos := positions[id]
		assert.LessOrEqual(t, pos.Y, halfY, "police %s should be in upper half", id)
	}

	// Thieves should be in lower half
	for _, id := range []string{"t1", "t2", "t3"} {
		pos := positions[id]
		assert.GreaterOrEqual(t, pos.Y, halfY, "thief %s should be in lower half", id)
	}
}

func TestGenerateSpawnPositions_MinDistance(t *testing.T) {
	players := []*Player{
		{ID: "p1", Role: RolePolice},
		{ID: "t1", Role: RoleThief},
		{ID: "t2", Role: RoleThief},
		{ID: "t3", Role: RoleThief},
		{ID: "t4", Role: RoleThief},
		{ID: "t5", Role: RoleThief},
	}

	// Run multiple times to catch randomness issues
	for i := 0; i < 20; i++ {
		positions := GenerateSpawnPositions(players)
		placed := make([]Position, 0, len(positions))
		for _, pos := range positions {
			for _, existing := range placed {
				dist := Distance(pos.X, pos.Y, existing.X, existing.Y)
				assert.GreaterOrEqual(t, dist, MinSpawnDistance-1.0,
					"players should maintain minimum spawn distance (got %.1f)", dist)
			}
			placed = append(placed, pos)
		}
	}
}

func TestGenerateSpawnPositions_WithinMapBounds(t *testing.T) {
	players := []*Player{
		{ID: "p1", Role: RolePolice},
		{ID: "t1", Role: RoleThief},
		{ID: "t2", Role: RoleThief},
	}

	for i := 0; i < 20; i++ {
		positions := GenerateSpawnPositions(players)
		for id, pos := range positions {
			assert.GreaterOrEqual(t, pos.X, 0.0, "player %s X should be >= 0", id)
			assert.LessOrEqual(t, pos.X, float64(MapWidth), "player %s X should be <= MapWidth", id)
			assert.GreaterOrEqual(t, pos.Y, 0.0, "player %s Y should be >= 0", id)
			assert.LessOrEqual(t, pos.Y, float64(MapHeight), "player %s Y should be <= MapHeight", id)
		}
	}
}

func TestJailPosition(t *testing.T) {
	assert.Equal(t, float64(MapWidth)/2, JailX)
	assert.Equal(t, float64(MapHeight)*0.8, JailY)
	assert.Greater(t, JailX, 0.0)
	assert.Greater(t, JailY, 0.0)
	assert.LessOrEqual(t, JailX, float64(MapWidth))
	assert.LessOrEqual(t, JailY, float64(MapHeight))
}

func TestIsFarEnough(t *testing.T) {
	existing := []Position{
		{X: 500, Y: 500},
	}

	// Too close
	assert.False(t, isFarEnough(550, 500, existing))

	// Far enough
	assert.True(t, isFarEnough(800, 800, existing))

	// Empty list - always far enough
	assert.True(t, isFarEnough(100, 100, nil))
}
