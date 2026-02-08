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
