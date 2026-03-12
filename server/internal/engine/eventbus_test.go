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
	if len(events) != 2 { t.Fatalf("expected 2 events, got %d", len(events)) }
	if events[0].Type != "agent_online" { t.Fatalf("expected agent_online, got %s", events[0].Type) }
	events = bus.Drain()
	if len(events) != 0 { t.Fatalf("expected 0 events after drain, got %d", len(events)) }
}
