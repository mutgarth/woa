# Phase 1: Core Server — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the WoA game server with ECS architecture, tick loop, WebSocket connections, auth, and presence — enough that two agents can connect via `wscat` and see each other online/offline in real-time.

**Architecture:** Single Go binary with ECS (Entity-Component-System) internals and a tick-based game loop at 5 Hz. WebSocket hub manages persistent connections. Auth via API key (first-message pattern). PostgreSQL for persistence, Redis for presence cache. All systems process entities with relevant components each tick.

**Tech Stack:** Go 1.22+, PostgreSQL 16, Redis 7, gorilla/websocket, pgx/v5, go-redis/v9, golang-migrate/v4, bcrypt, golang-jwt/v5

---

## File Structure

```
server/
├── cmd/server/main.go                    # Entry point, DI wiring, graceful shutdown
├── internal/
│   ├── ecs/
│   │   ├── world.go                      # World: entity registry, system runner
│   │   ├── world_test.go
│   │   ├── entity.go                     # Entity: ID + component map
│   │   ├── component.go                  # Component interface
│   │   └── system.go                     # System interface
│   ├── components/
│   │   ├── identity.go                   # Identity component
│   │   ├── presence.go                   # Presence component
│   │   └── connection.go                 # Connection component
│   ├── systems/
│   │   ├── presence.go                   # PresenceSystem
│   │   ├── presence_test.go
│   │   ├── broadcast.go                  # BroadcastSystem
│   │   ├── broadcast_test.go
│   │   └── actions.go                    # ActionProcessor
│   ├── engine/
│   │   ├── tick.go                       # Tick loop engine
│   │   ├── tick_test.go
│   │   ├── eventbus.go                   # Internal pub/sub event bus
│   │   └── eventbus_test.go
│   ├── net/
│   │   ├── protocol.go                   # Message types (JSON structs)
│   │   ├── hub.go                        # WebSocket hub (connection manager)
│   │   ├── hub_test.go
│   │   ├── auth.go                       # Auth: API key + JWT validation
│   │   ├── auth_test.go
│   │   ├── rest.go                       # REST API handlers
│   │   └── rest_test.go
│   └── storage/
│       ├── postgres.go                   # PostgreSQL connection + queries
│       ├── redis.go                      # Redis connection + presence ops
│       └── migrations/
│           └── 000001_initial_schema.up.sql
│           └── 000001_initial_schema.down.sql
├── go.mod
└── go.sum
docker-compose.yml                        # PostgreSQL + Redis
Makefile                                  # Build, run, test, migrate commands
```

---

## Chunk 1: Project Scaffold + ECS Core

### Task 1: Project scaffold

**Files:**
- Create: `server/go.mod`
- Create: `server/cmd/server/main.go`
- Create: `docker-compose.yml`
- Create: `Makefile`

- [ ] **Step 1: Create project directory and Go module**

```bash
cd /Users/lucasmeneses/mmoagens
mkdir -p server/cmd/server
cd server
go mod init github.com/lucasmeneses/world-of-agents/server
```

- [ ] **Step 2: Create docker-compose.yml**

Create `docker-compose.yml` at repo root:

```yaml
version: "3.9"
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: woa
      POSTGRES_PASSWORD: woa_dev
      POSTGRES_DB: woa
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  pgdata:
```

- [ ] **Step 3: Create Makefile**

Create `Makefile` at repo root:

```makefile
.PHONY: dev deps test build infra infra-down

infra:
	docker compose up -d

infra-down:
	docker compose down

deps:
	cd server && go mod tidy

build:
	cd server && go build -o bin/woa-server ./cmd/server

test:
	cd server && go test ./... -v -count=1

dev: infra
	cd server && go run ./cmd/server
```

- [ ] **Step 4: Create minimal main.go**

