package systems_test

import (
	"testing"
	"time"
	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	"github.com/lucasmeneses/world-of-agents/server/internal/systems"
)

func TestPresenceSystem_MarksOfflineAfterTimeout(t *testing.T) {
	world := ecs.NewWorld()
	bus := engine.NewEventBus()
	sys := systems.NewPresenceSystem(bus, 10*time.Second)
	e := ecs.NewEntity()
	e.Add(&components.Identity{Name: "stale-agent"})
	e.Add(&components.Presence{Status: components.StatusOnline, LastHeartbeat: time.Now().Add(-15 * time.Second)})
	world.AddEntity(e)
	sys.Update(world, 1)
	p := e.Get(components.PresenceType).(*components.Presence)
	if p.Status != "offline" { t.Fatalf("expected offline, got %s", p.Status) }
	events := bus.Drain()
	if len(events) == 0 { t.Fatal("expected agent_offline event") }
	if events[0].Type != "agent_offline" { t.Fatalf("expected agent_offline, got %s", events[0].Type) }
}

func TestPresenceSystem_KeepsOnlineWithRecentHeartbeat(t *testing.T) {
	world := ecs.NewWorld()
	bus := engine.NewEventBus()
	sys := systems.NewPresenceSystem(bus, 10*time.Second)
	e := ecs.NewEntity()
	e.Add(&components.Identity{Name: "fresh-agent"})
	e.Add(&components.Presence{Status: components.StatusOnline, LastHeartbeat: time.Now()})
	world.AddEntity(e)
	sys.Update(world, 1)
	p := e.Get(components.PresenceType).(*components.Presence)
	if p.Status != components.StatusOnline { t.Fatalf("expected online, got %s", p.Status) }
	events := bus.Drain()
	if len(events) != 0 { t.Fatalf("expected no events, got %d", len(events)) }
}
