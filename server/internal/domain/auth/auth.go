package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/agent"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	DisplayName  string
}

type Claims struct {
	UserID uuid.UUID
	Email  string
}

type Service struct {
	users  UserRepository
	agents agent.AgentRepository
	tokens TokenService
	hasher HashService
}

func NewService(users UserRepository, agents agent.AgentRepository, tokens TokenService, hasher HashService) *Service {
	return &Service{users: users, agents: agents, tokens: tokens, hasher: hasher}
}

func (s *Service) Register(ctx context.Context, email, password, displayName string) (*User, string, error) {
	hash, err := s.hasher.HashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}
	user, err := s.users.Create(ctx, email, hash, displayName)
	if err != nil {
		return nil, "", err
	}
	token, err := s.tokens.Generate(user.ID, user.Email)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}
	return user, token, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*User, string, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if err := s.hasher.CheckPassword(user.PasswordHash, password); err != nil {
		return nil, "", err
	}
	token, err := s.tokens.Generate(user.ID, user.Email)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}
	return user, token, nil
}

func (s *Service) AuthenticateByAPIKey(ctx context.Context, apiKey string) (*agent.Agent, error) {
	hash := s.hasher.HashAPIKey(apiKey)
	return s.agents.GetByAPIKeyHash(ctx, hash)
}

func (s *Service) AuthenticateByToken(ctx context.Context, token string) (*Claims, error) {
	return s.tokens.Validate(token)
}

func (s *Service) CreateAgent(ctx context.Context, ownerID uuid.UUID, name string, agentType agent.AgentType) (*agent.Agent, string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, "", err
	}
	a := &agent.Agent{
		ID:        uuid.New(),
		OwnerID:   ownerID,
		Name:      name,
		AgentType: agentType,
	}
	hash := s.hasher.HashAPIKey(apiKey)
	if err := s.agents.Create(ctx, a, hash); err != nil {
		return nil, "", err
	}
	return a, apiKey, nil
}

func (s *Service) ListAgents(ctx context.Context, ownerID uuid.UUID) ([]agent.Agent, error) {
	return s.agents.ListByOwner(ctx, ownerID)
}

func (s *Service) DeleteAgent(ctx context.Context, id, ownerID uuid.UUID) error {
	return s.agents.Delete(ctx, id, ownerID)
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "woa_" + hex.EncodeToString(b), nil
}