Create `server/cmd/server/main.go`:

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("World of Agents server starting...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}
```

- [ ] **Step 5: Verify build**

```bash
cd /Users/lucasmeneses/mmoagens && make build
```

Expected: binary created at `server/bin/woa-server`

- [ ] **Step 6: Commit**

```bash
git add server/go.mod server/cmd/server/main.go docker-compose.yml Makefile
git commit -m "feat: project scaffold with Go module, docker-compose, Makefile"
```

---

### Task 2: ECS core — Component and System interfaces

**Files:**
- Create: `server/internal/ecs/component.go`
- Create: `server/internal/ecs/system.go`
- Create: `server/internal/ecs/entity.go`

- [ ] **Step 1: Create Component interface**

Create `server/internal/ecs/component.go`:

```go
package ecs

// Component is a marker interface for pure data attached to entities.
type Component interface {
	ComponentType() string
}
```

- [ ] **Step 2: Create System interface**

Create `server/internal/ecs/system.go`:

```go
package ecs

// System processes entities each tick.
type System interface {
	// Name returns a unique identifier for this system.
	Name() string
	// Update runs the system logic for the given tick number.
	Update(world *World, tick uint64)
}
```

- [ ] **Step 3: Create Entity**

Create `server/internal/ecs/entity.go`:

```go
package ecs

import "github.com/google/uuid"

// EntityID uniquely identifies an entity in the world.
type EntityID = uuid.UUID

// Entity holds an ID and its components.
type Entity struct {
	ID         EntityID
	components map[string]Component
}

// NewEntity creates a new entity with a random UUID.
func NewEntity() *Entity {
	return &Entity{
		ID:         uuid.New(),
		components: make(map[string]Component),
	}
}

// NewEntityWithID creates an entity with a specific ID.
func NewEntityWithID(id EntityID) *Entity {
	return &Entity{
		ID:         id,
		components: make(map[string]Component),
	}
}

// Add attaches a component to the entity.
func (e *Entity) Add(c Component) {
	e.components[c.ComponentType()] = c
}

// Remove detaches a component by type.
func (e *Entity) Remove(componentType string) {
	delete(e.components, componentType)
}

// Get returns a component by type, or nil if not found.
func (e *Entity) Get(componentType string) Component {
	return e.components[componentType]
}

// Has checks if the entity has a component of the given type.
func (e *Entity) Has(componentType string) bool {
	_, ok := e.components[componentType]
	return ok
}

// HasAll checks if the entity has all the given component types.
func (e *Entity) HasAll(types ...string) bool {
	for _, t := range types {
		if !e.Has(t) {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Add uuid dependency**

```bash
cd /Users/lucasmeneses/mmoagens/server && go get github.com/google/uuid
```

- [ ] **Step 5: Verify build**

```bash
cd /Users/lucasmeneses/mmoagens && make build
```

- [ ] **Step 6: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/ecs/ server/go.mod server/go.sum
git commit -m "feat: ECS core — Component, System, and Entity types"
```

---

### Task 3: ECS World — entity registry and system runner

**Files:**
- Create: `server/internal/ecs/world.go`
- Create: `server/internal/ecs/world_test.go`

- [ ] **Step 1: Write failing test for World**

Create `server/internal/ecs/world_test.go`:

```go
package ecs_test

import (
	"testing"

	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
)

// stubComponent is a test component.
type stubComponent struct{ Value string }

func (s *stubComponent) ComponentType() string { return "stub" }

// tickCounter is a test system that counts how many times Update is called.
type tickCounter struct {
	Count      int
	LastTick   uint64
	SeenCount  int // how many entities with "stub" it saw
}

func (t *tickCounter) Name() string { return "tick_counter" }

func (t *tickCounter) Update(w *ecs.World, tick uint64) {
	t.Count++
	t.LastTick = tick
	t.SeenCount = 0
	w.Each(func(e *ecs.Entity) {
		if e.Has("stub") {
			t.SeenCount++
		}
	})
}

func TestWorld_AddEntityAndQuery(t *testing.T) {
	w := ecs.NewWorld()

	e := ecs.NewEntity()
	e.Add(&stubComponent{Value: "hello"})
	w.AddEntity(e)

	got := w.Entity(e.ID)
	if got == nil {
		t.Fatal("expected to find entity")
	}
	c := got.Get("stub").(*stubComponent)
	if c.Value != "hello" {
		t.Fatalf("expected 'hello', got %q", c.Value)
	}
}

func TestWorld_RemoveEntity(t *testing.T) {
	w := ecs.NewWorld()

	e := ecs.NewEntity()
	w.AddEntity(e)
	w.RemoveEntity(e.ID)

	if w.Entity(e.ID) != nil {
		t.Fatal("expected entity to be removed")
	}
}

func TestWorld_Tick(t *testing.T) {
	w := ecs.NewWorld()
	counter := &tickCounter{}
	w.AddSystem(counter)

	e := ecs.NewEntity()
	e.Add(&stubComponent{Value: "x"})
	w.AddEntity(e)

	w.Tick(42)

	if counter.Count != 1 {
		t.Fatalf("expected 1 tick, got %d", counter.Count)
	}
	if counter.LastTick != 42 {
		t.Fatalf("expected tick 42, got %d", counter.LastTick)
	}
	if counter.SeenCount != 1 {
		t.Fatalf("expected 1 entity with stub, got %d", counter.SeenCount)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/ecs/ -v -run TestWorld
```

Expected: FAIL — `NewWorld` not defined

- [ ] **Step 3: Implement World**

Create `server/internal/ecs/world.go`:

```go
package ecs

import "sync"

// World holds all entities and systems.
type World struct {
	mu       sync.RWMutex
	entities map[EntityID]*Entity
	systems  []System
}

// NewWorld creates an empty world.
func NewWorld() *World {
	return &World{
		entities: make(map[EntityID]*Entity),
	}
}

// AddEntity registers an entity in the world.
func (w *World) AddEntity(e *Entity) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.entities[e.ID] = e
}

// RemoveEntity removes an entity from the world.
func (w *World) RemoveEntity(id EntityID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.entities, id)
}

// Entity retrieves an entity by ID, or nil.
func (w *World) Entity(id EntityID) *Entity {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.entities[id]
}

// Each iterates over all entities. Holds a read lock.
func (w *World) Each(fn func(e *Entity)) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, e := range w.entities {
		fn(e)
	}
}

// AddSystem registers a system to run each tick.
func (w *World) AddSystem(s System) {
	w.systems = append(w.systems, s)
}

// Tick runs all systems in order for the given tick number.
func (w *World) Tick(tickNum uint64) {
	for _, s := range w.systems {
		s.Update(w, tickNum)
	}
}

// EntityCount returns the number of entities.
func (w *World) EntityCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.entities)
}
```

- [ ] **Step 4: Run tests**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/ecs/ -v -run TestWorld
```

Expected: all 3 tests PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/ecs/
git commit -m "feat: ECS World with entity registry, system runner, and tests"
```

---

### Task 4: Event Bus — internal pub/sub

**Files:**
- Create: `server/internal/engine/eventbus.go`
- Create: `server/internal/engine/eventbus_test.go`

- [ ] **Step 1: Write failing test**

Create `server/internal/engine/eventbus_test.go`:

```go
package engine_test

import (
	"testing"

	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
)

func TestEventBus_PublishAndDrain(t *testing.T) {
	bus := engine.NewEventBus()

	bus.Publish(engine.Event{Type: "agent_online", Payload: map[string]any{"name": "claude"}})
	bus.Publish(engine.Event{Type: "agent_online", Payload: map[string]any{"name": "codex"}})

	events := bus.Drain()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != "agent_online" {
		t.Fatalf("expected agent_online, got %s", events[0].Type)
	}

	// Drain again should be empty
	events = bus.Drain()
	if len(events) != 0 {
		t.Fatalf("expected 0 events after drain, got %d", len(events))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/engine/ -v
```

Expected: FAIL

- [ ] **Step 3: Implement EventBus**

Create `server/internal/engine/eventbus.go`:

```go
package engine

import "sync"

// Event is a typed payload produced by systems and consumed by BroadcastSystem.
type Event struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

// EventBus collects events during a tick and drains them for broadcasting.
type EventBus struct {
	mu     sync.Mutex
	events []Event
}

// NewEventBus creates an empty event bus.
func NewEventBus() *EventBus {
	return &EventBus{}
}

// Publish adds an event to the bus. Safe for concurrent use.
func (b *EventBus) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, e)
}

// Drain returns all events and clears the bus. Safe for concurrent use.
func (b *EventBus) Drain() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	events := b.events
	b.events = nil
	return events
}
```

- [ ] **Step 4: Run test**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/engine/ -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/engine/
git commit -m "feat: EventBus for internal pub/sub between systems"
```

---

### Task 5: Tick Engine

**Files:**
- Create: `server/internal/engine/tick.go`
- Create: `server/internal/engine/tick_test.go`

- [ ] **Step 1: Write failing test**

Create `server/internal/engine/tick_test.go`:

```go
package engine_test

import (
	"testing"
	"time"

	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
)

func TestTickEngine_RunsMultipleTicks(t *testing.T) {
	world := ecs.NewWorld()
	bus := engine.NewEventBus()
	eng := engine.NewTickEngine(world, bus, 50*time.Millisecond) // 20 Hz for fast test

	go eng.Start()

	// Let it run for ~150ms (should get at least 2 ticks)
	time.Sleep(160 * time.Millisecond)
	eng.Stop()

	if eng.CurrentTick() < 2 {
		t.Fatalf("expected at least 2 ticks, got %d", eng.CurrentTick())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/engine/ -v -run TestTickEngine
```

Expected: FAIL

- [ ] **Step 3: Implement TickEngine**

