package systems

import (
	"time"
	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
)

type PresenceSystem struct {
	bus     *engine.EventBus
	timeout time.Duration
}

func NewPresenceSystem(bus *engine.EventBus, timeout time.Duration) *PresenceSystem {
	return &PresenceSystem{bus: bus, timeout: timeout}
}

func (s *PresenceSystem) Name() string { return "presence" }

func (s *PresenceSystem) Update(world *ecs.World, tick uint64) {
	now := time.Now()
	world.Each(func(e *ecs.Entity) {
		if !e.HasAll(components.PresenceType, components.IdentityType) { return }
		p := e.Get(components.PresenceType).(*components.Presence)
		if p.Status == "offline" { return }
		if now.Sub(p.LastHeartbeat) > s.timeout {
			p.Status = "offline"
			identity := e.Get(components.IdentityType).(*components.Identity)
			s.bus.Publish(engine.Event{
				Type:    "agent_offline",
				Payload: map[string]any{"agent_id": e.ID.String(), "name": identity.Name, "reason": "timeout"},
				Scope:   engine.GlobalScope(),
			})
		}
	})
}
