package systems

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/guild"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
)

type GuildSystem struct {
	service *guild.Service
	bus     *engine.EventBus
}

func NewGuildSystem(service *guild.Service, bus *engine.EventBus) *GuildSystem {
	return &GuildSystem{service: service, bus: bus}
}

func (s *GuildSystem) HandleAction(world *ecs.World, action wonet.IncomingAction) {
	entity := world.Entity(action.EntityID)
	if entity == nil {
		return
	}
	ctx := context.Background()

	switch action.Envelope.Type {
	case "guild_create":
		s.handleCreate(ctx, entity, action.Raw)
	case "guild_join":
		s.handleJoin(ctx, entity, action.Raw)
	case "guild_leave":
		s.handleLeave(ctx, entity)
	}
}

func (s *GuildSystem) handleCreate(ctx context.Context, entity *ecs.Entity, raw []byte) {
	var msg wonet.GuildCreateMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}
	identity := entity.Get(components.IdentityType).(*components.Identity)
	visibility := msg.Payload.Visibility
	if visibility == "" {
		visibility = "public"
	}

	g, err := s.service.Create(ctx, msg.Payload.Name, msg.Payload.Description, visibility, identity.OwnerID, identity.AgentDBID)
	if err != nil {
		sendError(entity, "GUILD_CREATE_FAILED", err.Error())
		return
	}

	entity.Add(&components.GuildMembership{
		GuildID: g.ID, GuildName: g.Name, Role: string(guild.RoleOwner),
	})

	s.bus.Publish(engine.Event{
		Type: "guild_created",
		Payload: map[string]any{
			"guild": map[string]any{
				"id": g.ID.String(), "name": g.Name,
				"description": g.Description, "visibility": g.Visibility,
			},
		},
		Scope: engine.GlobalScope(),
	})
	slog.Info("guild created", "name", g.Name, "by", identity.Name)
}

func (s *GuildSystem) handleJoin(ctx context.Context, entity *ecs.Entity, raw []byte) {
	var msg wonet.GuildJoinMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}
	identity := entity.Get(components.IdentityType).(*components.Identity)

	mem, err := s.service.Join(ctx, msg.Payload.GuildName, identity.AgentDBID)
	if err != nil {
		sendError(entity, "GUILD_JOIN_FAILED", err.Error())
		return
	}

	g, _ := s.service.GetByID(ctx, mem.GuildID)
	entity.Add(&components.GuildMembership{
		GuildID: g.ID, GuildName: g.Name, Role: string(mem.Role),
	})

	s.bus.Publish(engine.Event{
		Type: "member_joined",
		Payload: map[string]any{
			"guild_id": g.ID.String(),
			"agent":    map[string]any{"id": identity.AgentDBID.String(), "name": identity.Name, "type": identity.AgentType},
		},
		Scope: engine.GuildScope(g.ID),
	})
	slog.Info("agent joined guild", "agent", identity.Name, "guild", g.Name)
}

func (s *GuildSystem) handleLeave(ctx context.Context, entity *ecs.Entity) {
	identity := entity.Get(components.IdentityType).(*components.Identity)

	g, _, err := s.service.GetAgentGuild(ctx, identity.AgentDBID)
	if err != nil {
		sendError(entity, "GUILD_NOT_MEMBER", "not a guild member")
		return
	}

	if err := s.service.Leave(ctx, identity.AgentDBID); err != nil {
		sendError(entity, "GUILD_LEAVE_FAILED", err.Error())
		return
	}

	entity.Remove(components.GuildMembershipType)

	s.bus.Publish(engine.Event{
		Type: "member_left",
		Payload: map[string]any{
			"guild_id": g.ID.String(),
			"agent_id": identity.AgentDBID.String(),
		},
		Scope: engine.GuildScope(g.ID),
	})
	slog.Info("agent left guild", "agent", identity.Name, "guild", g.Name)
}
