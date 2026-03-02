package game

import (
	"fmt"
	"math/rand"
	"time"
)

// StumbleStone represents a speed-reducing obstacle on the map.
type StumbleStone struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

// StumbleStoneManager tracks active stumble stones and respawn timers.
type StumbleStoneManager struct {
	Active       []*StumbleStone
	RespawnQueue []time.Duration // countdown timers for pending respawns
	placed       []placedObj     // existing map objects to avoid overlap
	nextID       int
}

// NewStumbleStoneManager creates a manager and spawns initial stumble stones.
func NewStumbleStoneManager(mapObjects []MapObject) *StumbleStoneManager {
	placed := make([]placedObj, 0, len(mapObjects))
	for _, obj := range mapObjects {
		placed = append(placed, placedObj{x: obj.X, y: obj.Y})
	}

	sm := &StumbleStoneManager{
		placed: placed,
	}

	for i := 0; i < MaxStumbleStones; i++ {
		sm.spawnStone()
	}

	return sm
}

func (sm *StumbleStoneManager) spawnStone() {
	sm.nextID++
	id := fmt.Sprintf("stone_%d", sm.nextID)

	// Collect all occupied positions (map objects + active stones)
	occupied := make([]placedObj, len(sm.placed))
	copy(occupied, sm.placed)
	for _, s := range sm.Active {
		occupied = append(occupied, placedObj{x: s.X, y: s.Y})
	}

	const margin = 40.0 // stone size / 2
	centerX := float64(MapWidth) / 2
	centerY := float64(MapHeight) / 2

	var x, y float64
	for attempts := 0; attempts < 100; attempts++ {
		x = margin + rand.Float64()*(float64(MapWidth)-2*margin)
		y = margin + rand.Float64()*(float64(MapHeight)-2*margin)

		if Distance(x, y, centerX, centerY) < ObjectSpawnRadius {
			continue
		}

		tooClose := false
		for _, p := range occupied {
			if Distance(x, y, p.x, p.y) < ObjectMinDistance/2 {
				tooClose = true
				break
			}
		}
		if !tooClose {
			break
		}
	}

	sm.Active = append(sm.Active, &StumbleStone{ID: id, X: x, Y: y})
}

// Update processes stumble stone respawn timers. Call once per tick.
func (sm *StumbleStoneManager) Update(dt time.Duration) {
	remaining := sm.RespawnQueue[:0]
	for _, t := range sm.RespawnQueue {
		t -= dt
		if t <= 0 {
			sm.spawnStone()
		} else {
			remaining = append(remaining, t)
		}
	}
	sm.RespawnQueue = remaining
}

// CheckPickup tests if any player steps on a stumble stone. Returns picked up stone IDs.
func (sm *StumbleStoneManager) CheckPickup(players []*Player) []string {
	var picked []string
	var remaining []*StumbleStone

	for _, s := range sm.Active {
		collected := false
		for _, p := range players {
			if p.IsArrested() {
				continue
			}
			if Distance(p.X, p.Y, s.X, s.Y) <= StumbleStonePickupRange {
				p.Slowed = true
				p.SlowTimer = StumbleSlowDuration
				collected = true
				picked = append(picked, s.ID)
				break
			}
		}
		if !collected {
			remaining = append(remaining, s)
		}
	}

	sm.Active = remaining

	// Queue respawns for collected stones
	for range picked {
		sm.RespawnQueue = append(sm.RespawnQueue, StumbleRespawnTime)
	}

	return picked
}

// UpdatePlayerSlows decrements slow timers for all players. Call once per tick.
func UpdatePlayerSlows(players []*Player, dt time.Duration) {
	for _, p := range players {
		if p.Slowed {
			p.SlowTimer -= dt
			if p.SlowTimer <= 0 {
				p.Slowed = false
				p.SlowTimer = 0
			}
		}
	}
}
