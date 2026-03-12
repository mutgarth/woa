package systems_test

import (
	"testing"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	"github.com/lucasmeneses/world-of-agents/server/internal/systems"
)

func TestBroadcastSystem_NoEventsNoBroadcast(t *testing.T) {
	world := ecs.NewWorld()
	bus := engine.NewEventBus()
	sys := systems.NewBroadcastSystem(bus)
	sys.Update(world, 1)
}

func TestBroadcastSystem_Name(t *testing.T) {
	bus := engine.NewEventBus()
	sys := systems.NewBroadcastSystem(bus)
	if sys.Name() != "broadcast" { t.Fatalf("expected 'broadcast', got %q", sys.Name()) }
}
