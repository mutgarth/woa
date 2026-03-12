package components

import "github.com/google/uuid"

const GuildMembershipType = "guild_membership"

type GuildMembership struct {
	GuildID   uuid.UUID
	GuildName string
	Role      string
}

func (g *GuildMembership) ComponentType() string { return GuildMembershipType }
