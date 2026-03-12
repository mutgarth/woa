package systems

import (
	"encoding/json"

	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
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
	if len(events) == 0 {
		return
	}

	tickMsg := map[string]any{
		"type":   "tick",
		"number": tick,
	}

	// Group events by scope for efficient delivery
	// Global events go to everyone
	// Guild events go to matching guild members
	// Direct events go to specific agents

	// Build per-entity event lists
	entityEvents := make(map[ecs.EntityID][]any)

	for _, event := range events {
		payload := map[string]any{
			"type":    event.Type,
			"payload": event.Payload,
		}

		switch event.Scope.Type {
		case "global", "":
			// Send to all entities with connections
			world.Each(func(e *ecs.Entity) {
				if e.Get(components.ConnectionType) != nil {
					entityEvents[e.ID] = append(entityEvents[e.ID], payload)
				}
			})

		case "guild":
			if event.Scope.GuildID == nil {
				continue
			}
			targetGuildID := *event.Scope.GuildID
			world.Each(func(e *ecs.Entity) {
				if e.Get(components.ConnectionType) == nil {
					return
				}
				gm := e.Get(components.GuildMembershipType)
				if gm == nil {
					return
				}
				if gm.(*components.GuildMembership).GuildID == targetGuildID {
					entityEvents[e.ID] = append(entityEvents[e.ID], payload)
				}
			})

		case "direct":
			world.Each(func(e *ecs.Entity) {
				if e.Get(components.ConnectionType) == nil {
					return
				}
				identity := e.Get(components.IdentityType)
				if identity == nil {
					return
				}
				agentID := identity.(*components.Identity).AgentDBID
				for _, targetID := range event.Scope.AgentIDs {
					if agentID == targetID {
						entityEvents[e.ID] = append(entityEvents[e.ID], payload)
						break
					}
				}
			})
		}
	}

	// Send tick messages to each entity with their filtered events
	world.Each(func(e *ecs.Entity) {
		c := e.Get(components.ConnectionType)
		if c == nil {
			return
		}
		conn := c.(*components.Connection)

		evts := entityEvents[e.ID]
		if len(evts) == 0 {
			// Still send empty tick for heartbeat purposes
			evts = []any{}
		}

		msg := make(map[string]any)
		for k, v := range tickMsg {
			msg[k] = v
		}
		msg["events"] = evts

		data, err := json.Marshal(msg)
		if err != nil {
			return
		}
		select {
		case conn.Send <- data:
		default:
		}
	})
}
