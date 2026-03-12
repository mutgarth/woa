package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(addr string) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &RedisStore{client: client}, nil
}

func (r *RedisStore) Close() error {
	return r.client.Close()
}

type PresenceData struct {
	Status    string    `json:"status"`
	Zone      string    `json:"zone"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *RedisStore) SetPresence(ctx context.Context, agentID string, data PresenceData, ttl time.Duration) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, "agent:"+agentID+":presence", b, ttl).Err()
}

func (r *RedisStore) GetPresence(ctx context.Context, agentID string) (*PresenceData, error) {
	b, err := r.client.Get(ctx, "agent:"+agentID+":presence").Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var data PresenceData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *RedisStore) DeletePresence(ctx context.Context, agentID string) error {
	return r.client.Del(ctx, "agent:"+agentID+":presence").Err()
}
