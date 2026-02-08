package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/account"
)

const schema = `
CREATE TABLE IF NOT EXISTS accounts (
    id TEXT PRIMARY KEY,
    game_center_id TEXT UNIQUE,
    nickname TEXT NOT NULL DEFAULT '',
    is_guest BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_accounts_game_center_id ON accounts(game_center_id);
`

// PostgresStore implements AccountStore using PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore connects to PostgreSQL and initializes the schema.
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresStore{pool: pool}, nil
}

// FindByGameCenterID looks up an account by Game Center player ID.
func (s *PostgresStore) FindByGameCenterID(ctx context.Context, gcID string) (*account.Account, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, game_center_id, nickname, is_guest, created_at, last_login_at
		 FROM accounts WHERE game_center_id = $1`, gcID)

	acc, err := scanAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return acc, err
}

// FindByID looks up an account by internal ID.
func (s *PostgresStore) FindByID(ctx context.Context, id string) (*account.Account, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, game_center_id, nickname, is_guest, created_at, last_login_at
		 FROM accounts WHERE id = $1`, id)

	acc, err := scanAccount(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return acc, err
}

// Create inserts a new account.
func (s *PostgresStore) Create(ctx context.Context, acc *account.Account) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO accounts (id, game_center_id, nickname, is_guest, created_at, last_login_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		acc.ID, acc.GameCenterID, acc.Nickname, acc.IsGuest, acc.CreatedAt, acc.LastLoginAt)
	return err
}

// UpdateLastLogin updates the last login timestamp.
func (s *PostgresStore) UpdateLastLogin(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE accounts SET last_login_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

// UpdateNickname updates the account nickname.
func (s *PostgresStore) UpdateNickname(ctx context.Context, id string, nickname string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE accounts SET nickname = $1 WHERE id = $2`, nickname, id)
	return err
}

// Close releases database resources.
func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}

func scanAccount(row pgx.Row) (*account.Account, error) {
	var acc account.Account
	err := row.Scan(&acc.ID, &acc.GameCenterID, &acc.Nickname, &acc.IsGuest, &acc.CreatedAt, &acc.LastLoginAt)
	if err != nil {
		return nil, err
	}
	return &acc, nil
}
