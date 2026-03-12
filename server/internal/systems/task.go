// server/internal/systems/task.go
package systems

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/task"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
)

type TaskSystem struct {
	svc *task.Service
	bus *engine.EventBus
}

func NewTaskSystem(svc *task.Service, bus *engine.EventBus) *TaskSystem {
	return &TaskSystem{svc: svc, bus: bus}
}

func (s *TaskSystem) HandleAction(world *ecs.World, action wonet.IncomingAction) {
	entity := world.Entity(action.EntityID)
	if entity == nil {
		return
	}

	// Resolve agent ID from Identity component
	identity := entity.Get(components.IdentityType)
	if identity == nil {
		return
	}
	agentID := identity.(*components.Identity).AgentDBID

	// Resolve guild from GuildMembership component
	gm := entity.Get(components.GuildMembershipType)

	ctx := context.Background()

	switch action.Envelope.Type {
	case "task_post":
		s.handlePost(ctx, world, action, entity, agentID, gm)
	case "task_claim":
		s.handleTaskAction(ctx, world, action, entity, agentID, gm, "claim")
	case "task_complete":
		s.handleTaskAction(ctx, world, action, entity, agentID, gm, "complete")
	case "task_abandon":
		s.handleTaskAction(ctx, world, action, entity, agentID, gm, "abandon")
	case "task_fail":
		s.handleTaskAction(ctx, world, action, entity, agentID, gm, "fail")
	case "task_cancel":
		s.handleTaskAction(ctx, world, action, entity, agentID, gm, "cancel")
	}
}

func (s *TaskSystem) handlePost(ctx context.Context, world *ecs.World, action wonet.IncomingAction, entity *ecs.Entity, agentID uuid.UUID, gm ecs.Component) {
	if gm == nil {
		sendError(entity, "NOT_IN_GUILD", "you must be in a guild to post tasks")
		return
	}
	guildID := gm.(*components.GuildMembership).GuildID

	var msg wonet.TaskPostMessage
	if err := json.Unmarshal(action.Raw, &msg); err != nil {
		sendError(entity, "BAD_REQUEST", "invalid task_post message")
		return
	}

	priority := task.Priority(msg.Payload.Priority)
	if priority == "" {
		priority = task.PriorityNormal
	}

	t, err := s.svc.Post(ctx, guildID, agentID, msg.Payload.Title, msg.Payload.Description, priority)
	if err != nil {
		sendError(entity, "TASK_ERROR", err.Error())
		return
	}

	s.bus.Publish(engine.Event{
		Type: "task_created",
		Payload: map[string]any{
			"task": map[string]any{
				"id": t.ID.String(), "guild_id": t.GuildID.String(),
				"posted_by": t.PostedBy.String(), "title": t.Title,
				"description": t.Description, "priority": string(t.Priority),
				"status": string(t.Status),
			},
		},
		Scope: engine.GuildScope(guildID),
	})

	slog.Info("task posted", "task_id", t.ID.String(), "guild_id", guildID.String(), "title", t.Title)
}

func (s *TaskSystem) handleTaskAction(ctx context.Context, world *ecs.World, action wonet.IncomingAction, entity *ecs.Entity, agentID uuid.UUID, gm ecs.Component, op string) {
	var msg wonet.TaskActionMessage
	if err := json.Unmarshal(action.Raw, &msg); err != nil {
		sendError(entity, "BAD_REQUEST", "invalid task action message")
		return
	}

	taskID, err := uuid.Parse(msg.Payload.TaskID)
	if err != nil {
		sendError(entity, "BAD_REQUEST", "invalid task_id")
		return
	}

	var t *task.Task
	switch op {
	case "claim":
		t, err = s.svc.Claim(ctx, taskID, agentID)
	case "complete":
		t, err = s.svc.Complete(ctx, taskID, agentID, msg.Payload.Result)
	case "abandon":
		t, err = s.svc.Abandon(ctx, taskID, agentID)
	case "fail":
		t, err = s.svc.Fail(ctx, taskID, agentID)
	case "cancel":
		t, err = s.svc.Cancel(ctx, taskID, agentID)
	}

	if err != nil {
		sendError(entity, "TASK_ERROR", err.Error())
		return
	}

	// Resolve guild scope
	var guildID uuid.UUID
	if gm != nil {
		guildID = gm.(*components.GuildMembership).GuildID
	} else {
		guildID = t.GuildID
	}

	eventType := "task_" + string(t.Status) // task_claimed, task_completed, etc.
	if op == "abandon" {
		eventType = "task_abandoned"
	}

	payload := map[string]any{
		"task_id":  t.ID.String(),
		"agent_id": agentID.String(),
		"status":   string(t.Status),
	}
	if t.Result != "" {
		payload["result"] = t.Result
	}

	s.bus.Publish(engine.Event{
		Type:    eventType,
		Payload: payload,
		Scope:   engine.GuildScope(guildID),
	})

	slog.Info("task action", "op", op, "task_id", t.ID.String(), "agent_id", agentID.String())
}
