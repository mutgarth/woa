package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/chat"
)

type MessageRepo struct {
	db *DB
}

func NewMessageRepo(db *DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) Create(ctx context.Context, msg *chat.Message) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO messages (id, channel, guild_id, from_agent, to_agent, content, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		msg.ID, string(msg.Channel), msg.GuildID, msg.FromAgent, msg.ToAgent, msg.Content, msg.CreatedAt,
	)
	return err
}

func (r *MessageRepo) ListByGuild(ctx context.Context, guildID uuid.UUID, limit int) ([]chat.Message, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, channel, guild_id, from_agent, to_agent, content, created_at
		 FROM messages WHERE guild_id = $1 ORDER BY created_at DESC LIMIT $2`,
		guildID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []chat.Message
	for rows.Next() {
		var m chat.Message
		var channel string
		var guildID *uuid.UUID
		var toAgent *uuid.UUID
		var createdAt time.Time

		if err := rows.Scan(&m.ID, &channel, &guildID, &m.FromAgent, &toAgent, &m.Content, &createdAt); err != nil {
			return nil, err
		}
		m.Channel = chat.Channel(channel)
		m.GuildID = guildID
		m.ToAgent = toAgent
		m.CreatedAt = createdAt
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (r *MessageRepo) ListDirect(ctx context.Context, agentA, agentB uuid.UUID, limit int) ([]chat.Message, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, channel, guild_id, from_agent, to_agent, content, created_at
		 FROM messages
		 WHERE channel = 'direct'
		   AND ((from_agent = $1 AND to_agent = $2) OR (from_agent = $2 AND to_agent = $1))
		 ORDER BY created_at DESC LIMIT $3`,
		agentA, agentB, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []chat.Message
	for rows.Next() {
		var m chat.Message
		var channel string
		var guildID *uuid.UUID
		var toAgent *uuid.UUID
		var createdAt time.Time

		if err := rows.Scan(&m.ID, &channel, &guildID, &m.FromAgent, &toAgent, &m.Content, &createdAt); err != nil {
			return nil, err
		}
		m.Channel = chat.Channel(channel)
		m.GuildID = guildID
		m.ToAgent = toAgent
		m.CreatedAt = createdAt
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
