package task

import (
	"time"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain"
)

type Status string

const (
	StatusOpen      Status = "open"
	StatusClaimed   Status = "claimed"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

type Task struct {
	ID          uuid.UUID
	GuildID     uuid.UUID
	PostedBy    uuid.UUID
	ClaimedBy   *uuid.UUID
	Title       string
	Description string
	Priority    Priority
	Status      Status
	Result      string
	CreatedAt   time.Time
	ClaimedAt   *time.Time
	CompletedAt *time.Time
}

func NewTask(guildID, postedBy uuid.UUID, title, description string, priority Priority) *Task {
	return &Task{
		ID: uuid.New(), GuildID: guildID, PostedBy: postedBy,
		Title: title, Description: description, Priority: priority,
		Status: StatusOpen, CreatedAt: time.Now(),
	}
}

func (t *Task) Claim(agentID uuid.UUID) error {
	if t.Status != StatusOpen {
		return domain.ErrInvalidTransition
	}
	t.Status = StatusClaimed
	t.ClaimedBy = &agentID
	now := time.Now()
	t.ClaimedAt = &now
	return nil
}

func (t *Task) Complete(agentID uuid.UUID, result string) error {
	if t.Status != StatusClaimed {
		return domain.ErrInvalidTransition
	}
	if t.ClaimedBy == nil || *t.ClaimedBy != agentID {
		return domain.ErrNotClaimer
	}
	t.Status = StatusCompleted
	t.Result = result
	now := time.Now()
	t.CompletedAt = &now
	return nil
}

func (t *Task) Abandon(agentID uuid.UUID) error {
	if t.Status != StatusClaimed {
		return domain.ErrInvalidTransition
	}
	if t.ClaimedBy == nil || *t.ClaimedBy != agentID {
		return domain.ErrNotClaimer
	}
	t.Status = StatusOpen
	t.ClaimedBy = nil
	t.ClaimedAt = nil
	return nil
}

func (t *Task) Fail(agentID uuid.UUID) error {
	if t.Status != StatusClaimed {
		return domain.ErrInvalidTransition
	}
	if t.ClaimedBy == nil || *t.ClaimedBy != agentID {
		return domain.ErrNotClaimer
	}
	t.Status = StatusFailed
	now := time.Now()
	t.CompletedAt = &now
	return nil
}

func (t *Task) Cancel(callerID uuid.UUID) error {
	if t.Status != StatusOpen {
		return domain.ErrInvalidTransition
	}
	if t.PostedBy != callerID {
		return domain.ErrPermissionDenied
	}
	t.Status = StatusCancelled
	now := time.Now()
	t.CompletedAt = &now
	return nil
}
