package guild

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleMember Role = "member"
)

type Guild struct {
	ID          uuid.UUID
	Name        string
	Description string
	OwnerID     uuid.UUID // user ID who owns the guild
	Visibility  string    // "public", "private"
	MaxMembers  int
	CreatedAt   time.Time
}

type Membership struct {
	GuildID  uuid.UUID
	AgentID  uuid.UUID
	Role     Role
	JoinedAt time.Time
}
