package components

import "github.com/google/uuid"

const IdentityType = "identity"

type Identity struct {
	Name      string
	AgentType string
	OwnerID   uuid.UUID
	AgentDBID uuid.UUID
}

func (i *Identity) ComponentType() string { return IdentityType }
