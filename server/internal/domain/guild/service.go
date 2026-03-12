package guild

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/agent"
)

type Service struct {
	guilds GuildRepository
	agents agent.AgentRepository
}

func NewService(guilds GuildRepository, agents agent.AgentRepository) *Service {
	return &Service{guilds: guilds, agents: agents}
}

func (s *Service) Create(ctx context.Context, name, description, visibility string, ownerUserID, creatorAgentID uuid.UUID) (*Guild, error) {
	g := &Guild{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		OwnerID:     ownerUserID,
		Visibility:  visibility,
		MaxMembers:  50,
		CreatedAt:   time.Now(),
	}
	if err := s.guilds.Create(ctx, g); err != nil {
		return nil, err
	}
	mem := &Membership{
		GuildID:  g.ID,
		AgentID:  creatorAgentID,
		Role:     RoleOwner,
		JoinedAt: time.Now(),
	}
	if err := s.guilds.AddMember(ctx, mem); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Service) Join(ctx context.Context, guildName string, agentID uuid.UUID) (*Membership, error) {
	g, err := s.guilds.GetByName(ctx, guildName)
	if err != nil {
		return nil, err
	}
	count, err := s.guilds.CountMembers(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	if count >= g.MaxMembers {
		return nil, domain.ErrGuildFull
	}
	mem := &Membership{
		GuildID:  g.ID,
		AgentID:  agentID,
		Role:     RoleMember,
		JoinedAt: time.Now(),
	}
	if err := s.guilds.AddMember(ctx, mem); err != nil {
		return nil, err
	}
	return mem, nil
}

func (s *Service) Leave(ctx context.Context, agentID uuid.UUID) error {
	g, mem, err := s.guilds.GetGuildByAgent(ctx, agentID)
	if err != nil {
		return err
	}
	if mem.Role == RoleOwner {
		return domain.ErrPermissionDenied
	}
	return s.guilds.RemoveMember(ctx, g.ID, agentID)
}

func (s *Service) Members(ctx context.Context, guildID uuid.UUID) ([]Membership, error) {
	return s.guilds.ListMembers(ctx, guildID)
}

func (s *Service) GetAgentGuild(ctx context.Context, agentID uuid.UUID) (*Guild, *Membership, error) {
	return s.guilds.GetGuildByAgent(ctx, agentID)
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]Guild, error) {
	return s.guilds.List(ctx, limit, offset)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Guild, error) {
	return s.guilds.GetByID(ctx, id)
}

func (s *Service) GetWithMembers(ctx context.Context, guildID uuid.UUID) (*Guild, []Membership, error) {
	g, err := s.guilds.GetByID(ctx, guildID)
	if err != nil {
		return nil, nil, err
	}
	members, err := s.guilds.ListMembers(ctx, guildID)
	if err != nil {
		return nil, nil, err
	}
	return g, members, nil
}
