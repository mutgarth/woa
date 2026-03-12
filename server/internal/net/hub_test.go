package net_test

import (
	"testing"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
	"github.com/lucasmeneses/world-of-agents/server/internal/storage"
)

func TestNewHub_CreatesWithActionQueue(t *testing.T) {
	world := ecs.NewWorld()
	bus := engine.NewEventBus()
	hub := wonet.NewHub(world, bus, (*storage.DB)(nil), nil)
	if hub == nil { t.Fatal("expected hub to be created") }
	if hub.ActionQueue == nil { t.Fatal("expected action queue to be initialized") }
}