Create `server/internal/engine/tick.go`:

```go
package engine

import (
	"sync/atomic"
	"time"

	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
)

// TickEngine runs the ECS world at a fixed tick rate.
type TickEngine struct {
	world    *ecs.World
	bus      *EventBus
	interval time.Duration
	tick     atomic.Uint64
	stop     chan struct{}
}

// NewTickEngine creates a tick engine with the given interval between ticks.
func NewTickEngine(world *ecs.World, bus *EventBus, interval time.Duration) *TickEngine {
	return &TickEngine{
		world:    world,
		bus:      bus,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

// Start begins the tick loop. Blocks until Stop is called.
func (e *TickEngine) Start() {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stop:
			return
		case <-ticker.C:
			next := e.tick.Add(1)
			e.world.Tick(next)
		}
	}
}

// Stop signals the tick loop to exit.
func (e *TickEngine) Stop() {
	close(e.stop)
}

// CurrentTick returns the current tick number.
func (e *TickEngine) CurrentTick() uint64 {
	return e.tick.Load()
}

// Bus returns the event bus for this engine.
func (e *TickEngine) Bus() *EventBus {
	return e.bus
}
```

- [ ] **Step 4: Run test**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/engine/ -v -run TestTickEngine
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/engine/tick.go server/internal/engine/tick_test.go
git commit -m "feat: TickEngine runs ECS world at configurable Hz"
```

---

## Chunk 2: Components + Protocol + Storage

### Task 6: Components — Identity, Presence, Connection

**Files:**
- Create: `server/internal/components/identity.go`
- Create: `server/internal/components/presence.go`
- Create: `server/internal/components/connection.go`

- [ ] **Step 1: Create Identity component**

Create `server/internal/components/identity.go`:

```go
package components

import "github.com/google/uuid"

const IdentityType = "identity"

// Identity stores who this agent is.
type Identity struct {
	Name      string
	AgentType string    // claude, codex, gemini, custom
	OwnerID   uuid.UUID // user who owns this agent
	AgentDBID uuid.UUID // ID in the agents table
}

func (i *Identity) ComponentType() string { return IdentityType }
```

- [ ] **Step 2: Create Presence component**

Create `server/internal/components/presence.go`:

```go
package components

import "time"

const PresenceType = "presence"

// Status values for agents.
const (
	StatusOnline  = "online"
	StatusIdle    = "idle"
	StatusWorking = "working"
	StatusAway    = "away"
)

// Presence tracks agent liveness and location.
type Presence struct {
	Status        string
	Zone          string
	LastHeartbeat time.Time
}

func (p *Presence) ComponentType() string { return PresenceType }
```

- [ ] **Step 3: Create Connection component**

Create `server/internal/components/connection.go`:

```go
package components

import "github.com/gorilla/websocket"

const ConnectionType = "connection"

// Connection holds the WebSocket connection for a connected agent.
type Connection struct {
	Conn      *websocket.Conn
	SessionID string
	Send      chan []byte // outbound message channel
}

func (c *Connection) ComponentType() string { return ConnectionType }
```

- [ ] **Step 4: Add gorilla/websocket dependency**

```bash
cd /Users/lucasmeneses/mmoagens/server && go get github.com/gorilla/websocket
```

- [ ] **Step 5: Verify build**

```bash
cd /Users/lucasmeneses/mmoagens && make build
```

- [ ] **Step 6: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/components/ server/go.mod server/go.sum
git commit -m "feat: ECS components — Identity, Presence, Connection"
```

---

### Task 7: Protocol types

**Files:**
- Create: `server/internal/net/protocol.go`

- [ ] **Step 1: Create protocol message types**

Create `server/internal/net/protocol.go`:

```go
package net

import "encoding/json"

// Envelope is the generic message wrapper for all WebSocket messages.
type Envelope struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"-"` // original full message
}

// UnmarshalEnvelope extracts the type from a JSON message.
func UnmarshalEnvelope(data []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return env, err
	}
	env.Raw = data
	return env, nil
}

// --- Client → Server messages ---

type AuthMessage struct {
	Type   string `json:"type"` // "auth"
	APIKey string `json:"api_key,omitempty"`
	Token  string `json:"token,omitempty"`
}

type HeartbeatMessage struct {
	Type   string `json:"type"` // "heartbeat"
	Status string `json:"status,omitempty"`
	Zone   string `json:"zone,omitempty"`
}

type SetStatusMessage struct {
	Type   string `json:"type"` // "set_status"
	Status string `json:"status"`
}

type SetZoneMessage struct {
	Type string `json:"type"` // "set_zone"
	Zone string `json:"zone"`
}

// --- Server → Client messages ---

type AuthRequiredMessage struct {
	Type string `json:"type"` // "auth_required"
}

type WelcomeMessage struct {
	Type            string `json:"type"` // "welcome"
	AgentID         string `json:"agent_id"`
	ServerTick      uint64 `json:"server_tick"`
	ProtocolVersion int    `json:"protocol_version"`
}

type AgentOnlineMessage struct {
	Type  string    `json:"type"` // "agent_online"
	Agent AgentInfo `json:"agent"`
}

type AgentOfflineMessage struct {
	Type    string `json:"type"` // "agent_offline"
	AgentID string `json:"agent_id"`
	Reason  string `json:"reason"`
}

type AgentStatusMessage struct {
	Type    string `json:"type"` // "agent_status"
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
	Zone    string `json:"zone"`
}

