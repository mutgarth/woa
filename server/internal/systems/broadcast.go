package systems

import (
	"encoding/json"
	"log/slog"
	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	"github.com/lucasmeneses/world-of-agents/server/internal/net"
)

type BroadcastSystem struct {
	bus *engine.EventBus
}

func NewBroadcastSystem(bus *engine.EventBus) *BroadcastSystem {
	return &BroadcastSystem{bus: bus}
}

func (s *BroadcastSystem) Name() string { return "broadcast" }

func (s *BroadcastSystem) Update(world *ecs.World, tick uint64) {
	events := s.bus.Drain()
	if len(events) == 0 { return }
	tickMsg := net.TickMessage{Type: "tick", Number: tick, Events: make([]any, len(events))}
	for i, e := range events { tickMsg.Events[i] = e }
	data, err := json.Marshal(tickMsg)
	if err != nil {
		slog.Error("failed to marshal tick message", "error", err)
		return
	}
	world.Each(func(e *ecs.Entity) {
		c := e.Get(components.ConnectionType)
		if c == nil { return }
		conn := c.(*components.Connection)
		select {
		case conn.Send <- data:
		default:
		}
	})
}
