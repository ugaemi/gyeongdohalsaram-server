package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessArrests(t *testing.T) {
	const dt = 0.05 // 20 TPS

	tests := []struct {
		name           string
		players        []*Player
		dt             float64
		wantEvents     int
		checkAfter     func(t *testing.T, players []*Player)
	}{
		{
			name: "accumulates ArrestProgress on contact",
			players: []*Player{
				{ID: "p1", X: 100, Y: 100, Role: RolePolice},
				{ID: "t1", X: 150, Y: 100, Role: RoleThief, State: StateFree},
			},
			dt:         dt,
			wantEvents: 0,
			checkAfter: func(t *testing.T, players []*Player) {
				assert.InDelta(t, dt, players[1].ArrestProgress, 0.001)
			},
		},
		{
			name: "arrest confirmed at 1.5s cumulative",
			players: []*Player{
				{ID: "p1", X: 100, Y: 100, Role: RolePolice},
				{ID: "t1", X: 150, Y: 100, Role: RoleThief, State: StateFree, ArrestProgress: ArrestDuration - dt},
			},
			dt:         dt,
			wantEvents: 1,
			checkAfter: func(t *testing.T, players []*Player) {
				assert.Equal(t, StateArrested, players[1].State)
				assert.InDelta(t, JailX, players[1].X, 0.001)
				assert.InDelta(t, JailY, players[1].Y, 0.001)
			},
		},
		{
			name: "ArrestProgress retained when out of range",
			players: []*Player{
				{ID: "p1", X: 100, Y: 100, Role: RolePolice},
				{ID: "t1", X: 500, Y: 500, Role: RoleThief, State: StateFree, ArrestProgress: 0.5},
			},
			dt:         dt,
			wantEvents: 0,
			checkAfter: func(t *testing.T, players []*Player) {
				assert.InDelta(t, 0.5, players[1].ArrestProgress, 0.001)
			},
		},
		{
			name: "already arrested thief is ignored",
			players: []*Player{
				{ID: "p1", X: 100, Y: 100, Role: RolePolice},
				{ID: "t1", X: 110, Y: 100, Role: RoleThief, State: StateArrested, ArrestProgress: 0.0},
			},
			dt:         dt,
			wantEvents: 0,
			checkAfter: func(t *testing.T, players []*Player) {
				assert.InDelta(t, 0.0, players[1].ArrestProgress, 0.001)
			},
		},
		{
			name: "invincible thief is ignored",
			players: []*Player{
				{ID: "p1", X: 100, Y: 100, Role: RolePolice},
				{ID: "t1", X: 110, Y: 100, Role: RoleThief, State: StateInvincible, ArrestProgress: 0.0},
			},
			dt:         dt,
			wantEvents: 0,
			checkAfter: func(t *testing.T, players []*Player) {
				assert.InDelta(t, 0.0, players[1].ArrestProgress, 0.001)
			},
		},
		{
			name: "multiple police on same thief accumulates only one dt per tick",
			players: []*Player{
				{ID: "p1", X: 100, Y: 100, Role: RolePolice},
				{ID: "p2", X: 100, Y: 110, Role: RolePolice},
				{ID: "t1", X: 110, Y: 105, Role: RoleThief, State: StateFree},
			},
			dt:         dt,
			wantEvents: 0,
			checkAfter: func(t *testing.T, players []*Player) {
				assert.InDelta(t, dt, players[2].ArrestProgress, 0.001)
			},
		},
		{
			name: "no police no progress",
			players: []*Player{
				{ID: "t1", X: 100, Y: 100, Role: RoleThief, State: StateFree},
				{ID: "t2", X: 110, Y: 100, Role: RoleThief, State: StateFree},
			},
			dt:         dt,
			wantEvents: 0,
			checkAfter: func(t *testing.T, players []*Player) {
				assert.InDelta(t, 0.0, players[0].ArrestProgress, 0.001)
				assert.InDelta(t, 0.0, players[1].ArrestProgress, 0.001)
			},
		},
		{
			name: "arrest event contains correct police and thief IDs",
			players: []*Player{
				{ID: "cop1", X: 100, Y: 100, Role: RolePolice},
				{ID: "thief1", X: 150, Y: 100, Role: RoleThief, State: StateFree, ArrestProgress: ArrestDuration - dt},
			},
			dt:         dt,
			wantEvents: 1,
			checkAfter: func(t *testing.T, players []*Player) {
				// Verified via event check below
			},
		},
		{
			name: "empty player list",
			players: nil,
			dt:         dt,
			wantEvents: 0,
			checkAfter: func(t *testing.T, players []*Player) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := ProcessArrests(tt.players, tt.dt)
			assert.Len(t, events, tt.wantEvents)
			tt.checkAfter(t, tt.players)
		})
	}

	// Verify event content for the "arrest event contains correct IDs" case
	t.Run("arrest event IDs verified", func(t *testing.T) {
		players := []*Player{
			{ID: "cop1", X: 100, Y: 100, Role: RolePolice},
			{ID: "thief1", X: 150, Y: 100, Role: RoleThief, State: StateFree, ArrestProgress: ArrestDuration - dt},
		}
		events := ProcessArrests(players, dt)
		assert.Len(t, events, 1)
		assert.Equal(t, "cop1", events[0].PoliceID)
		assert.Equal(t, "thief1", events[0].ThiefID)
	})
}

func TestProcessArrests_CumulativeAcrossTicks(t *testing.T) {
	const dt = 0.05

	police := &Player{ID: "p1", X: 100, Y: 100, Role: RolePolice}
	thief := &Player{ID: "t1", X: 150, Y: 100, Role: RoleThief, State: StateFree}
	players := []*Player{police, thief}

	// Simulate 30 ticks (1.5s) of contact
	var allEvents []ArrestEvent
	for i := 0; i < 30; i++ {
		events := ProcessArrests(players, dt)
		allEvents = append(allEvents, events...)
	}

	assert.Len(t, allEvents, 1)
	assert.Equal(t, StateArrested, thief.State)
	assert.InDelta(t, JailX, thief.X, 0.001)
	assert.InDelta(t, JailY, thief.Y, 0.001)
}

func TestProcessArrests_IntermittentContact(t *testing.T) {
	const dt = 0.05

	police := &Player{ID: "p1", X: 100, Y: 100, Role: RolePolice}
	thief := &Player{ID: "t1", X: 150, Y: 100, Role: RoleThief, State: StateFree}
	players := []*Player{police, thief}

	// 10 ticks in range (0.5s accumulated)
	for i := 0; i < 10; i++ {
		ProcessArrests(players, dt)
	}
	assert.InDelta(t, 0.5, thief.ArrestProgress, 0.001)

	// Move thief out of range for 10 ticks — progress should not decrease
	thief.X = 500
	for i := 0; i < 10; i++ {
		ProcessArrests(players, dt)
	}
	assert.InDelta(t, 0.5, thief.ArrestProgress, 0.001)

	// Move thief back in range for 20 more ticks (1.0s more → total 1.5s)
	thief.X = 150
	var allEvents []ArrestEvent
	for i := 0; i < 20; i++ {
		events := ProcessArrests(players, dt)
		allEvents = append(allEvents, events...)
	}
	assert.Len(t, allEvents, 1)
	assert.Equal(t, StateArrested, thief.State)
}
