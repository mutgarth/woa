package systems

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/chat"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
)

type ChatSystem struct {
	svc *chat.Service
	bus *engine.EventBus
}

func NewChatSystem(svc *chat.Service, bus *engine.EventBus) *ChatSystem {
	return &ChatSystem{svc: svc, bus: bus}
}

func (s *ChatSystem) HandleAction(world *ecs.World, action wonet.IncomingAction) {
	entity := world.Entity(action.EntityID)
	if entity == nil {
		return
	}

	identity := entity.Get(components.IdentityType)
	if identity == nil {
		return
	}
	agentID := identity.(*components.Identity).AgentDBID
	agentName := identity.(*components.Identity).Name

	var msg wonet.ChatMessage
	if err := json.Unmarshal(action.Raw, &msg); err != nil {
		sendError(entity, "BAD_REQUEST", "invalid message format")
		return
	}

	ctx := context.Background()

	switch chat.Channel(msg.Payload.Channel) {
	case chat.ChannelGuild:
		s.handleGuildMessage(ctx, entity, agentID, agentName, msg)
	case chat.ChannelDirect:
		s.handleDirectMessage(ctx, entity, agentID, agentName, msg)
	default:
		sendError(entity, "BAD_REQUEST", "invalid channel: must be 'guild' or 'direct'")
	}
}

func (s *ChatSystem) handleGuildMessage(ctx context.Context, entity *ecs.Entity, agentID uuid.UUID, agentName string, msg wonet.ChatMessage) {
	gm := entity.Get(components.GuildMembershipType)
	if gm == nil {
		sendError(entity, "NOT_IN_GUILD", "you must be in a guild to send guild messages")
		return
	}
	guildID := gm.(*components.GuildMembership).GuildID

	m, err := s.svc.SendGuild(ctx, guildID, agentID, msg.Payload.Content)
	if err != nil {
		sendError(entity, "CHAT_ERROR", err.Error())
		return
	}

	s.bus.Publish(engine.Event{
		Type: "message",
		Payload: map[string]any{
			"id":      m.ID.String(),
			"channel": "guild",
			"from": map[string]any{
				"agent_id": agentID.String(),
				"name":     agentName,
			},
			"content":    m.Content,
			"created_at": m.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
		Scope: engine.GuildScope(guildID),
	})

	slog.Info("guild message", "from", agentName, "guild_id", guildID.String())
}

func (s *ChatSystem) handleDirectMessage(ctx context.Context, entity *ecs.Entity, agentID uuid.UUID, agentName string, msg wonet.ChatMessage) {
	toID, err := uuid.Parse(msg.Payload.To)
	if err != nil {
		sendError(entity, "BAD_REQUEST", "invalid 'to' agent ID")
		return
	}

	m, err := s.svc.SendDirect(ctx, agentID, toID, msg.Payload.Content)
	if err != nil {
		sendError(entity, "CHAT_ERROR", err.Error())
		return
	}

	s.bus.Publish(engine.Event{
		Type: "message",
		Payload: map[string]any{
			"id":      m.ID.String(),
			"channel": "direct",
			"from": map[string]any{
				"agent_id": agentID.String(),
				"name":     agentName,
			},
			"to":         toID.String(),
			"content":    m.Content,
			"created_at": m.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
		Scope: engine.DirectScope(agentID, toID),
	})

	slog.Info("direct message", "from", agentName, "to", toID.String())
}
