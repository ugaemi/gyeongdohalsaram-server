package game

// ArrestEvent represents a confirmed arrest of a thief by a police officer.
type ArrestEvent struct {
	PoliceID string
	ThiefID  string
}

// ProcessArrests checks arrest pairs, accumulates ArrestProgress on contacted thieves,
// and returns events for any arrests that were confirmed this tick.
// Each thief accumulates at most one dt per tick regardless of how many police are in range.
func ProcessArrests(players []*Player, dt float64) []ArrestEvent {
	pairs := FindArrestPairs(players)

	// Track which thieves are being contacted this tick (deduplicate by thief).
	// Store the first police ID that touches each thief for the event.
	contacted := make(map[string]string) // thief ID -> police ID
	for _, pair := range pairs {
		cop, thief := pair[0], pair[1]
		if _, ok := contacted[thief.ID]; !ok {
			contacted[thief.ID] = cop.ID
		}
	}

	// Accumulate progress and check for arrests.
	var events []ArrestEvent
	for _, p := range players {
		if p.Role != RoleThief {
			continue
		}
		if _, ok := contacted[p.ID]; !ok {
			continue
		}
		p.ArrestProgress += dt
		if p.ArrestProgress >= ArrestDuration {
			p.Arrest()
			events = append(events, ArrestEvent{
				PoliceID: contacted[p.ID],
				ThiefID:  p.ID,
			})
		}
	}

	return events
}
