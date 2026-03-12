package guild

import (
	"context"

	"github.com/google/uuid"
)

type GuildRepository interface {
	Create(ctx context.Context, guild *Guild) error
	GetByID(ctx context.Context, id uuid.UUID) (*Guild, error)
	GetByName(ctx context.Context, name string) (*Guild, error)
	List(ctx context.Context, limit, offset int) ([]Guild, error)
	AddMember(ctx context.Context, m *Membership) error
	RemoveMember(ctx context.Context, guildID, agentID uuid.UUID) error
	GetMembership(ctx context.Context, guildID, agentID uuid.UUID) (*Membership, error)
	ListMembers(ctx context.Context, guildID uuid.UUID) ([]Membership, error)
	CountMembers(ctx context.Context, guildID uuid.UUID) (int, error)
	GetGuildByAgent(ctx context.Context, agentID uuid.UUID) (*Guild, *Membership, error)
}
