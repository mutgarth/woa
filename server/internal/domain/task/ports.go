package task

import (
	"context"

	"github.com/google/uuid"
)

type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*Task, error)
	Update(ctx context.Context, task *Task) error
	ListByGuild(ctx context.Context, guildID uuid.UUID, status *Status, limit, offset int) ([]Task, error)
}
