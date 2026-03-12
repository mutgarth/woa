package chat

import (
	"context"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/guild"
)

type Service struct {
	messages MessageRepository
	guilds   guild.GuildRepository
}

func NewService(messages MessageRepository, guilds guild.GuildRepository) *Service {
	return &Service{messages: messages, guilds: guilds}
}

func (s *Service) SendGuild(ctx context.Context, guildID, fromAgent uuid.UUID, content string) (*Message, error) {
	_, err := s.guilds.GetMembership(ctx, guildID, fromAgent)
	if err != nil {
		return nil, domain.ErrNotMember
	}
	msg := NewGuildMessage(guildID, fromAgent, content)
	if err := s.messages.Create(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *Service) SendDirect(ctx context.Context, from, to uuid.UUID, content string) (*Message, error) {
	msg := NewDirectMessage(from, to, content)
	if err := s.messages.Create(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *Service) GuildHistory(ctx context.Context, guildID uuid.UUID, limit int) ([]Message, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	return s.messages.ListByGuild(ctx, guildID, limit)
}

func (s *Service) DirectHistory(ctx context.Context, agentA, agentB uuid.UUID, limit int) ([]Message, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	return s.messages.ListDirect(ctx, agentA, agentB, limit)
}
