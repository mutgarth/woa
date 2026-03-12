package agent

import (
	"time"

	"github.com/google/uuid"
)

type AgentType string

const (
	AgentTypeClaude AgentType = "claude"
	AgentTypeCodex  AgentType = "codex"
	AgentTypeGemini AgentType = "gemini"
	AgentTypeCustom AgentType = "custom"
)

type Agent struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Name      string
	AgentType AgentType
	CreatedAt time.Time
}
