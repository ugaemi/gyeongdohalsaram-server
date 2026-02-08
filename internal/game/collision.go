package game

import "math"

// Distance calculates the Euclidean distance between two points.
func Distance(x1, y1, x2, y2 float64) float64 {
	dx := x1 - x2
	dy := y1 - y2
	return math.Sqrt(dx*dx + dy*dy)
}

// InArrestRange checks if a police officer is within arrest range of a thief.
func InArrestRange(police, thief *Player) bool {
	return Distance(police.X, police.Y, thief.X, thief.Y) <= ArrestRange
}

// InJailRange checks if a free thief is within rescue range of the jail.
func InJailRange(thief *Player, jailX, jailY float64) bool {
	return Distance(thief.X, thief.Y, jailX, jailY) <= JailRange
}

// FindArrestPairs returns pairs of [police, thief] where the police is within
// arrest range of a free (non-arrested, non-invincible) thief.
func FindArrestPairs(players []*Player) [][2]*Player {
	var police []*Player
	var thieves []*Player

	for _, p := range players {
		switch p.Role {
		case RolePolice:
			police = append(police, p)
		case RoleThief:
			if p.IsFree() {
				thieves = append(thieves, p)
			}
		}
	}

	var pairs [][2]*Player
	for _, cop := range police {
		for _, thief := range thieves {
			if InArrestRange(cop, thief) {
				pairs = append(pairs, [2]*Player{cop, thief})
			}
		}
	}
	return pairs
}

// FindJailRescueCandidates returns free thieves that are within rescue range of the jail.
func FindJailRescueCandidates(players []*Player, jailX, jailY float64) []*Player {
	var candidates []*Player
	for _, p := range players {
		if p.Role == RoleThief && p.IsFree() && InJailRange(p, jailX, jailY) {
			candidates = append(candidates, p)
		}
	}
	return candidates
}
