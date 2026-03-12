package systems

import (
	"encoding/json"
	"time"

	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
)

type ActionRouter struct {
	bus         *engine.EventBus
	actionQueue <-chan wonet.IncomingAction
	guild       ActionHandler
	task        ActionHandler
	chat        ActionHandler
}

func NewActionRouter(bus *engine.EventBus, queue <-chan wonet.IncomingAction, guild, task, chat ActionHandler) *ActionRouter {
	return &ActionRouter{bus: bus, actionQueue: queue, guild: guild, task: task, chat: chat}
}

func (r *ActionRouter) Name() string { return "action_router" }

func (r *ActionRouter) Update(world *ecs.World, tick uint64) {
	for {
		select {
		case action := <-r.actionQueue:
			r.route(world, action)
		default:
			return
		}
	}
}

func (r *ActionRouter) route(world *ecs.World, action wonet.IncomingAction) {
	switch action.Envelope.Type {
	// Presence actions (handled inline)
	case "heartbeat":
		r.handleHeartbeat(world, action)
	case "set_status":
		r.handleSetStatus(world, action)
	case "set_zone":
		r.handleSetZone(world, action)

	// Guild actions
	case "guild_create", "guild_join", "guild_leave":
		if r.guild != nil {
			r.guild.HandleAction(world, action)
		}

	// Task actions
	case "task_post", "task_claim", "task_complete", "task_abandon", "task_fail", "task_cancel":
		if r.task != nil {
			r.task.HandleAction(world, action)
		}

	// Chat actions
	case "message":
		if r.chat != nil {
			r.chat.HandleAction(world, action)
		}
	}
}

// Presence handlers (moved from ActionProcessor)
func (r *ActionRouter) handleHeartbeat(world *ecs.World, action wonet.IncomingAction) {
	entity := world.Entity(action.EntityID)
	if entity == nil {
		return
	}
	var msg wonet.HeartbeatMessage
	if err := json.Unmarshal(action.Raw, &msg); err != nil {
		return
	}
	p := entity.Get(components.PresenceType)
	if p == nil {
		return
	}
	presence := p.(*components.Presence)
	presence.LastHeartbeat = time.Now()
	if msg.Status != "" {
		presence.Status = msg.Status
	}
	if msg.Zone != "" {
		presence.Zone = msg.Zone
	}
}

func (r *ActionRouter) handleSetStatus(world *ecs.World, action wonet.IncomingAction) {
	entity := world.Entity(action.EntityID)
	if entity == nil {
		return
	}
	var msg wonet.SetStatusMessage
	if err := json.Unmarshal(action.Raw, &msg); err != nil {
		return
	}
	p := entity.Get(components.PresenceType)
	if p == nil {
		return
	}
	presence := p.(*components.Presence)
	oldStatus := presence.Status
	presence.Status = msg.Status
	if oldStatus != msg.Status {
		identity := entity.Get(components.IdentityType).(*components.Identity)
		r.bus.Publish(engine.Event{
			Type: "agent_status",
			Payload: map[string]any{
				"agent_id": entity.ID.String(), "name": identity.Name,
				"status": msg.Status, "zone": presence.Zone,
			},
			Scope: engine.GlobalScope(),
		})
	}
}

func (r *ActionRouter) handleSetZone(world *ecs.World, action wonet.IncomingAction) {
	entity := world.Entity(action.EntityID)
	if entity == nil {
		return
	}
	var msg wonet.SetZoneMessage
	if err := json.Unmarshal(action.Raw, &msg); err != nil {
		return
	}
	p := entity.Get(components.PresenceType)
	if p == nil {
		return
	}
	presence := p.(*components.Presence)
	presence.Zone = msg.Zone
}
