package main

import (
	"testing"

	"github.com/lucasmeneses/world-of-agents/pkg/woasdk"
)

func TestEventBuf_PushAndDrain(t *testing.T) {
	buf := newEventBuf(5)
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a1"})
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a2"})
	events := buf.Drain()
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events := buf.Drain(); len(events) != 0 {
		t.Fatalf("got %d after drain, want 0", len(events))
	}
}

func TestEventBuf_Overflow(t *testing.T) {
	buf := newEventBuf(3)
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a1"})
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a2"})
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a3"})
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a4"}) // drops a1
	events := buf.Drain()
	if len(events) != 3 {
		t.Fatalf("got %d, want 3", len(events))
	}
	if events[0].(*woasdk.AgentOnlineEvent).AgentID != "a2" {
		t.Fatalf("oldest should be a2, got %s", events[0].(*woasdk.AgentOnlineEvent).AgentID)
	}
}

func TestEventBuf_Recent(t *testing.T) {
	buf := newEventBuf(100)
	for i := 0; i < 30; i++ {
		buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a"})
	}
	recent := buf.Recent(20)
	if len(recent) != 20 {
		t.Fatalf("got %d recent, want 20", len(recent))
	}
	// Recent does NOT drain
	if buf.Len() != 30 {
		t.Fatalf("buf should still have 30, got %d", buf.Len())
	}
}
