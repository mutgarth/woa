// server/internal/adapters/postgres/user_repo.go
package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/auth"
)

type UserRepo struct {
	db *DB
}

func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, email, passwordHash, displayName string) (*auth.User, error) {
	u := &auth.User{}
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, display_name)
		 VALUES ($1, $2, $3)
		 RETURNING id, email, password_hash, display_name`,
		email, passwordHash, displayName,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	u := &auth.User{}
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	u := &auth.User{}
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName)
	if err != nil {
		return nil, err
	}
	return u, nil
}
