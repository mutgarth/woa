package woasdk_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lucasmeneses/world-of-agents/pkg/woasdk"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func mockServer(t *testing.T, tickEvents []map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		conn.WriteJSON(map[string]string{"type": "auth_required"})

		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var auth map[string]string
		json.Unmarshal(raw, &auth)
		if auth["api_key"] != "test-key" {
			conn.WriteJSON(map[string]any{"type": "error", "code": "AUTH_FAILED", "message": "bad key"})
			return
		}

		conn.WriteJSON(map[string]any{
			"type": "welcome", "agent_id": "agent-001",
			"server_tick": 0, "protocol_version": 1,
		})

		if len(tickEvents) > 0 {
			conn.WriteJSON(map[string]any{"type": "tick", "number": 1, "events": tickEvents})
		}

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
}

func wsURL(s *httptest.Server) string {
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

func TestConnect_Success(t *testing.T) {
	srv := mockServer(t, nil)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := woasdk.Connect(ctx, woasdk.Config{ServerURL: wsURL(srv), APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if client.AgentID() != "agent-001" {
		t.Fatalf("got agent_id %q, want agent-001", client.AgentID())
	}
}

func TestConnect_BadKey(t *testing.T) {
	srv := mockServer(t, nil)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := woasdk.Connect(ctx, woasdk.Config{ServerURL: wsURL(srv), APIKey: "wrong-key"})
	if err == nil {
		t.Fatal("expected error for bad key")
	}
	if !strings.Contains(err.Error(), "AUTH_FAILED") {
		t.Fatalf("error should contain AUTH_FAILED, got: %v", err)
	}
}

func TestEvents_ReceiveMultipleTypes(t *testing.T) {
	events := []map[string]any{
		{"type": "guild_created", "payload": map[string]any{
			"guild": map[string]any{"id": "g1", "name": "demo", "description": "", "visibility": "public"},
		}},
		{"type": "message", "payload": map[string]any{
			"id": "m1", "channel": "guild", "from": map[string]any{"agent_id": "a1", "name": "Bot"},
			"content": "hello", "created_at": "2026-01-01T00:00:00Z",
		}},
	}
	srv := mockServer(t, events)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := woasdk.Connect(ctx, woasdk.Config{ServerURL: wsURL(srv), APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	for i := 0; i < 2; i++ {
		select {
		case evt := <-client.Events():
			switch i {
			case 0:
				if _, ok := evt.(*woasdk.GuildCreatedEvent); !ok {
					t.Fatalf("event 0: expected *GuildCreatedEvent, got %T", evt)
				}
			case 1:
				me, ok := evt.(*woasdk.MessageEvent)
				if !ok {
					t.Fatalf("event 1: expected *MessageEvent, got %T", evt)
				}
				if me.From.AgentID != "a1" {
					t.Fatalf("from.agent_id: got %q, want a1", me.From.AgentID)
				}
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for event %d", i)
		}
	}
}

func TestActions_VerifyWireFormat(t *testing.T) {
	received := make(chan map[string]any, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		defer conn.Close()
		conn.WriteJSON(map[string]string{"type": "auth_required"})
		conn.ReadMessage()
		conn.WriteJSON(map[string]any{"type": "welcome", "agent_id": "a1", "server_tick": 0, "protocol_version": 1})
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var m map[string]any
			json.Unmarshal(raw, &m)
			if m["type"] != "heartbeat" {
				received <- m
			}
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := woasdk.Connect(ctx, woasdk.Config{ServerURL: wsURL(srv), APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	client.Guild.Create("g1", "desc", "public")
	client.Task.Post("Fix bug", "broken", "high")
	client.Chat.SendDirect("a2", "hey")
	client.Presence.SetStatus("busy")

	for i := 0; i < 4; i++ {
		select {
		case msg := <-received:
			switch i {
			case 0:
				if msg["type"] != "guild_create" {
					t.Fatalf("msg 0: got type %q, want guild_create", msg["type"])
				}
			case 1:
				if msg["type"] != "task_post" {
					t.Fatalf("msg 1: got type %q, want task_post", msg["type"])
				}
			case 2:
				if msg["type"] != "message" {
					t.Fatalf("msg 2: got type %q, want message", msg["type"])
				}
			case 3:
				if msg["type"] != "set_status" {
					t.Fatalf("msg 3: got type %q, want set_status", msg["type"])
				}
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for message %d", i)
		}
	}
}
