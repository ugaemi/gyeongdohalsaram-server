package store

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/account"
)

func getTestDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping PostgreSQL integration test")
	}
	return url
}

func setupTestStore(t *testing.T) *PostgresStore {
	t.Helper()
	url := getTestDatabaseURL(t)
	ctx := context.Background()

	s, err := NewPostgresStore(ctx, url)
	require.NoError(t, err)

	// Clean up accounts table for test isolation
	_, err = s.pool.Exec(ctx, "DELETE FROM accounts")
	require.NoError(t, err)

	t.Cleanup(func() {
		s.Close()
	})

	return s
}

func TestPostgresStore_CreateAndFindByID(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	acc := account.NewGameCenterAccount("G:test-001", "테스트유저")
	err := s.Create(ctx, acc)
	require.NoError(t, err)

	found, err := s.FindByID(ctx, acc.ID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, acc.ID, found.ID)
	assert.Equal(t, "테스트유저", found.Nickname)
	assert.False(t, found.IsGuest)
	require.NotNil(t, found.GameCenterID)
	assert.Equal(t, "G:test-001", *found.GameCenterID)
}

func TestPostgresStore_CreateAndFindByGameCenterID(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	acc := account.NewGameCenterAccount("G:test-002", "유저2")
	err := s.Create(ctx, acc)
	require.NoError(t, err)

	found, err := s.FindByGameCenterID(ctx, "G:test-002")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, acc.ID, found.ID)
}

func TestPostgresStore_FindByGameCenterID_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	found, err := s.FindByGameCenterID(ctx, "G:nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPostgresStore_FindByID_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	found, err := s.FindByID(ctx, "nonexistent-id")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPostgresStore_CreateGuestAccount(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	acc := account.NewGuestAccount("게스트유저")
	err := s.Create(ctx, acc)
	require.NoError(t, err)

	found, err := s.FindByID(ctx, acc.ID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.True(t, found.IsGuest)
	assert.Nil(t, found.GameCenterID)
	assert.Equal(t, "게스트유저", found.Nickname)
}

func TestPostgresStore_UpdateNickname(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	acc := account.NewGuestAccount("이전닉네임")
	err := s.Create(ctx, acc)
	require.NoError(t, err)

	err = s.UpdateNickname(ctx, acc.ID, "새닉네임")
	require.NoError(t, err)

	found, err := s.FindByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, "새닉네임", found.Nickname)
}

func TestPostgresStore_UpdateLastLogin(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	acc := account.NewGameCenterAccount("G:test-login", "로그인테스트")
	err := s.Create(ctx, acc)
	require.NoError(t, err)

	err = s.UpdateLastLogin(ctx, acc.ID)
	require.NoError(t, err)

	found, err := s.FindByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.True(t, found.LastLoginAt.After(acc.CreatedAt) || found.LastLoginAt.Equal(acc.CreatedAt))
}

func TestPostgresStore_DuplicateGameCenterID(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	acc1 := account.NewGameCenterAccount("G:duplicate", "유저1")
	err := s.Create(ctx, acc1)
	require.NoError(t, err)

	acc2 := account.NewGameCenterAccount("G:duplicate", "유저2")
	err = s.Create(ctx, acc2)
	assert.Error(t, err)
}
