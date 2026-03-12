package net_test

import (
	"testing"

	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
)

func TestHubCreation(t *testing.T) {
	w := ecs.NewWorld()
	bus := engine.NewEventBus()
	hub := wonet.NewHub(w, bus, nil) // nil auth for unit test
	if hub.ActionQueue == nil {
		t.Fatal("ActionQueue should be initialized")
	}
}
