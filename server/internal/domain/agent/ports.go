package agent

import (
	"context"

	"github.com/google/uuid"
)

type AgentRepository interface {
	Create(ctx context.Context, agent *Agent, apiKeyHash string) error
	GetByAPIKeyHash(ctx context.Context, hash string) (*Agent, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]Agent, error)
	Delete(ctx context.Context, id, ownerID uuid.UUID) error
}
