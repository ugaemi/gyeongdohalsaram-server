package game

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Role int

const (
	RoleNone Role = iota
	RolePolice
	RoleThief
)

func (r Role) String() string {
	switch r {
	case RolePolice:
		return "police"
	case RoleThief:
		return "thief"
	default:
		return "none"
	}
}

// MarshalJSON serializes Role as a string.
func (r Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

// UnmarshalJSON deserializes Role from a string.
func (r *Role) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "police":
		*r = RolePolice
	case "thief":
		*r = RoleThief
	default:
		*r = RoleNone
	}
	return nil
}

type PlayerState int

const (
	StateFree PlayerState = iota
	StateArrested
	StateInvincible
)

func (s PlayerState) String() string {
	switch s {
	case StateFree:
		return "free"
	case StateArrested:
		return "arrested"
	case StateInvincible:
		return "invincible"
	default:
		return "unknown"
	}
}

// MarshalJSON serializes PlayerState as a string.
func (s PlayerState) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON deserializes PlayerState from a string.
func (s *PlayerState) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	switch str {
	case "arrested":
		*s = StateArrested
	case "invincible":
		*s = StateInvincible
	default:
		*s = StateFree
	}
	return nil
}

type Player struct {
	ID           string      `json:"id"`
	Nickname     string      `json:"nickname"`
	Role         Role        `json:"role"`
	State        PlayerState `json:"state"`
	X            float64     `json:"x"`
	Y            float64     `json:"y"`
	Ready        bool        `json:"ready"`
	LastMoveTime time.Time   `json:"-"`
}

func NewPlayer(nickname string) *Player {
	return &Player{
		ID:       uuid.New().String(),
		Nickname: nickname,
		Role:     RoleNone,
		State:    StateFree,
	}
}

func (p *Player) SetRole(role Role) {
	p.Role = role
}

func (p *Player) SetPosition(x, y float64) {
	p.X = x
	p.Y = y
}

func (p *Player) Arrest() {
	p.State = StateArrested
}

func (p *Player) Release() {
	p.State = StateFree
}

func (p *Player) SetInvincible() {
	p.State = StateInvincible
}

func (p *Player) IsArrested() bool {
	return p.State == StateArrested
}

func (p *Player) IsFree() bool {
	return p.State == StateFree
}

func (p *Player) IsInvincible() bool {
	return p.State == StateInvincible
}

func (p *Player) Reset() {
	p.State = StateFree
	p.Ready = false
	p.X = 0
	p.Y = 0
	p.LastMoveTime = time.Time{}
}
