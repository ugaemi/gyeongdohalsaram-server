package game

// CheckPoliceWin returns true if all thieves are arrested.
func CheckPoliceWin(players []*Player) bool {
	thiefCount := 0
	arrestedCount := 0
	for _, p := range players {
		if p.Role == RoleThief {
			thiefCount++
			if p.IsArrested() {
				arrestedCount++
			}
		}
	}
	return thiefCount > 0 && thiefCount == arrestedCount
}

// CheckThiefWin returns true if the timer expired and at least one thief is free.
func CheckThiefWin(players []*Player, timerExpired bool) bool {
	if !timerExpired {
		return false
	}
	for _, p := range players {
		if p.Role == RoleThief && !p.IsArrested() {
			return true
		}
	}
	return false
}
