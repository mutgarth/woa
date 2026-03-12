package auth

import (
	"context"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash, displayName string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type TokenService interface {
	Generate(userID uuid.UUID, email string) (string, error)
	Validate(token string) (*Claims, error)
}

type HashService interface {
	HashPassword(password string) (string, error)
	CheckPassword(hash, password string) error
	HashAPIKey(key string) string
}
