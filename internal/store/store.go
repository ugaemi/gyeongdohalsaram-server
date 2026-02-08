package store

import (
	"context"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/account"
)

// AccountStore defines the interface for persistent account storage.
type AccountStore interface {
	// FindByGameCenterID looks up an account by Game Center player ID.
	FindByGameCenterID(ctx context.Context, gcID string) (*account.Account, error)
	// FindByID looks up an account by internal ID.
	FindByID(ctx context.Context, id string) (*account.Account, error)
	// Create inserts a new account.
	Create(ctx context.Context, acc *account.Account) error
	// UpdateLastLogin updates the last login timestamp.
	UpdateLastLogin(ctx context.Context, id string) error
	// UpdateNickname updates the account nickname.
	UpdateNickname(ctx context.Context, id string, nickname string) error
	// Close releases database resources.
	Close() error
}
