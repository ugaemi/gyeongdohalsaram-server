package game

type RoomState int

const (
	StateWaiting RoomState = iota
	StatePlaying
	StateEnded
)

func (s RoomState) String() string {
	switch s {
	case StateWaiting:
		return "waiting"
	case StatePlaying:
		return "playing"
	case StateEnded:
		return "ended"
	default:
		return "unknown"
	}
}

type WinResult int

const (
	WinNone WinResult = iota
	WinPolice
	WinThief
)

func (w WinResult) String() string {
	switch w {
	case WinPolice:
		return "police"
	case WinThief:
		return "thief"
	default:
		return "none"
	}
}