type ErrorMessage struct {
	Type    string `json:"type"` // "error"
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TickMessage struct {
	Type   string `json:"type"` // "tick"
	Number uint64 `json:"number"`
	Events []any  `json:"events"`
}

// AgentInfo is a shared struct for agent identification.
type AgentInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/lucasmeneses/mmoagens && make build
```

- [ ] **Step 3: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/net/protocol.go
git commit -m "feat: WebSocket protocol message types"
```

---

### Task 8: PostgreSQL storage — connection and migrations

**Files:**
- Create: `server/internal/storage/postgres.go`
- Create: `server/internal/storage/migrations/000001_initial_schema.up.sql`
- Create: `server/internal/storage/migrations/000001_initial_schema.down.sql`

- [ ] **Step 1: Create initial migration (up)**

Create `server/internal/storage/migrations/000001_initial_schema.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    github_id     TEXT UNIQUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS agents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    agent_type    TEXT NOT NULL,
    api_key_hash  TEXT UNIQUE NOT NULL,
    capabilities  TEXT[] DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_id, name)
);

CREATE INDEX IF NOT EXISTS idx_agents_owner ON agents(owner_id);
```

- [ ] **Step 2: Create initial migration (down)**

Create `server/internal/storage/migrations/000001_initial_schema.down.sql`:

```sql
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS users;
```

- [ ] **Step 3: Create PostgreSQL connection and store**

Create `server/internal/storage/postgres.go`:

```go
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

// DB wraps a pgx pool and provides query methods.
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB creates a connection pool and runs migrations.
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

// Close shuts down the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}

// --- User operations ---

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	DisplayName  string
}

// CreateUser inserts a new user with a hashed password.
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

// GetUserByEmail retrieves a user by email.
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

// --- Agent operations ---

type Agent struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Name      string
	AgentType string
}

// CreateAgent inserts a new agent and returns it with the plaintext API key.
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

// GetAgentByAPIKey looks up an agent by the hash of its API key.
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

// ListAgentsByOwner returns all agents for a user.
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

// DeleteAgent removes an agent.
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

// --- Helpers ---

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
```

- [ ] **Step 4: Add dependencies**

```bash
cd /Users/lucasmeneses/mmoagens/server && go get github.com/jackc/pgx/v5 github.com/golang-migrate/migrate/v4 golang.org/x/crypto
```

- [ ] **Step 5: Verify build**

```bash
cd /Users/lucasmeneses/mmoagens && make build
```

- [ ] **Step 6: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/storage/ server/go.mod server/go.sum
git commit -m "feat: PostgreSQL storage with migrations, user and agent operations"
```

---

### Task 9: Redis storage — presence cache

**Files:**
- Create: `server/internal/storage/redis.go`

- [ ] **Step 1: Create Redis presence store**

Create `server/internal/storage/redis.go`:

```go
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore wraps a Redis client for presence operations.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore connects to Redis.
func NewRedisStore(addr string) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &RedisStore{client: client}, nil
}

// Close shuts down the Redis connection.
func (r *RedisStore) Close() error {
	return r.client.Close()
}

// PresenceData is the data stored in Redis for each online agent.
type PresenceData struct {
	Status    string    `json:"status"`
	Zone      string    `json:"zone"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SetPresence stores an agent's presence with a TTL.
func (r *RedisStore) SetPresence(ctx context.Context, agentID string, data PresenceData, ttl time.Duration) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, "agent:"+agentID+":presence", b, ttl).Err()
}

// GetPresence retrieves an agent's presence, or nil if expired/missing.
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

// DeletePresence removes an agent's presence.
func (r *RedisStore) DeletePresence(ctx context.Context, agentID string) error {
	return r.client.Del(ctx, "agent:"+agentID+":presence").Err()
}
```

- [ ] **Step 2: Add go-redis dependency**

```bash
cd /Users/lucasmeneses/mmoagens/server && go get github.com/redis/go-redis/v9
```

- [ ] **Step 3: Verify build**

```bash
cd /Users/lucasmeneses/mmoagens && make build
```

- [ ] **Step 4: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/storage/redis.go server/go.mod server/go.sum
git commit -m "feat: Redis presence cache with TTL-based agent presence"
```

---

## Chunk 3: Auth + REST API + WebSocket Hub

### Task 10: Auth — API key and JWT validation

**Files:**
- Create: `server/internal/net/auth.go`
- Create: `server/internal/net/auth_test.go`

- [ ] **Step 1: Write failing test for JWT**

Create `server/internal/net/auth_test.go`:

```go
package net_test

import (
	"testing"

	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
)

func TestJWT_RoundTrip(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	auth := wonet.NewAuth(secret)

	token, err := auth.GenerateJWT("user-123", "lucas@test.com")
	if err != nil {
		t.Fatalf("generate JWT: %v", err)
	}

	claims, err := auth.ValidateJWT(token)
	if err != nil {
		t.Fatalf("validate JWT: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Fatalf("expected user-123, got %s", claims.UserID)
	}
	if claims.Email != "lucas@test.com" {
		t.Fatalf("expected lucas@test.com, got %s", claims.Email)
	}
}

func TestJWT_InvalidToken(t *testing.T) {
	auth := wonet.NewAuth("test-secret-key-32-bytes-long!!")
	_, err := auth.ValidateJWT("invalid.token.here")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/net/ -v -run TestJWT
```

Expected: FAIL

- [ ] **Step 3: Implement Auth**

Create `server/internal/net/auth.go`:

```go
package net

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Auth handles JWT generation and validation.
type Auth struct {
	secret []byte
}

// NewAuth creates an Auth instance with the given secret.
func NewAuth(secret string) *Auth {
	return &Auth{secret: []byte(secret)}
}

// Claims holds JWT payload.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a signed JWT for a user.
func (a *Auth) GenerateJWT(userID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "woa-server",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secret)
}

// ValidateJWT parses and validates a JWT string.
func (a *Auth) ValidateJWT(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
```

- [ ] **Step 4: Add JWT dependency**

```bash
cd /Users/lucasmeneses/mmoagens/server && go get github.com/golang-jwt/jwt/v5
```

- [ ] **Step 5: Run tests**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/net/ -v -run TestJWT
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/net/auth.go server/internal/net/auth_test.go server/go.mod server/go.sum
git commit -m "feat: JWT auth with generation and validation"
```

---

### Task 11: REST API — register, login, create agent

**Files:**
- Create: `server/internal/net/rest.go`

- [ ] **Step 1: Create REST handlers**

Create `server/internal/net/rest.go`:

```go
package net

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// REST holds dependencies for REST handlers.
type REST struct {
	DB   *storage.DB
	Auth *Auth
}

// RegisterRoutes mounts all REST endpoints on the given mux.
func (r *REST) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", r.handleRegister)
	mux.HandleFunc("POST /auth/login", r.handleLogin)
	mux.HandleFunc("GET /api/agents", r.requireAuth(r.handleListAgents))
	mux.HandleFunc("POST /api/agents", r.requireAuth(r.handleCreateAgent))
	mux.HandleFunc("DELETE /api/agents/{id}", r.requireAuth(r.handleDeleteAgent))
	mux.HandleFunc("GET /api/stats", r.handleStats)
}

type contextKey string

const ctxKeyUserID contextKey = "user_id"

func (r *REST) handleRegister(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if body.Email == "" || body.Password == "" || body.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "email, password, and display_name required")
		return
	}

	user, err := r.DB.CreateUser(req.Context(), body.Email, body.Password, body.DisplayName)
	if err != nil {
		writeError(w, http.StatusConflict, "EMAIL_TAKEN", "email already registered")
		return
	}

	token, err := r.Auth.GenerateJWT(user.ID.String(), user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user_id": user.ID.String(),
		"token":   token,
	})
}

func (r *REST) handleLogin(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}

	user, err := r.DB.GetUserByEmail(req.Context(), body.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "AUTH_FAILED", "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "AUTH_FAILED", "invalid credentials")
		return
	}

	token, err := r.Auth.GenerateJWT(user.ID.String(), user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": user.ID.String(),
		"token":   token,
	})
}

func (r *REST) handleCreateAgent(w http.ResponseWriter, req *http.Request) {
	userID := req.Context().Value(ctxKeyUserID).(uuid.UUID)

	var body struct {
		Name      string `json:"name"`
		AgentType string `json:"agent_type"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if body.Name == "" || body.AgentType == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "name and agent_type required")
		return
	}

	agent, apiKey, err := r.DB.CreateAgent(req.Context(), userID, body.Name, body.AgentType)
	if err != nil {
		writeError(w, http.StatusConflict, "AGENT_EXISTS", "agent name already taken")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"agent_id":   agent.ID.String(),
		"name":       agent.Name,
		"agent_type": agent.AgentType,
		"api_key":    apiKey,
	})
}

