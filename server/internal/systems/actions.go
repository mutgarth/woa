package systems

import (
	"encoding/json"
	"time"
	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
)

type ActionProcessor struct {
	bus         *engine.EventBus
	actionQueue <-chan wonet.IncomingAction
}

func NewActionProcessor(bus *engine.EventBus, queue <-chan wonet.IncomingAction) *ActionProcessor {
	return &ActionProcessor{bus: bus, actionQueue: queue}
}

func (s *ActionProcessor) Name() string { return "actions" }

func (s *ActionProcessor) Update(world *ecs.World, tick uint64) {
	for {
		select {
		case action := <-s.actionQueue:
			s.processAction(world, action)
		default:
			return
		}
	}
}

func (s *ActionProcessor) processAction(world *ecs.World, action wonet.IncomingAction) {
	entity := world.Entity(action.EntityID)
	if entity == nil { return }
	switch action.Envelope.Type {
	case "heartbeat":
		s.handleHeartbeat(entity, action.Raw)
	case "set_status":
		s.handleSetStatus(entity, action.Raw)
	case "set_zone":
		s.handleSetZone(entity, action.Raw)
	}
}

func (s *ActionProcessor) handleHeartbeat(entity *ecs.Entity, raw []byte) {
	var msg wonet.HeartbeatMessage
	if err := json.Unmarshal(raw, &msg); err != nil { return }
	p := entity.Get(components.PresenceType)
	if p == nil { return }
	presence := p.(*components.Presence)
	presence.LastHeartbeat = time.Now()
	if msg.Status != "" { presence.Status = msg.Status }
	if msg.Zone != "" { presence.Zone = msg.Zone }
}

func (s *ActionProcessor) handleSetStatus(entity *ecs.Entity, raw []byte) {
	var msg wonet.SetStatusMessage
	if err := json.Unmarshal(raw, &msg); err != nil { return }
	p := entity.Get(components.PresenceType)
	if p == nil { return }
	presence := p.(*components.Presence)
	oldStatus := presence.Status
	presence.Status = msg.Status
	if oldStatus != msg.Status {
		identity := entity.Get(components.IdentityType).(*components.Identity)
		s.bus.Publish(engine.Event{
			Type: "agent_status",
			Payload: map[string]any{
				"agent_id": entity.ID.String(), "name": identity.Name,
				"status": msg.Status, "zone": presence.Zone,
			},
			Scope: engine.GlobalScope(),
		})
	}
}

func (s *ActionProcessor) handleSetZone(entity *ecs.Entity, raw []byte) {
	var msg wonet.SetZoneMessage
	if err := json.Unmarshal(raw, &msg); err != nil { return }
	p := entity.Get(components.PresenceType)
	if p == nil { return }
	presence := p.(*components.Presence)
	presence.Zone = msg.Zone
}
