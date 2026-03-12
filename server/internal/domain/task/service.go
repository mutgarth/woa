package task

import (
	"context"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/guild"
)

type Service struct {
	tasks  TaskRepository
	guilds guild.GuildRepository
}

func NewService(tasks TaskRepository, guilds guild.GuildRepository) *Service {
	return &Service{tasks: tasks, guilds: guilds}
}

func (s *Service) Post(ctx context.Context, guildID, agentID uuid.UUID, title, description string, priority Priority) (*Task, error) {
	_, err := s.guilds.GetMembership(ctx, guildID, agentID)
	if err != nil {
		return nil, domain.ErrNotMember
	}
	t := NewTask(guildID, agentID, title, description, priority)
	if err := s.tasks.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Claim(ctx context.Context, taskID, agentID uuid.UUID) (*Task, error) {
	t, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	_, err = s.guilds.GetMembership(ctx, t.GuildID, agentID)
	if err != nil {
		return nil, domain.ErrNotMember
	}
	if err := t.Claim(agentID); err != nil {
		return nil, err
	}
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Complete(ctx context.Context, taskID, agentID uuid.UUID, result string) (*Task, error) {
	t, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if err := t.Complete(agentID, result); err != nil {
		return nil, err
	}
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Abandon(ctx context.Context, taskID, agentID uuid.UUID) (*Task, error) {
	t, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if err := t.Abandon(agentID); err != nil {
		return nil, err
	}
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Fail(ctx context.Context, taskID, agentID uuid.UUID) (*Task, error) {
	t, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if err := t.Fail(agentID); err != nil {
		return nil, err
	}
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Cancel(ctx context.Context, taskID, agentID uuid.UUID) (*Task, error) {
	t, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if err := t.Cancel(agentID); err != nil {
		return nil, err
	}
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) List(ctx context.Context, guildID uuid.UUID, status *Status, limit, offset int) ([]Task, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	return s.tasks.ListByGuild(ctx, guildID, status, limit, offset)
}
