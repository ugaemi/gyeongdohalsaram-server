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
	MoveSpeed = 400.0 // pixels per second
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
