package account

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGameCenterAccount(t *testing.T) {
	acc := NewGameCenterAccount("G:123456", "테스트유저")

	assert.NotEmpty(t, acc.ID)
	require.NotNil(t, acc.GameCenterID)
	assert.Equal(t, "G:123456", *acc.GameCenterID)
	assert.Equal(t, "테스트유저", acc.Nickname)
	assert.False(t, acc.IsGuest)
	assert.False(t, acc.CreatedAt.IsZero())
	assert.False(t, acc.LastLoginAt.IsZero())
}

func TestNewGuestAccount(t *testing.T) {
	acc := NewGuestAccount("게스트")

	assert.NotEmpty(t, acc.ID)
	assert.Nil(t, acc.GameCenterID)
	assert.Equal(t, "게스트", acc.Nickname)
	assert.True(t, acc.IsGuest)
	assert.False(t, acc.CreatedAt.IsZero())
	assert.False(t, acc.LastLoginAt.IsZero())
}

func TestNewGameCenterAccount_UniqueIDs(t *testing.T) {
	acc1 := NewGameCenterAccount("G:111", "유저1")
	acc2 := NewGameCenterAccount("G:222", "유저2")

	assert.NotEqual(t, acc1.ID, acc2.ID)
}