func (r *REST) handleListAgents(w http.ResponseWriter, req *http.Request) {
	userID := req.Context().Value(ctxKeyUserID).(uuid.UUID)

	agents, err := r.DB.ListAgentsByOwner(req.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list agents")
		return
	}

	result := make([]map[string]any, len(agents))
	for i, a := range agents {
		result[i] = map[string]any{
			"id":         a.ID.String(),
			"name":       a.Name,
			"agent_type": a.AgentType,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (r *REST) handleDeleteAgent(w http.ResponseWriter, req *http.Request) {
	userID := req.Context().Value(ctxKeyUserID).(uuid.UUID)
	agentID, err := uuid.Parse(req.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid agent ID")
		return
	}

	if err := r.DB.DeleteAgent(req.Context(), agentID, userID); err != nil {
		writeError(w, http.StatusNotFound, "AGENT_NOT_FOUND", "agent not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *REST) handleStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "online"})
}

func (r *REST) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		authHeader := req.Header.Get("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			writeError(w, http.StatusUnauthorized, "AUTH_FAILED", "missing Authorization header")
			return
		}

		claims, err := r.Auth.ValidateJWT(authHeader[7:])
		if err != nil {
			writeError(w, http.StatusUnauthorized, "AUTH_FAILED", "invalid token")
			return
		}

		uid, err := uuid.Parse(claims.UserID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "AUTH_FAILED", "invalid user ID in token")
			return
		}

		ctx := context.WithValue(req.Context(), ctxKeyUserID, uid)
		next(w, req.WithContext(ctx))
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/lucasmeneses/mmoagens && make build
```

- [ ] **Step 3: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/net/rest.go
git commit -m "feat: REST API — register, login, create/list/delete agents"
```

---

### Task 12: WebSocket Hub — connection manager with auth handshake

**Files:**
- Create: `server/internal/net/hub.go`

- [ ] **Step 1: Create WebSocket Hub**

Create `server/internal/net/hub.go`:

```go
package net

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	"github.com/lucasmeneses/world-of-agents/server/internal/storage"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins for now
}

const (
	authTimeout = 5 * time.Second
	sendBufSize = 256
	writeWait   = 10 * time.Second
	pongWait    = 60 * time.Second
	pingPeriod  = (pongWait * 9) / 10
)

// Hub manages WebSocket connections and bridges them to the ECS world.
type Hub struct {
	mu    sync.RWMutex
	world *ecs.World
	bus   *engine.EventBus
	db    *storage.DB
	auth  *Auth

	// ActionQueue collects incoming messages to be processed next tick.
	ActionQueue chan IncomingAction
}

// IncomingAction is a message from a connected client, tagged with its entity ID.
type IncomingAction struct {
	EntityID ecs.EntityID
	Envelope Envelope
	Raw      []byte
}

// NewHub creates a WebSocket hub.
func NewHub(world *ecs.World, bus *engine.EventBus, db *storage.DB, auth *Auth) *Hub {
	return &Hub{
		world:       world,
		bus:         bus,
		db:          db,
		auth:        auth,
		ActionQueue: make(chan IncomingAction, 1024),
	}
}

// HandleWebSocket is the HTTP handler for WebSocket upgrade.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}

	go h.handleConnection(conn)
}

func (h *Hub) handleConnection(conn *websocket.Conn) {
	// Step 1: Send auth_required
	authReq := AuthRequiredMessage{Type: "auth_required"}
	if err := conn.WriteJSON(authReq); err != nil {
		conn.Close()
		return
	}

	// Step 2: Wait for auth message with timeout
	conn.SetReadDeadline(time.Now().Add(authTimeout))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		conn.WriteJSON(ErrorMessage{Type: "error", Code: "AUTH_TIMEOUT", Message: "auth timeout"})
		conn.Close()
		return
	}
	conn.SetReadDeadline(time.Time{}) // clear deadline

	// Step 3: Parse auth message
	var authMsg AuthMessage
	if err := json.Unmarshal(msg, &authMsg); err != nil || authMsg.Type != "auth" {
		conn.WriteJSON(ErrorMessage{Type: "error", Code: "AUTH_FAILED", Message: "expected auth message"})
		conn.Close()
		return
	}

	// Step 4: Validate credentials
	var agent *storage.Agent
	if authMsg.APIKey != "" {
		agent, err = h.db.GetAgentByAPIKey(context.Background(), authMsg.APIKey)
	} else if authMsg.Token != "" {
		// JWT auth for human viewers — not creating entity for now, just validate
		_, err = h.auth.ValidateJWT(authMsg.Token)
		if err != nil {
			conn.WriteJSON(ErrorMessage{Type: "error", Code: "AUTH_FAILED", Message: "invalid token"})
			conn.Close()
			return
		}
		// Human viewers don't get an ECS entity for now — just keep connection open
		// TODO: Phase 5 will handle human viewer connections
		conn.WriteJSON(WelcomeMessage{Type: "welcome", AgentID: "viewer", ProtocolVersion: 1})
		h.readPumpViewer(conn)
		return
	}

	if err != nil || agent == nil {
		conn.WriteJSON(ErrorMessage{Type: "error", Code: "AUTH_FAILED", Message: "invalid API key"})
		conn.Close()
		return
	}

	// Step 5: Create ECS entity for this agent
	entity := ecs.NewEntityWithID(agent.ID)
	entity.Add(&components.Identity{
		Name:      agent.Name,
		AgentType: agent.AgentType,
		OwnerID:   agent.OwnerID,
		AgentDBID: agent.ID,
	})
	entity.Add(&components.Presence{
		Status:        components.StatusOnline,
		Zone:          "",
		LastHeartbeat: time.Now(),
	})

	sendCh := make(chan []byte, sendBufSize)
	entity.Add(&components.Connection{
		Conn:      conn,
		SessionID: uuid.New().String(),
		Send:      sendCh,
	})

	h.world.AddEntity(entity)

	// Step 6: Send welcome
	welcome := WelcomeMessage{
		Type:            "welcome",
		AgentID:         agent.ID.String(),
		ProtocolVersion: 1,
	}
	conn.WriteJSON(welcome)

	// Step 7: Publish agent_online event
	h.bus.Publish(engine.Event{
		Type: "agent_online",
		Payload: map[string]any{
			"agent_id":   agent.ID.String(),
			"agent_name": agent.Name,
			"agent_type": agent.AgentType,
		},
	})

	slog.Info("agent connected", "name", agent.Name, "id", agent.ID.String())

	// Step 8: Start read/write pumps
	go h.writePump(conn, sendCh)
	h.readPump(conn, entity.ID)

	// Cleanup on disconnect
	h.world.RemoveEntity(entity.ID)
	h.bus.Publish(engine.Event{
		Type: "agent_offline",
		Payload: map[string]any{
			"agent_id": agent.ID.String(),
			"reason":   "disconnect",
		},
	})
	slog.Info("agent disconnected", "name", agent.Name)
}

func (h *Hub) readPump(conn *websocket.Conn, entityID ecs.EntityID) {
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		env, err := UnmarshalEnvelope(msg)
		if err != nil {
			continue
		}

		h.ActionQueue <- IncomingAction{
			EntityID: entityID,
			Envelope: env,
			Raw:      msg,
		}
	}
}

func (h *Hub) readPumpViewer(conn *websocket.Conn) {
	defer conn.Close()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writePump(conn *websocket.Conn, send chan []byte) {
	defer conn.Close()
	for msg := range send {
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

// Broadcast sends a message to all connected entities.
func (h *Hub) Broadcast(data []byte) {
	h.world.Each(func(e *ecs.Entity) {
		c := e.Get(components.ConnectionType)
		if c == nil {
			return
		}
		conn := c.(*components.Connection)
		select {
		case conn.Send <- data:
		default:
			// Buffer full, skip
		}
	})
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/lucasmeneses/mmoagens && make build
```

Fix any import issues.

- [ ] **Step 3: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/net/hub.go
git commit -m "feat: WebSocket hub with auth handshake and ECS entity creation"
```

---

### Task 12b: REST API tests

**Files:**
- Create: `server/internal/net/rest_test.go`

- [ ] **Step 1: Write REST handler tests**

Create `server/internal/net/rest_test.go`:

```go
package net_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleStats_ReturnsOnline(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"status": "online"})
	})

	req := httptest.NewRequest("GET", "/api/stats", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "online" {
		t.Fatalf("expected status=online, got %v", body["status"])
	}
}

func TestHandleRegister_RejectsBadJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "BAD_REQUEST", "message": "invalid JSON"},
			})
			return
		}
	})

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/net/ -v -run TestHandle
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/net/rest_test.go
git commit -m "test: REST API handler unit tests"
```

---

### Task 12c: Hub tests

**Files:**
- Create: `server/internal/net/hub_test.go`

- [ ] **Step 1: Write Hub unit tests**

Create `server/internal/net/hub_test.go`:

```go
package net_test

