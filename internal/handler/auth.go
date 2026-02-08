package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/ugaemi/gyeongdohalsaram-server/internal/account"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/auth"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/store"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

const authTimeout = 10 * time.Second

// AuthHandler handles authentication messages.
type AuthHandler struct {
	verifier *auth.GameCenterVerifier
	store    store.AccountStore
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(verifier *auth.GameCenterVerifier, store store.AccountStore) *AuthHandler {
	return &AuthHandler{
		verifier: verifier,
		store:    store,
	}
}

type authenticateRequest struct {
	Method string `json:"method"`

	// Game Center fields
	PlayerID     string `json:"player_id,omitempty"`
	BundleID     string `json:"bundle_id,omitempty"`
	PublicKeyURL string `json:"public_key_url,omitempty"`
	Signature    string `json:"signature,omitempty"`
	Salt         string `json:"salt,omitempty"`
	Timestamp    uint64 `json:"timestamp,omitempty"`

	// Guest fields
	Nickname string `json:"nickname,omitempty"`
}

type authSuccessResponse struct {
	Success   bool   `json:"success"`
	AccountID string `json:"account_id"`
	Nickname  string `json:"nickname"`
}

type authFailureResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// HandleAuthenticate processes an authentication request.
func (h *AuthHandler) HandleAuthenticate(client *ws.Client, msg ws.Message) {
	if client.Authenticated {
		client.SendMessage(ws.NewErrorMessage("already authenticated"))
		return
	}

	var req authenticateRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.sendFailure(client, "invalid auth data")
		return
	}

	switch req.Method {
	case "game_center":
		h.handleGameCenter(client, req)
	case "guest":
		h.handleGuest(client, req)
	default:
		h.sendFailure(client, "unknown auth method: "+req.Method)
	}
}

func (h *AuthHandler) handleGameCenter(client *ws.Client, req authenticateRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cred := &auth.GameCenterCredential{
		PlayerID:     req.PlayerID,
		BundleID:     req.BundleID,
		PublicKeyURL: req.PublicKeyURL,
		Signature:    req.Signature,
		Salt:         req.Salt,
		Timestamp:    req.Timestamp,
	}

	if err := h.verifier.Verify(ctx, cred); err != nil {
		slog.Warn("game center verification failed", "error", err, "client", client.ID)
		h.sendFailure(client, "verification failed")
		return
	}

	// Find or create account
	acc, err := h.store.FindByGameCenterID(ctx, req.PlayerID)
	if err != nil {
		slog.Error("failed to find account", "error", err)
		h.sendFailure(client, "internal error")
		return
	}

	if acc == nil {
		acc = account.NewGameCenterAccount(req.PlayerID, req.Nickname)
		if err := h.store.Create(ctx, acc); err != nil {
			slog.Error("failed to create account", "error", err)
			h.sendFailure(client, "internal error")
			return
		}
		slog.Info("new game center account created", "account_id", acc.ID, "gc_id", req.PlayerID)
	} else {
		_ = h.store.UpdateLastLogin(ctx, acc.ID)
	}

	h.authenticateClient(client, acc)
}

func (h *AuthHandler) handleGuest(client *ws.Client, req authenticateRequest) {
	if req.Nickname == "" {
		h.sendFailure(client, "nickname is required for guest login")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	acc := account.NewGuestAccount(req.Nickname)
	if err := h.store.Create(ctx, acc); err != nil {
		slog.Error("failed to create guest account", "error", err)
		h.sendFailure(client, "internal error")
		return
	}

	slog.Info("new guest account created", "account_id", acc.ID, "nickname", req.Nickname)
	h.authenticateClient(client, acc)
}

func (h *AuthHandler) authenticateClient(client *ws.Client, acc *account.Account) {
	client.AccountID = acc.ID
	client.Authenticated = true

	resp, _ := ws.NewMessage(ws.TypeAuthResult, authSuccessResponse{
		Success:   true,
		AccountID: acc.ID,
		Nickname:  acc.Nickname,
	})
	client.SendMessage(resp)

	slog.Info("client authenticated", "client", client.ID, "account_id", acc.ID)
}

func (h *AuthHandler) sendFailure(client *ws.Client, errMsg string) {
	resp, _ := ws.NewMessage(ws.TypeAuthResult, authFailureResponse{
		Success: false,
		Error:   errMsg,
	})
	client.SendMessage(resp)
}

// StartAuthTimeout closes the connection if the client doesn't authenticate in time.
func (h *AuthHandler) StartAuthTimeout(client *ws.Client) {
	time.AfterFunc(authTimeout, func() {
		if !client.Authenticated {
			slog.Info("auth timeout, closing connection", "client", client.ID)
			client.SendMessage(ws.NewErrorMessage("authentication timeout"))
			client.Conn.Close()
		}
	})
}
