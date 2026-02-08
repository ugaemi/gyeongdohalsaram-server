package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/account"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/auth"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/room"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

// mockAccountStore implements store.AccountStore for testing.
type mockAccountStore struct {
	accounts map[string]*account.Account // id -> account
	byGCID   map[string]*account.Account // game_center_id -> account
}

func newMockAccountStore() *mockAccountStore {
	return &mockAccountStore{
		accounts: make(map[string]*account.Account),
		byGCID:   make(map[string]*account.Account),
	}
}

func (m *mockAccountStore) FindByGameCenterID(_ context.Context, gcID string) (*account.Account, error) {
	return m.byGCID[gcID], nil
}

func (m *mockAccountStore) FindByID(_ context.Context, id string) (*account.Account, error) {
	return m.accounts[id], nil
}

func (m *mockAccountStore) Create(_ context.Context, acc *account.Account) error {
	m.accounts[acc.ID] = acc
	if acc.GameCenterID != nil {
		m.byGCID[*acc.GameCenterID] = acc
	}
	return nil
}

func (m *mockAccountStore) UpdateLastLogin(_ context.Context, _ string) error { return nil }
func (m *mockAccountStore) UpdateNickname(_ context.Context, _, _ string) error { return nil }
func (m *mockAccountStore) Close() error { return nil }

// mockClient creates a test client that captures sent messages.
type sentMessage struct {
	Type string
	Data json.RawMessage
}

func newTestClient(id string) (*ws.Client, chan sentMessage) {
	ch := make(chan sentMessage, 10)
	client := &ws.Client{
		ID:   id,
		Send: make(chan []byte, 256),
	}

	// Read sent messages in background
	go func() {
		for data := range client.Send {
			var msg sentMessage
			json.Unmarshal(data, &msg)
			ch <- msg
		}
	}()

	return client, ch
}

func readResponse(t *testing.T, ch chan sentMessage) sentMessage {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for response")
		return sentMessage{}
	}
}

func TestHandleAuthenticate_Guest(t *testing.T) {
	store := newMockAccountStore()
	verifier := auth.NewGameCenterVerifier(nil, 0)
	handler := NewAuthHandler(verifier, store)

	client, ch := newTestClient("test-client-1")

	data, _ := json.Marshal(authenticateRequest{
		Method:   "guest",
		Nickname: "테스트유저",
	})
	msg := ws.Message{Type: ws.TypeAuthenticate, Data: data}

	handler.HandleAuthenticate(client, msg)

	resp := readResponse(t, ch)
	assert.Equal(t, ws.TypeAuthResult, resp.Type)

	var result authSuccessResponse
	require.NoError(t, json.Unmarshal(resp.Data, &result))
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.AccountID)
	assert.Equal(t, "테스트유저", result.Nickname)

	assert.True(t, client.Authenticated)
	assert.Equal(t, result.AccountID, client.AccountID)

	// Verify account was created in store
	assert.Len(t, store.accounts, 1)
}

func TestHandleAuthenticate_GuestNoNickname(t *testing.T) {
	store := newMockAccountStore()
	verifier := auth.NewGameCenterVerifier(nil, 0)
	handler := NewAuthHandler(verifier, store)

	client, ch := newTestClient("test-client-2")

	data, _ := json.Marshal(authenticateRequest{
		Method: "guest",
	})
	msg := ws.Message{Type: ws.TypeAuthenticate, Data: data}

	handler.HandleAuthenticate(client, msg)

	resp := readResponse(t, ch)
	var result authFailureResponse
	require.NoError(t, json.Unmarshal(resp.Data, &result))
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "nickname is required")
	assert.False(t, client.Authenticated)
}

func TestHandleAuthenticate_AlreadyAuthenticated(t *testing.T) {
	store := newMockAccountStore()
	verifier := auth.NewGameCenterVerifier(nil, 0)
	handler := NewAuthHandler(verifier, store)

	client, ch := newTestClient("test-client-3")
	client.Authenticated = true

	data, _ := json.Marshal(authenticateRequest{Method: "guest", Nickname: "test"})
	msg := ws.Message{Type: ws.TypeAuthenticate, Data: data}

	handler.HandleAuthenticate(client, msg)

	resp := readResponse(t, ch)
	assert.Equal(t, ws.TypeError, resp.Type)
}

func TestHandleAuthenticate_UnknownMethod(t *testing.T) {
	store := newMockAccountStore()
	verifier := auth.NewGameCenterVerifier(nil, 0)
	handler := NewAuthHandler(verifier, store)

	client, ch := newTestClient("test-client-4")

	data, _ := json.Marshal(authenticateRequest{Method: "unknown"})
	msg := ws.Message{Type: ws.TypeAuthenticate, Data: data}

	handler.HandleAuthenticate(client, msg)

	resp := readResponse(t, ch)
	var result authFailureResponse
	require.NoError(t, json.Unmarshal(resp.Data, &result))
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "unknown auth method")
}

func TestAuthGuard_UnauthenticatedBlocked(t *testing.T) {
	store := newMockAccountStore()
	verifier := auth.NewGameCenterVerifier(nil, 0)

	rm := room.NewManager()
	router := NewRouter(rm, verifier, store)

	client, ch := newTestClient("test-client-5")

	// Try to create room without authenticating
	data, _ := json.Marshal(map[string]string{"nickname": "test"})
	rawMsg, _ := json.Marshal(ws.Message{Type: ws.TypeCreateRoom, Data: data})

	router.HandleMessage(&ws.ClientMessage{Client: client, Data: rawMsg})

	resp := readResponse(t, ch)
	assert.Equal(t, ws.TypeError, resp.Type)

	var errMsg ws.ErrorMessage
	json.Unmarshal(resp.Data, &errMsg)
	assert.Equal(t, "authentication required", errMsg.Message)
}

func TestAuthGuard_AuthenticatedAllowed(t *testing.T) {
	store := newMockAccountStore()
	verifier := auth.NewGameCenterVerifier(nil, 0)

	rm := room.NewManager()
	router := NewRouter(rm, verifier, store)

	client, ch := newTestClient("test-client-6")

	// Authenticate first
	authData, _ := json.Marshal(authenticateRequest{Method: "guest", Nickname: "플레이어"})
	authMsg, _ := json.Marshal(ws.Message{Type: ws.TypeAuthenticate, Data: authData})
	router.HandleMessage(&ws.ClientMessage{Client: client, Data: authMsg})

	// Read auth response
	authResp := readResponse(t, ch)
	assert.Equal(t, ws.TypeAuthResult, authResp.Type)
	assert.True(t, client.Authenticated)

	// Now create room should work
	roomData, _ := json.Marshal(map[string]string{"nickname": "플레이어"})
	roomMsg, _ := json.Marshal(ws.Message{Type: ws.TypeCreateRoom, Data: roomData})
	router.HandleMessage(&ws.ClientMessage{Client: client, Data: roomMsg})

	roomResp := readResponse(t, ch)
	assert.Equal(t, ws.TypeCreateRoom, roomResp.Type)
}
