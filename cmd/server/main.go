package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/config"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/handler"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/room"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/store"
	"github.com/ugaemi/gyeongdohalsaram-server/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

func main() {
	cfg := config.Load()
	setupLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	accountStore, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	hub := ws.NewHub()
	rm := room.NewManager()
	router := handler.NewRouter(rm)

	hub.OnMessage = router.HandleMessage
	hub.OnDisconnect = router.HandleDisconnect

	go hub.Run()

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{Addr: addr}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		slog.Info("shutting down server...")
		if err := server.Shutdown(context.Background()); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
		if err := accountStore.Close(); err != nil {
			slog.Error("database close error", "error", err)
		}
		slog.Info("database connection closed")
		cancel()
	}()

	slog.Info("server starting", "addr", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func handleWebSocket(hub *ws.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}

	client := ws.NewClient(fmt.Sprintf("client-%d", hub.ClientCount()+1), hub, conn)
	hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}

func setupLogger(cfg *config.Config) {
	var h slog.Handler
	opts := &slog.HandlerOptions{}

	switch cfg.LogLevel {
	case "debug":
		opts.Level = slog.LevelDebug
	case "warn":
		opts.Level = slog.LevelWarn
	case "error":
		opts.Level = slog.LevelError
	default:
		opts.Level = slog.LevelInfo
	}

	switch cfg.LogFormat {
	case "json":
		h = slog.NewJSONHandler(os.Stdout, opts)
	default:
		h = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(h))
}
