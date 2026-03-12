package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type PresenceCache struct {
	client *goredis.Client
}

func NewPresenceCache(addr string) (*PresenceCache, error) {
	client := goredis.NewClient(&goredis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &PresenceCache{client: client}, nil
}

func (r *PresenceCache) Close() error {
	return r.client.Close()
}

type PresenceData struct {
	Status    string    `json:"status"`
	Zone      string    `json:"zone"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *PresenceCache) SetPresence(ctx context.Context, agentID string, data PresenceData, ttl time.Duration) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, "agent:"+agentID+":presence", b, ttl).Err()
}

func (r *PresenceCache) GetPresence(ctx context.Context, agentID string) (*PresenceData, error) {
	b, err := r.client.Get(ctx, "agent:"+agentID+":presence").Bytes()
	if err == goredis.Nil {
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

func (r *PresenceCache) DeletePresence(ctx context.Context, agentID string) error {
	return r.client.Del(ctx, "agent:"+agentID+":presence").Err()
}
