package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
	"github.com/lucasmeneses/world-of-agents/server/internal/storage"
	"github.com/lucasmeneses/world-of-agents/server/internal/systems"
)

func main() {
	slog.Info("World of Agents server starting...")

	dbURL := envOr("DATABASE_URL", "postgres://woa:woa_dev@localhost:5433/woa?sslmode=disable")
	redisAddr := envOr("REDIS_ADDR", "localhost:6380")
	jwtSecret := envOr("JWT_SECRET", "woa-dev-secret-change-in-production!")
	listenAddr := envOr("LISTEN_ADDR", ":8083")
	tickRate := 200 * time.Millisecond

	ctx := context.Background()
	db, err := storage.NewDB(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redisStore, err := storage.NewRedisStore(redisAddr)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisStore.Close()
	_ = redisStore // will be wired into PresenceSystem in Phase 2

	world := ecs.NewWorld()
	bus := engine.NewEventBus()

	auth := wonet.NewAuth(jwtSecret)
	hub := wonet.NewHub(world, bus, db, auth)

	actionProcessor := systems.NewActionProcessor(bus, hub.ActionQueue)
	presenceSystem := systems.NewPresenceSystem(bus, 15*time.Second)
	broadcastSystem := systems.NewBroadcastSystem(bus)

	world.AddSystem(actionProcessor)
	world.AddSystem(presenceSystem)
	world.AddSystem(broadcastSystem)

	eng := engine.NewTickEngine(world, bus, tickRate)
	go eng.Start()

	mux := http.NewServeMux()
	rest := &wonet.REST{DB: db, Auth: auth}
	rest.RegisterRoutes(mux)
	mux.HandleFunc("GET /ws", hub.HandleWebSocket)

	server := &http.Server{Addr: listenAddr, Handler: mux}

	go func() {
		slog.Info("listening", "addr", listenAddr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	eng.Stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
	slog.Info("goodbye")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" { return v }
	return fallback
}
