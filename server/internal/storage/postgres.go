package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type DB struct {
	Pool *pgxpool.Pool
}

func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if err := runMigrations(databaseURL); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return &DB{Pool: pool}, nil
}

func runMigrations(databaseURL string) error {
	d, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, databaseURL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	DisplayName  string
}

func (db *DB) CreateUser(ctx context.Context, email, password, displayName string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &User{}
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, display_name)
		 VALUES ($1, $2, $3)
		 RETURNING id, email, password_hash, display_name`,
		email, string(hash), displayName,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (db *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName)
	if err != nil {
		return nil, err
	}
	return u, nil
}

type Agent struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Name      string
	AgentType string
}

func (db *DB) CreateAgent(ctx context.Context, ownerID uuid.UUID, name, agentType string) (*Agent, string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, "", err
	}
	hash := hashAPIKey(apiKey)
	a := &Agent{}
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO agents (owner_id, name, agent_type, api_key_hash)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, owner_id, name, agent_type`,
		ownerID, name, agentType, hash,
	).Scan(&a.ID, &a.OwnerID, &a.Name, &a.AgentType)
	if err != nil {
		return nil, "", fmt.Errorf("create agent: %w", err)
	}
	return a, apiKey, nil
}

func (db *DB) GetAgentByAPIKey(ctx context.Context, apiKey string) (*Agent, error) {
	hash := hashAPIKey(apiKey)
	a := &Agent{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, owner_id, name, agent_type FROM agents WHERE api_key_hash = $1`,
		hash,
	).Scan(&a.ID, &a.OwnerID, &a.Name, &a.AgentType)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (db *DB) ListAgentsByOwner(ctx context.Context, ownerID uuid.UUID) ([]Agent, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, owner_id, name, agent_type FROM agents WHERE owner_id = $1 ORDER BY created_at`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Name, &a.AgentType); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

func (db *DB) DeleteAgent(ctx context.Context, id, ownerID uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx,
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

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "woa_" + hex.EncodeToString(b), nil
}

func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
