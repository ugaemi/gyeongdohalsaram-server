package account

import (
	"time"

	"github.com/google/uuid"
)

// Account represents a persistent player account.
type Account struct {
	ID           string    `json:"id"`
	GameCenterID *string   `json:"game_center_id,omitempty"`
	Nickname     string    `json:"nickname"`
	IsGuest      bool      `json:"is_guest"`
	CreatedAt    time.Time `json:"created_at"`
	LastLoginAt  time.Time `json:"last_login_at"`
}

// NewGameCenterAccount creates a new account linked to a Game Center player ID.
func NewGameCenterAccount(gameCenterID, nickname string) *Account {
	now := time.Now()
	return &Account{
		ID:           uuid.New().String(),
		GameCenterID: &gameCenterID,
		Nickname:     nickname,
		IsGuest:      false,
		CreatedAt:    now,
		LastLoginAt:  now,
	}
}

// NewGuestAccount creates a new guest account with only a nickname.
func NewGuestAccount(nickname string) *Account {
	now := time.Now()
	return &Account{
		ID:          uuid.New().String(),
		Nickname:    nickname,
		IsGuest:     true,
		CreatedAt:   now,
		LastLoginAt: now,
	}
}
