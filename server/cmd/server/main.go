package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cryptoadapter "github.com/lucasmeneses/world-of-agents/server/internal/adapters/crypto"
	jwtadapter "github.com/lucasmeneses/world-of-agents/server/internal/adapters/jwt"
	"github.com/lucasmeneses/world-of-agents/server/internal/adapters/postgres"
	redisadapter "github.com/lucasmeneses/world-of-agents/server/internal/adapters/redis"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/auth"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/chat"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/guild"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/task"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
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

	// Driven adapters
	db, err := postgres.NewDB(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redisCache, err := redisadapter.NewPresenceCache(redisAddr)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisCache.Close()
	_ = redisCache // wired into PresenceSystem in a future phase

	userRepo := postgres.NewUserRepo(db)
	agentRepo := postgres.NewAgentRepo(db)
	guildRepo := postgres.NewGuildRepo(db)
	taskRepo := postgres.NewTaskRepo(db)
	messageRepo := postgres.NewMessageRepo(db)
	tokenService := jwtadapter.NewTokenService(jwtSecret)
	hashService := cryptoadapter.NewHashService()

	// Domain services
	authService := auth.NewService(userRepo, agentRepo, tokenService, hashService)
	guildService := guild.NewService(guildRepo, agentRepo)
	taskService := task.NewService(taskRepo, guildRepo)
	chatService := chat.NewService(messageRepo, guildRepo)

	// ECS + Engine
	world := ecs.NewWorld()
	bus := engine.NewEventBus()

	// Driving adapters
	hub := wonet.NewHub(world, bus, authService)

	guildSystem := systems.NewGuildSystem(guildService, bus)
	taskSystem := systems.NewTaskSystem(taskService, bus)
	chatSystem := systems.NewChatSystem(chatService, bus)
	actionRouter := systems.NewActionRouter(bus, hub.ActionQueue, guildSystem, taskSystem, chatSystem)
	presenceSystem := systems.NewPresenceSystem(bus, 15*time.Second)
	broadcastSystem := systems.NewBroadcastSystem(bus)

	world.AddSystem(actionRouter)
	world.AddSystem(presenceSystem)
	world.AddSystem(broadcastSystem)

	eng := engine.NewTickEngine(world, bus, tickRate)
	go eng.Start()

	mux := http.NewServeMux()
	rest := wonet.NewREST(authService, guildService, taskService)
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
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
