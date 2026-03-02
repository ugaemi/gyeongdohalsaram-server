package game

import (
	"fmt"
	"math/rand"
	"time"
)

// Booster represents a speed boost item on the map.
type Booster struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

// BoosterManager tracks active boosters and respawn timers.
type BoosterManager struct {
	Active       []*Booster
	RespawnQueue []time.Duration // countdown timers for pending respawns
	placed       []placedObj     // existing map objects to avoid overlap
	nextID       int
}

// NewBoosterManager creates a manager and spawns initial boosters.
func NewBoosterManager(mapObjects []MapObject) *BoosterManager {
	placed := make([]placedObj, 0, len(mapObjects))
	for _, obj := range mapObjects {
		placed = append(placed, placedObj{x: obj.X, y: obj.Y})
	}

	bm := &BoosterManager{
		placed: placed,
	}

	for i := 0; i < MaxBoosters; i++ {
		bm.spawnBooster()
	}

	return bm
}

func (bm *BoosterManager) spawnBooster() {
	bm.nextID++
	id := fmt.Sprintf("boost_%d", bm.nextID)

	// Collect all occupied positions (map objects + active boosters)
	occupied := make([]placedObj, len(bm.placed))
	copy(occupied, bm.placed)
	for _, b := range bm.Active {
		occupied = append(occupied, placedObj{x: b.X, y: b.Y})
	}

	const margin = 40.0 // booster size / 2
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

	bm.Active = append(bm.Active, &Booster{ID: id, X: x, Y: y})
}

// Update processes booster timers and respawns. Call once per tick.
func (bm *BoosterManager) Update(dt time.Duration) {
	// Process respawn timers
	remaining := bm.RespawnQueue[:0]
	for _, t := range bm.RespawnQueue {
		t -= dt
		if t <= 0 {
			bm.spawnBooster()
		} else {
			remaining = append(remaining, t)
		}
	}
	bm.RespawnQueue = remaining
}

// CheckPickup tests if any player picks up a booster. Returns picked up booster IDs.
func (bm *BoosterManager) CheckPickup(players []*Player) []string {
	var picked []string
	var remaining []*Booster

	for _, b := range bm.Active {
		collected := false
		for _, p := range players {
			if p.IsArrested() {
				continue
			}
			if Distance(p.X, p.Y, b.X, b.Y) <= BoosterPickupRange {
				p.Boosted = true
				p.BoostTimer = BoosterDuration
				collected = true
				picked = append(picked, b.ID)
				break
			}
		}
		if !collected {
			remaining = append(remaining, b)
		}
	}

	bm.Active = remaining

	// Queue respawns for collected boosters
	for range picked {
		bm.RespawnQueue = append(bm.RespawnQueue, BoosterRespawnTime)
	}

	return picked
}

// UpdatePlayerBoosts decrements boost timers for all players. Call once per tick.
func UpdatePlayerBoosts(players []*Player, dt time.Duration) {
	for _, p := range players {
		if p.Boosted {
			p.BoostTimer -= dt
			if p.BoostTimer <= 0 {
				p.Boosted = false
				p.BoostTimer = 0
			}
		}
	}
}
