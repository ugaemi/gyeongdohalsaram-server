package game

import "math/rand"

// Jail position constants (center-bottom area of the map).
const (
	JailX = float64(MapWidth) / 2   // 1620
	JailY = float64(MapHeight) * 0.8 // 4608
)

// GenerateSpawnPositions assigns spawn positions for all players.
// Police spawn in the upper half (y: 0~2880), thieves in the lower half (y: 2880~5760).
// Maintains MinSpawnDistance between all players.
func GenerateSpawnPositions(players []*Player) map[string]Position {
	positions := make(map[string]Position, len(players))
	placed := make([]Position, 0, len(players))

	for _, p := range players {
		var minY, maxY float64
		if p.Role == RolePolice {
			minY = 0
			maxY = float64(MapHeight) / 2
		} else {
			minY = float64(MapHeight) / 2
			maxY = float64(MapHeight)
		}

		pos := generatePosition(float64(0), float64(MapWidth), minY, maxY, placed)
		positions[p.ID] = pos
		placed = append(placed, pos)
	}

	return positions
}

// Position represents a 2D coordinate.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// generatePosition finds a random position within bounds that respects MinSpawnDistance
// from all existing positions. Falls back to a random position after maxAttempts.
func generatePosition(minX, maxX, minY, maxY float64, existing []Position) Position {
	const maxAttempts = 100
	// Add margin so players don't spawn at exact edges
	const margin = MinSpawnDistance

	adjMinX := minX + margin
	adjMaxX := maxX - margin
	adjMinY := minY + margin
	adjMaxY := maxY - margin

	for i := 0; i < maxAttempts; i++ {
		x := adjMinX + rand.Float64()*(adjMaxX-adjMinX)
		y := adjMinY + rand.Float64()*(adjMaxY-adjMinY)

		if isFarEnough(x, y, existing) {
			return Position{X: x, Y: y}
		}
	}

	// Fallback: return a random position even if distance is not guaranteed
	x := adjMinX + rand.Float64()*(adjMaxX-adjMinX)
	y := adjMinY + rand.Float64()*(adjMaxY-adjMinY)
	return Position{X: x, Y: y}
}

// isFarEnough checks if (x, y) is at least MinSpawnDistance from all existing positions.
func isFarEnough(x, y float64, existing []Position) bool {
	for _, p := range existing {
		if Distance(x, y, p.X, p.Y) < MinSpawnDistance {
			return false
		}
	}
	return true
}
