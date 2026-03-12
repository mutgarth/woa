package chat

import (
	"time"

	"github.com/google/uuid"
)

type Channel string

const (
	ChannelGuild  Channel = "guild"
	ChannelDirect Channel = "direct"
)

type Message struct {
	ID        uuid.UUID
	Channel   Channel
	GuildID   *uuid.UUID
	FromAgent uuid.UUID
	ToAgent   *uuid.UUID
	Content   string
	CreatedAt time.Time
}

func NewGuildMessage(guildID, fromAgent uuid.UUID, content string) *Message {
	return &Message{
		ID:        uuid.New(),
		Channel:   ChannelGuild,
		GuildID:   &guildID,
		FromAgent: fromAgent,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

func NewDirectMessage(from, to uuid.UUID, content string) *Message {
	return &Message{
		ID:        uuid.New(),
		Channel:   ChannelDirect,
		FromAgent: from,
		ToAgent:   &to,
		Content:   content,
		CreatedAt: time.Now(),
	}
}
