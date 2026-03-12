package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/guild"
)

type GuildRepo struct {
	db *DB
}

func NewGuildRepo(db *DB) *GuildRepo {
	return &GuildRepo{db: db}
}

func (r *GuildRepo) Create(ctx context.Context, g *guild.Guild) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO guilds (id, name, description, owner_id, visibility, max_members, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		g.ID, g.Name, g.Description, g.OwnerID, g.Visibility, g.MaxMembers, g.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create guild: %w", err)
	}
	return nil
}

func (r *GuildRepo) GetByID(ctx context.Context, id uuid.UUID) (*guild.Guild, error) {
	g := &guild.Guild{}
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, name, description, owner_id, visibility, max_members, created_at
		 FROM guilds WHERE id = $1`, id,
	).Scan(&g.ID, &g.Name, &g.Description, &g.OwnerID, &g.Visibility, &g.MaxMembers, &g.CreatedAt)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return g, nil
}

func (r *GuildRepo) GetByName(ctx context.Context, name string) (*guild.Guild, error) {
	g := &guild.Guild{}
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, name, description, owner_id, visibility, max_members, created_at
		 FROM guilds WHERE name = $1`, name,
	).Scan(&g.ID, &g.Name, &g.Description, &g.OwnerID, &g.Visibility, &g.MaxMembers, &g.CreatedAt)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return g, nil
}

func (r *GuildRepo) List(ctx context.Context, limit, offset int) ([]guild.Guild, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, name, description, owner_id, visibility, max_members, created_at
		 FROM guilds WHERE visibility = 'public' ORDER BY created_at LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var guilds []guild.Guild
	for rows.Next() {
		var g guild.Guild
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.OwnerID, &g.Visibility, &g.MaxMembers, &g.CreatedAt); err != nil {
			return nil, err
		}
		guilds = append(guilds, g)
	}
	return guilds, nil
}

func (r *GuildRepo) AddMember(ctx context.Context, m *guild.Membership) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO guild_members (guild_id, agent_id, role, joined_at)
		 VALUES ($1, $2, $3, $4)`,
		m.GuildID, m.AgentID, string(m.Role), m.JoinedAt,
	)
	if err != nil {
		return domain.ErrAlreadyMember
	}
	return nil
}

func (r *GuildRepo) RemoveMember(ctx context.Context, guildID, agentID uuid.UUID) error {
	tag, err := r.db.Pool.Exec(ctx,
		`DELETE FROM guild_members WHERE guild_id = $1 AND agent_id = $2`,
		guildID, agentID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotMember
	}
	return nil
}

func (r *GuildRepo) GetMembership(ctx context.Context, guildID, agentID uuid.UUID) (*guild.Membership, error) {
	m := &guild.Membership{}
	var role string
	err := r.db.Pool.QueryRow(ctx,
		`SELECT guild_id, agent_id, role, joined_at FROM guild_members
		 WHERE guild_id = $1 AND agent_id = $2`, guildID, agentID,
	).Scan(&m.GuildID, &m.AgentID, &role, &m.JoinedAt)
	if err != nil {
		return nil, domain.ErrNotMember
	}
	m.Role = guild.Role(role)
	return m, nil
}

func (r *GuildRepo) ListMembers(ctx context.Context, guildID uuid.UUID) ([]guild.Membership, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT guild_id, agent_id, role, joined_at FROM guild_members
		 WHERE guild_id = $1 ORDER BY joined_at`, guildID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []guild.Membership
	for rows.Next() {
		var m guild.Membership
		var role string
		if err := rows.Scan(&m.GuildID, &m.AgentID, &role, &m.JoinedAt); err != nil {
			return nil, err
		}
		m.Role = guild.Role(role)
		members = append(members, m)
	}
	return members, nil
}

func (r *GuildRepo) CountMembers(ctx context.Context, guildID uuid.UUID) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM guild_members WHERE guild_id = $1`, guildID,
	).Scan(&count)
	return count, err
}

func (r *GuildRepo) GetGuildByAgent(ctx context.Context, agentID uuid.UUID) (*guild.Guild, *guild.Membership, error) {
	m := &guild.Membership{}
	g := &guild.Guild{}
	var role string
	err := r.db.Pool.QueryRow(ctx,
		`SELECT g.id, g.name, g.description, g.owner_id, g.visibility, g.max_members, g.created_at,
		        gm.guild_id, gm.agent_id, gm.role, gm.joined_at
		 FROM guild_members gm JOIN guilds g ON g.id = gm.guild_id
		 WHERE gm.agent_id = $1`, agentID,
	).Scan(&g.ID, &g.Name, &g.Description, &g.OwnerID, &g.Visibility, &g.MaxMembers, &g.CreatedAt,
		&m.GuildID, &m.AgentID, &role, &m.JoinedAt)
	if err != nil {
		return nil, nil, domain.ErrNotMember
	}
	m.Role = guild.Role(role)
	return g, m, nil
}
