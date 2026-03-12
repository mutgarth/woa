// server/internal/adapters/postgres/agent_repo.go
package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/agent"
)

type AgentRepo struct {
	db *DB
}

func NewAgentRepo(db *DB) *AgentRepo {
	return &AgentRepo{db: db}
}

func (r *AgentRepo) Create(ctx context.Context, a *agent.Agent, apiKeyHash string) error {
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO agents (id, owner_id, name, agent_type, api_key_hash)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		a.ID, a.OwnerID, a.Name, string(a.AgentType), apiKeyHash,
	).Scan(&a.ID)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	return nil
}

func (r *AgentRepo) GetByAPIKeyHash(ctx context.Context, hash string) (*agent.Agent, error) {
	a := &agent.Agent{}
	var agentType string
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, owner_id, name, agent_type FROM agents WHERE api_key_hash = $1`,
		hash,
	).Scan(&a.ID, &a.OwnerID, &a.Name, &agentType)
	if err != nil {
		return nil, err
	}
	a.AgentType = agent.AgentType(agentType)
	return a, nil
}

func (r *AgentRepo) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]agent.Agent, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, owner_id, name, agent_type FROM agents WHERE owner_id = $1 ORDER BY created_at`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []agent.Agent
	for rows.Next() {
		var a agent.Agent
		var agentType string
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Name, &agentType); err != nil {
			return nil, err
		}
		a.AgentType = agent.AgentType(agentType)
		agents = append(agents, a)
	}
	return agents, nil
}

func (r *AgentRepo) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	tag, err := r.db.Pool.Exec(ctx,
		`DELETE FROM agents WHERE id = $1 AND owner_id = $2`, id, ownerID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}
	return nil
}
