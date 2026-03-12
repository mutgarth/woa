package engine

import "sync"

type Event struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
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
