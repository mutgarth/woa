package engine_test

import (
	"testing"
	"time"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
)

func TestTickEngine_RunsMultipleTicks(t *testing.T) {
	world := ecs.NewWorld()
	bus := engine.NewEventBus()
	eng := engine.NewTickEngine(world, bus, 50*time.Millisecond)
	go eng.Start()
	time.Sleep(160 * time.Millisecond)
	eng.Stop()
	if eng.CurrentTick() < 2 {
		t.Fatalf("expected at least 2 ticks, got %d", eng.CurrentTick())
	}
}
