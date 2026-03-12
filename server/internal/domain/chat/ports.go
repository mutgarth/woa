package chat

import (
	"context"

	"github.com/google/uuid"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) error
	ListByGuild(ctx context.Context, guildID uuid.UUID, limit int) ([]Message, error)
	ListDirect(ctx context.Context, agentA, agentB uuid.UUID, limit int) ([]Message, error)
}
