package engine

import (
	"sync"

	"github.com/google/uuid"
)

type EventScope struct {
	Type     string      // "global", "guild", "direct"
	GuildID  *uuid.UUID  // for guild-scoped events
	AgentIDs []uuid.UUID // for direct-scoped events
}

func GlobalScope() EventScope {
	return EventScope{Type: "global"}
}

func GuildScope(guildID uuid.UUID) EventScope {
	return EventScope{Type: "guild", GuildID: &guildID}
}

func DirectScope(agentIDs ...uuid.UUID) EventScope {
	return EventScope{Type: "direct", AgentIDs: agentIDs}
}

type Event struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
	Scope   EventScope     `json:"-"` // not serialized — used by BroadcastSystem only
}

type EventBus struct {
	mu     sync.Mutex
	events []Event
}

func NewEventBus() *EventBus { return &EventBus{} }

func (b *EventBus) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, e)
}

func (b *EventBus) Drain() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	events := b.events
	b.events = nil
	return events
}