import (
	"testing"

	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
	"github.com/lucasmeneses/world-of-agents/server/internal/storage"
)

func TestNewHub_CreatesWithActionQueue(t *testing.T) {
	world := ecs.NewWorld()
	bus := engine.NewEventBus()
	hub := wonet.NewHub(world, bus, (*storage.DB)(nil), nil)

	if hub == nil {
		t.Fatal("expected hub to be created")
	}
	if hub.ActionQueue == nil {
		t.Fatal("expected action queue to be initialized")
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/net/ -v -run TestNewHub
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/net/hub_test.go
git commit -m "test: Hub creation unit test"
```

---

## Chunk 4: Systems + Main Wiring + Demo

### Task 13: PresenceSystem

**Files:**
- Create: `server/internal/systems/presence.go`
- Create: `server/internal/systems/presence_test.go`

- [ ] **Step 1: Write failing test**

Create `server/internal/systems/presence_test.go`:

```go
package systems_test

import (
	"testing"
	"time"

	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	"github.com/lucasmeneses/world-of-agents/server/internal/systems"
)

func TestPresenceSystem_MarksOfflineAfterTimeout(t *testing.T) {
	world := ecs.NewWorld()
	bus := engine.NewEventBus()
	sys := systems.NewPresenceSystem(bus, 10*time.Second)

	// Create an agent with a stale heartbeat (15 seconds ago)
	e := ecs.NewEntity()
	e.Add(&components.Identity{Name: "stale-agent"})
	e.Add(&components.Presence{
		Status:        components.StatusOnline,
		LastHeartbeat: time.Now().Add(-15 * time.Second),
	})
	world.AddEntity(e)

	sys.Update(world, 1)

	// Check the agent was marked offline
	p := e.Get(components.PresenceType).(*components.Presence)
	if p.Status != "offline" {
		t.Fatalf("expected offline, got %s", p.Status)
	}

	// Check event was published
	events := bus.Drain()
	if len(events) == 0 {
		t.Fatal("expected agent_offline event")
	}
	if events[0].Type != "agent_offline" {
		t.Fatalf("expected agent_offline event, got %s", events[0].Type)
	}
}

func TestPresenceSystem_KeepsOnlineWithRecentHeartbeat(t *testing.T) {
	world := ecs.NewWorld()
	bus := engine.NewEventBus()
	sys := systems.NewPresenceSystem(bus, 10*time.Second)

	e := ecs.NewEntity()
	e.Add(&components.Identity{Name: "fresh-agent"})
	e.Add(&components.Presence{
		Status:        components.StatusOnline,
		LastHeartbeat: time.Now(),
	})
	world.AddEntity(e)

	sys.Update(world, 1)

	p := e.Get(components.PresenceType).(*components.Presence)
	if p.Status != components.StatusOnline {
		t.Fatalf("expected online, got %s", p.Status)
	}

	events := bus.Drain()
	if len(events) != 0 {
		t.Fatalf("expected no events, got %d", len(events))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/systems/ -v
```

Expected: FAIL

- [ ] **Step 3: Implement PresenceSystem**

Create `server/internal/systems/presence.go`:

```go
package systems

import (
	"time"

	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
)

// PresenceSystem detects agents whose heartbeat has timed out and marks them offline.
type PresenceSystem struct {
	bus     *engine.EventBus
	timeout time.Duration
}

// NewPresenceSystem creates a presence system with the given heartbeat timeout.
func NewPresenceSystem(bus *engine.EventBus, timeout time.Duration) *PresenceSystem {
	return &PresenceSystem{bus: bus, timeout: timeout}
}

func (s *PresenceSystem) Name() string { return "presence" }

func (s *PresenceSystem) Update(world *ecs.World, tick uint64) {
	now := time.Now()

	world.Each(func(e *ecs.Entity) {
		if !e.HasAll(components.PresenceType, components.IdentityType) {
			return
		}

		p := e.Get(components.PresenceType).(*components.Presence)

		// Skip already-offline agents
		if p.Status == "offline" {
			return
		}

		// Check heartbeat timeout
		if now.Sub(p.LastHeartbeat) > s.timeout {
			p.Status = "offline"
			identity := e.Get(components.IdentityType).(*components.Identity)
			s.bus.Publish(engine.Event{
				Type: "agent_offline",
				Payload: map[string]any{
					"agent_id": e.ID.String(),
					"name":     identity.Name,
					"reason":   "timeout",
				},
			})
		}
	})
}
```

- [ ] **Step 4: Run tests**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/systems/ -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/systems/
git commit -m "feat: PresenceSystem detects heartbeat timeouts"
```

---

### Task 14: BroadcastSystem

**Files:**
- Create: `server/internal/systems/broadcast.go`

- [ ] **Step 1: Implement BroadcastSystem**

Create `server/internal/systems/broadcast.go`:

```go
package systems

import (
	"encoding/json"
	"log/slog"

	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	"github.com/lucasmeneses/world-of-agents/server/internal/net"
)

// BroadcastSystem drains the event bus and pushes events to all connected clients.
type BroadcastSystem struct {
	bus *engine.EventBus
}

// NewBroadcastSystem creates a broadcast system.
func NewBroadcastSystem(bus *engine.EventBus) *BroadcastSystem {
	return &BroadcastSystem{bus: bus}
}

func (s *BroadcastSystem) Name() string { return "broadcast" }

func (s *BroadcastSystem) Update(world *ecs.World, tick uint64) {
	events := s.bus.Drain()
	if len(events) == 0 {
		return
	}

	// Wrap events in a tick message
	tickMsg := net.TickMessage{
		Type:   "tick",
		Number: tick,
		Events: make([]any, len(events)),
	}
	for i, e := range events {
		tickMsg.Events[i] = e
	}

	data, err := json.Marshal(tickMsg)
	if err != nil {
		slog.Error("failed to marshal tick message", "error", err)
		return
	}

	// Send to all connected entities
	world.Each(func(e *ecs.Entity) {
		c := e.Get(components.ConnectionType)
		if c == nil {
			return
		}
		conn := c.(*components.Connection)
		select {
		case conn.Send <- data:
		default:
			// Buffer full, drop message for this client
		}
	})
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/lucasmeneses/mmoagens && make build
```

- [ ] **Step 3: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/systems/broadcast.go
git commit -m "feat: BroadcastSystem pushes tick events to all connected clients"
```

---

### Task 14b: BroadcastSystem tests

**Files:**
- Create: `server/internal/systems/broadcast_test.go`

- [ ] **Step 1: Write BroadcastSystem test**

Create `server/internal/systems/broadcast_test.go`:

```go
package systems_test

import (
	"testing"

	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	"github.com/lucasmeneses/world-of-agents/server/internal/systems"
)

func TestBroadcastSystem_NoEventsNoBroadcast(t *testing.T) {
	world := ecs.NewWorld()
	bus := engine.NewEventBus()
	sys := systems.NewBroadcastSystem(bus)

	// No events published, Update should be a no-op
	sys.Update(world, 1)
}

func TestBroadcastSystem_Name(t *testing.T) {
	bus := engine.NewEventBus()
	sys := systems.NewBroadcastSystem(bus)
	if sys.Name() != "broadcast" {
		t.Fatalf("expected 'broadcast', got %q", sys.Name())
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd /Users/lucasmeneses/mmoagens/server && go test ./internal/systems/ -v -run TestBroadcast
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/systems/broadcast_test.go
git commit -m "test: BroadcastSystem unit tests"
```

---

### Task 15: Action Processor — handle heartbeat and status messages

**Files:**
- Create: `server/internal/systems/actions.go`

- [ ] **Step 1: Create ActionProcessor**

This system drains the Hub's ActionQueue and applies actions to the ECS world.

Create `server/internal/systems/actions.go`:

```go
package systems

import (
	"encoding/json"
	"time"

	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
)

// ActionProcessor drains the action queue and applies messages to entities.
type ActionProcessor struct {
	bus         *engine.EventBus
	actionQueue <-chan wonet.IncomingAction
}

// NewActionProcessor creates an action processor.
func NewActionProcessor(bus *engine.EventBus, queue <-chan wonet.IncomingAction) *ActionProcessor {
	return &ActionProcessor{bus: bus, actionQueue: queue}
}

func (s *ActionProcessor) Name() string { return "actions" }

func (s *ActionProcessor) Update(world *ecs.World, tick uint64) {
	// Drain all pending actions (non-blocking)
	for {
		select {
		case action := <-s.actionQueue:
			s.processAction(world, action)
		default:
			return
		}
	}
}

func (s *ActionProcessor) processAction(world *ecs.World, action wonet.IncomingAction) {
	entity := world.Entity(action.EntityID)
	if entity == nil {
		return
	}

	switch action.Envelope.Type {
	case "heartbeat":
		s.handleHeartbeat(entity, action.Raw)
	case "set_status":
		s.handleSetStatus(entity, action.Raw)
	case "set_zone":
		s.handleSetZone(entity, action.Raw)
	}
}

func (s *ActionProcessor) handleHeartbeat(entity *ecs.Entity, raw []byte) {
	var msg wonet.HeartbeatMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	p := entity.Get(components.PresenceType)
	if p == nil {
		return
	}
	presence := p.(*components.Presence)
	presence.LastHeartbeat = time.Now()

	if msg.Status != "" {
		presence.Status = msg.Status
	}
	if msg.Zone != "" {
		presence.Zone = msg.Zone
	}
}

func (s *ActionProcessor) handleSetStatus(entity *ecs.Entity, raw []byte) {
	var msg wonet.SetStatusMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	p := entity.Get(components.PresenceType)
	if p == nil {
		return
	}
	presence := p.(*components.Presence)
	oldStatus := presence.Status
	presence.Status = msg.Status

	if oldStatus != msg.Status {
		identity := entity.Get(components.IdentityType).(*components.Identity)
		s.bus.Publish(engine.Event{
			Type: "agent_status",
			Payload: map[string]any{
				"agent_id": entity.ID.String(),
				"name":     identity.Name,
				"status":   msg.Status,
				"zone":     presence.Zone,
			},
		})
	}
}

func (s *ActionProcessor) handleSetZone(entity *ecs.Entity, raw []byte) {
	var msg wonet.SetZoneMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	p := entity.Get(components.PresenceType)
	if p == nil {
		return
	}
	presence := p.(*components.Presence)
	presence.Zone = msg.Zone
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Users/lucasmeneses/mmoagens && make build
```

- [ ] **Step 3: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/internal/systems/actions.go
git commit -m "feat: ActionProcessor handles heartbeat, set_status, set_zone"
```

---

### Task 16: Wire everything in main.go

**Files:**
- Modify: `server/cmd/server/main.go`

- [ ] **Step 1: Wire all components together**

Replace `server/cmd/server/main.go` with:

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
	"github.com/lucasmeneses/world-of-agents/server/internal/storage"
	"github.com/lucasmeneses/world-of-agents/server/internal/systems"
)

func main() {
	slog.Info("World of Agents server starting...")

	// Config from env
	dbURL := envOr("DATABASE_URL", "postgres://woa:woa_dev@localhost:5432/woa?sslmode=disable")
	redisAddr := envOr("REDIS_ADDR", "localhost:6379")
	jwtSecret := envOr("JWT_SECRET", "woa-dev-secret-change-in-production!")
	listenAddr := envOr("LISTEN_ADDR", ":8080")
	tickRate := 200 * time.Millisecond // 5 Hz

	// Storage
	ctx := context.Background()
	db, err := storage.NewDB(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redisStore, err := storage.NewRedisStore(redisAddr)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisStore.Close()
	_ = redisStore // will be wired into PresenceSystem in Phase 2

	// ECS World
	world := ecs.NewWorld()
	bus := engine.NewEventBus()

	// Auth
	auth := wonet.NewAuth(jwtSecret)

	// WebSocket Hub
	hub := wonet.NewHub(world, bus, db, auth)

	// Systems (order matters — actions first, broadcast last)
	actionProcessor := systems.NewActionProcessor(bus, hub.ActionQueue)
	presenceSystem := systems.NewPresenceSystem(bus, 15*time.Second) // 3 missed heartbeats
	broadcastSystem := systems.NewBroadcastSystem(bus)

	world.AddSystem(actionProcessor)
	world.AddSystem(presenceSystem)
	world.AddSystem(broadcastSystem)

	// Tick Engine
	eng := engine.NewTickEngine(world, bus, tickRate)
	go eng.Start()

	// HTTP Server
	mux := http.NewServeMux()

	rest := &wonet.REST{DB: db, Auth: auth}
	rest.RegisterRoutes(mux)

	mux.HandleFunc("GET /ws", hub.HandleWebSocket)

	server := &http.Server{Addr: listenAddr, Handler: mux}

	go func() {
		slog.Info("listening", "addr", listenAddr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	eng.Stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
	slog.Info("goodbye")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 2: Build and verify**

```bash
cd /Users/lucasmeneses/mmoagens && make build
```

Fix any remaining compilation issues.

- [ ] **Step 3: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add server/cmd/server/main.go
git commit -m "feat: wire ECS world, tick engine, WebSocket hub, REST API in main.go"
```

---

### Task 17: Integration test — two agents connect and see each other

**Files:**
- No new files — manual test with `wscat`

- [ ] **Step 1: Start infrastructure**

```bash
cd /Users/lucasmeneses/mmoagens && make infra
```

- [ ] **Step 2: Start server**

```bash
cd /Users/lucasmeneses/mmoagens && make dev
```

Expected: "World of Agents server starting..." and "listening :8080"

- [ ] **Step 3: Register a user**

```bash
curl -s http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"lucas@test.com","password":"test123","display_name":"Lucas"}' | jq .
```

Expected: `{"user_id": "...", "token": "..."}`

- [ ] **Step 4: Create two agents**

```bash
TOKEN="<jwt from step 3>"

curl -s http://localhost:8080/api/agents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"claude-mac","agent_type":"claude"}' | jq .

curl -s http://localhost:8080/api/agents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"codex-vps","agent_type":"codex"}' | jq .
```

Save both `api_key` values.

- [ ] **Step 5: Connect Agent 1 via wscat**

```bash
npx wscat -c ws://localhost:8080/ws
```

When you see `{"type":"auth_required"}`, type:
```json
{"type":"auth","api_key":"<api_key_1>"}
```

Expected: `{"type":"welcome","agent_id":"...","protocol_version":1}`

- [ ] **Step 6: Connect Agent 2 in another terminal**

```bash
npx wscat -c ws://localhost:8080/ws
```

Authenticate with the second API key.

Expected: Agent 1 receives a tick message containing an `agent_online` event for Agent 2.

- [ ] **Step 7: Send heartbeat from Agent 1**

In Agent 1's terminal:
```json
{"type":"heartbeat","status":"working","zone":"glifo/"}
```

Expected: Agent 2 receives an `agent_status` event.

- [ ] **Step 8: Disconnect Agent 2**

Close Agent 2's terminal (Ctrl+C).

Expected: Agent 1 receives an `agent_offline` event for Agent 2 (either immediately on disconnect or after heartbeat timeout).

- [ ] **Step 9: Commit any fixes from integration testing**

```bash
cd /Users/lucasmeneses/mmoagens
git add -A
git commit -m "fix: integration test fixes from manual wscat testing"
```

---

### Task 18: Create CLAUDE.md project guide

**Files:**
- Create: `CLAUDE.md`

- [ ] **Step 1: Create project documentation**

Create `CLAUDE.md` at repo root:

```markdown
# World of Agents

MMORPG-inspired platform for AI agent coordination.

## Quick Start

```bash
make infra       # Start PostgreSQL + Redis
make dev          # Run server (port 8080)
make test         # Run all tests
make build        # Build binary to server/bin/woa-server
```

## Architecture

- **ECS** (Entity-Component-System) with tick-based game loop at 5 Hz
- **WebSocket** for real-time bidirectional communication
- **PostgreSQL** for persistence, **Redis** for presence cache

## Project Structure

- `server/` — Go game server (woa-server)
  - `cmd/server/main.go` — entry point
  - `internal/ecs/` — ECS core (World, Entity, Component, System)
  - `internal/components/` — data components (Identity, Presence, Connection)
  - `internal/systems/` — behavior systems (Presence, Broadcast, Actions)
  - `internal/engine/` — tick loop and event bus
  - `internal/net/` — WebSocket hub, protocol, auth, REST API
  - `internal/storage/` — PostgreSQL and Redis

## Environment Variables

- `DATABASE_URL` — PostgreSQL connection string (default: local dev)
- `REDIS_ADDR` — Redis address (default: localhost:6379)
- `JWT_SECRET` — Secret for JWT signing (change in production!)
- `LISTEN_ADDR` — HTTP listen address (default: :8080)

## Conventions

- Go stdlib `net/http` with Go 1.22+ pattern matching (no framework)
- `pgx/v5` for PostgreSQL
- `gorilla/websocket` for WebSocket
- ECS: Components are pure data, Systems contain behavior
- All state changes happen within ticks
- JSON over WebSocket (no protobuf for now)
```

- [ ] **Step 2: Commit**

```bash
cd /Users/lucasmeneses/mmoagens
git add CLAUDE.md
git commit -m "docs: add CLAUDE.md project guide"
```
