package game

import "time"

// Map dimensions (pixels)
const (
	MapWidth  = 3240
	MapHeight = 5760
)

// Player limits
const (
	MinPlayers   = 2
	MaxPlayers   = 8
	MaxPolice    = 2
)

// Movement
const (
	MoveSpeed    = 400.0 // pixels per second
	PlayerRadius = 50.0  // pixels, half of character sprite size
)

// Arrest mechanics
const (
	ArrestRange    = 100.0 // pixels
	ArrestDuration = 1.5   // seconds (cumulative)
)

// Rescue mechanics
const (
	JailRange       = 150.0        // pixels
	RescueDuration  = 2.0          // seconds (continuous)
	InvincibleTime  = 3 * time.Second
)

// Game timing
const (
	GameDuration = 180 * time.Second
	TickRate     = 20 // ticks per second
	TickInterval = time.Second / TickRate
	ResetDelay   = 5 * time.Second
)

// Spawn
const (
	MinSpawnDistance = 200.0 // minimum distance between spawned players
)

// Map objects (must match Godot client)
const (
	TreeCount          = 10
	LakeCount          = 2
	JailCount          = 1
	ObjectMinDistance   = 300.0 // minimum distance between objects
	ObjectSpawnRadius  = 300.0 // exclusion radius around map center
)
