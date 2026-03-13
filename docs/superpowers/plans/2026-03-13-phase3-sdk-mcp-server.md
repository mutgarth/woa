# Phase 3: Go SDK + MCP Server — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go SDK client library and MCP server so AI agents (Claude Code, OpenClaw) can connect to the WoA world via WebSocket and interact through standard MCP tool calls.

**Architecture:** Standalone Go module at repo root. SDK (`pkg/woasdk`) wraps the WebSocket protocol with typed methods and an event channel. MCP server (`cmd/woa-mcp`) wraps the SDK, exposing WoA actions as MCP tools via stdio transport. The SDK uses a `sender` interface internally for testability.

**Tech Stack:** Go 1.25, gorilla/websocket v1.5.3, mark3labs/mcp-go (MCP framework), standard `testing` package

**Spec:** `docs/superpowers/specs/2026-03-13-phase3-sdk-mcp-server-design.md`

---

## File Structure

```
(repo root)
├── go.mod                      # New module: github.com/lucasmeneses/world-of-agents
├── go.sum
├── pkg/woasdk/
│   ├── events.go               # Event interface + all typed event structs
│   ├── protocol.go             # JSON envelope parsing, message marshaling
│   ├── protocol_test.go        # Protocol encoding/decoding tests
│   ├── client.go               # Connect(), Client, Close(), Events(), write loop
│   ├── actions.go              # GuildActions, TaskActions, ChatActions, PresenceActions
│   └── client_test.go          # Integration tests with mock WebSocket server
└── cmd/woa-mcp/
    ├── main.go                 # Entry point, env config, connect SDK, serve stdio
    ├── woaclient.go            # WoAClient interface (for testability)
    ├── eventbuf.go             # Ring buffer for event accumulation
    ├── eventbuf_test.go        # Ring buffer tests
    ├── handlers.go             # Extracted handler logic (testable without MCP framework)
    ├── handlers_test.go        # Handler unit tests with mock WoAClient
    ├── tools.go                # MCP tool registration (thin wrappers around handlers)
    └── tools_test.go           # Verify tool registration + response formatting
```

---

## Chunk 1: Go SDK

### Task 1: Module setup + Event types

**Files:**
- Create: `go.mod`
- Create: `pkg/woasdk/events.go`

- [ ] **Step 1: Create Go module at repo root**

```bash
cd /Users/lucasmeneses/mmoagens
go mod init github.com/lucasmeneses/world-of-agents
go get github.com/gorilla/websocket@v1.5.3
```

- [ ] **Step 2: Create `pkg/woasdk/events.go`**

All typed event structs implementing the `Event` interface. See spec section "Typed Events" and "Wire Protocol Reference > Event Types" for field names and JSON tags.

Key design notes:
- `MessageEvent.From` is `MessageSender{AgentID, Name}` — the server uses `"agent_id"` not `"id"` for chat messages (different from `AgentInfo` which uses `"id"`)
- `AgentOnlineEvent` has flat fields (`agent_id`, `agent_name`, `agent_type`) — NOT a nested `AgentInfo`
- `TickEvent.Events` has `json:"-"` because tick parsing happens in protocol.go, not via standard unmarshaling
- `DisconnectEvent` is SDK-synthetic (not from server), emitted when WebSocket closes

```go
package woasdk

// Event is the interface all WoA events implement.
type Event interface {
	EventType() string
}

// --- Connection events ---

type WelcomeEvent struct {
	AgentID         string `json:"agent_id"`
	ServerTick      uint64 `json:"server_tick"`
	ProtocolVersion int    `json:"protocol_version"`
}
func (e WelcomeEvent) EventType() string { return "welcome" }

type ErrorEvent struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
func (e ErrorEvent) EventType() string { return "error" }

// DisconnectEvent is synthetic — emitted by SDK when WebSocket closes.
type DisconnectEvent struct{ Err error }
func (e DisconnectEvent) EventType() string { return "disconnect" }

// --- Chat events ---

// MessageSender identifies chat sender. Wire uses "agent_id" not "id".
type MessageSender struct {
	AgentID string `json:"agent_id"`
	Name    string `json:"name"`
}

type MessageEvent struct {
	ID        string        `json:"id"`
	Channel   string        `json:"channel"`
	From      MessageSender `json:"from"`
	To        string        `json:"to,omitempty"`
	Content   string        `json:"content"`
	CreatedAt string        `json:"created_at"`
}
func (e MessageEvent) EventType() string { return "message" }

// --- Guild events ---

type AgentInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type GuildInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type GuildCreatedEvent struct{ Guild GuildInfo `json:"guild"` }
func (e GuildCreatedEvent) EventType() string { return "guild_created" }

type MemberJoinedEvent struct {
	GuildID string    `json:"guild_id"`
	Agent   AgentInfo `json:"agent"`
}
func (e MemberJoinedEvent) EventType() string { return "member_joined" }

type MemberLeftEvent struct {
	GuildID string `json:"guild_id"`
	AgentID string `json:"agent_id"`
}
func (e MemberLeftEvent) EventType() string { return "member_left" }

// --- Task events (one type per action, matching server) ---

type TaskInfo struct {
	ID          string `json:"id"`
	GuildID     string `json:"guild_id"`
	PostedBy    string `json:"posted_by"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
}

type TaskCreatedEvent struct{ Task TaskInfo `json:"task"` }
func (e TaskCreatedEvent) EventType() string { return "task_created" }

type TaskClaimedEvent struct {
	TaskID string `json:"task_id"`; AgentID string `json:"agent_id"`; Status string `json:"status"`
}
func (e TaskClaimedEvent) EventType() string { return "task_claimed" }

type TaskCompletedEvent struct {
	TaskID string `json:"task_id"`; AgentID string `json:"agent_id"`; Status string `json:"status"`; Result string `json:"result"`
}
func (e TaskCompletedEvent) EventType() string { return "task_completed" }

type TaskAbandonedEvent struct {
	TaskID string `json:"task_id"`; AgentID string `json:"agent_id"`; Status string `json:"status"`
}
func (e TaskAbandonedEvent) EventType() string { return "task_abandoned" }

type TaskFailedEvent struct {
	TaskID string `json:"task_id"`; AgentID string `json:"agent_id"`; Status string `json:"status"`
}
func (e TaskFailedEvent) EventType() string { return "task_failed" }

type TaskCancelledEvent struct {
	TaskID string `json:"task_id"`; AgentID string `json:"agent_id"`; Status string `json:"status"`
}
func (e TaskCancelledEvent) EventType() string { return "task_cancelled" }

// --- Presence events (flat fields, NOT nested AgentInfo) ---

type AgentOnlineEvent struct {
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
	AgentType string `json:"agent_type"`
}
func (e AgentOnlineEvent) EventType() string { return "agent_online" }

type AgentOfflineEvent struct {
	AgentID string `json:"agent_id"`
	Name    string `json:"name,omitempty"`
	Reason  string `json:"reason"`
}
func (e AgentOfflineEvent) EventType() string { return "agent_offline" }

type AgentStatusEvent struct {
	AgentID string `json:"agent_id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Zone    string `json:"zone"`
}
func (e AgentStatusEvent) EventType() string { return "agent_status" }

// --- Tick wrapper ---

// TickEvent wraps events from one server tick. Events field is parsed
// by protocol.go (not standard JSON unmarshal), hence json:"-".
type TickEvent struct {
	Number uint64  `json:"number"`
	Events []Event `json:"-"`
}
func (e TickEvent) EventType() string { return "tick" }
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/lucasmeneses/mmoagens && go build ./pkg/woasdk/`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum pkg/woasdk/events.go
git commit -m "feat(sdk): add Go module and event type definitions"
```

---

### Task 2: Protocol encoding/decoding

**Files:**
- Create: `pkg/woasdk/protocol.go`
- Create: `pkg/woasdk/protocol_test.go`

- [ ] **Step 1: Write `pkg/woasdk/protocol_test.go`**

Tests cover: parsing auth_required, welcome, error, and tick messages; parsing all event types from tick payload; marshaling all client-to-server message types.

```go
package woasdk

import (
	"encoding/json"
	"testing"
)

func TestParseServerMessage_AuthRequired(t *testing.T) {
	raw := json.RawMessage(`{"type":"auth_required"}`)
	msg, err := parseServerMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Type != "auth_required" {
		t.Fatalf("got type %q, want auth_required", msg.Type)
	}
}

func TestParseServerMessage_Welcome(t *testing.T) {
	raw := json.RawMessage(`{"type":"welcome","agent_id":"abc-123","server_tick":42,"protocol_version":1}`)
	msg, err := parseServerMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	evt := msg.Event.(*WelcomeEvent)
	if evt.AgentID != "abc-123" || evt.ServerTick != 42 {
		t.Fatalf("unexpected welcome: %+v", evt)
	}
}

func TestParseServerMessage_Error(t *testing.T) {
	raw := json.RawMessage(`{"type":"error","code":"AUTH_FAILED","message":"bad key"}`)
	msg, err := parseServerMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	evt := msg.Event.(*ErrorEvent)
	if evt.Code != "AUTH_FAILED" {
		t.Fatalf("got code %q, want AUTH_FAILED", evt.Code)
	}
}

func TestParseTickEvents_AllTypes(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"tick","number":5,"events":[
			{"type":"guild_created","payload":{"guild":{"id":"g1","name":"demo","description":"a guild","visibility":"public"}}},
			{"type":"member_joined","payload":{"guild_id":"g1","agent":{"id":"a1","name":"Bot","type":"explorer"}}},
			{"type":"member_left","payload":{"guild_id":"g1","agent_id":"a1"}},
			{"type":"task_created","payload":{"task":{"id":"t1","guild_id":"g1","posted_by":"a1","title":"Fix","description":"","priority":"high","status":"open"}}},
			{"type":"task_claimed","payload":{"task_id":"t1","agent_id":"a1","status":"claimed"}},
			{"type":"task_completed","payload":{"task_id":"t1","agent_id":"a1","status":"completed","result":"done"}},
			{"type":"task_abandoned","payload":{"task_id":"t1","agent_id":"a1","status":"open"}},
			{"type":"task_failed","payload":{"task_id":"t1","agent_id":"a1","status":"failed"}},
			{"type":"task_cancelled","payload":{"task_id":"t1","agent_id":"a1","status":"cancelled"}},
			{"type":"message","payload":{"id":"m1","channel":"guild","from":{"agent_id":"a1","name":"Bot"},"content":"hello","created_at":"2026-01-01T00:00:00Z"}},
			{"type":"agent_online","payload":{"agent_id":"a1","agent_name":"Bot","agent_type":"explorer"}},
			{"type":"agent_offline","payload":{"agent_id":"a1","name":"Bot","reason":"timeout"}},
			{"type":"agent_status","payload":{"agent_id":"a1","name":"Bot","status":"busy","zone":"mining"}}
		]
	}`)
	msg, err := parseServerMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	tick := msg.Event.(*TickEvent)
	if tick.Number != 5 {
		t.Fatalf("got tick %d, want 5", tick.Number)
	}
	if len(tick.Events) != 13 {
		t.Fatalf("got %d events, want 13", len(tick.Events))
	}

	// Spot-check key types
	if gc, ok := tick.Events[0].(*GuildCreatedEvent); !ok || gc.Guild.Name != "demo" {
		t.Fatalf("event 0: expected GuildCreatedEvent with name=demo, got %T", tick.Events[0])
	}
	if me, ok := tick.Events[9].(*MessageEvent); !ok || me.From.AgentID != "a1" {
		t.Fatalf("event 9: expected MessageEvent with from.agent_id=a1, got %T", tick.Events[9])
	}
	if ao, ok := tick.Events[10].(*AgentOnlineEvent); !ok || ao.AgentName != "Bot" {
		t.Fatalf("event 10: expected AgentOnlineEvent, got %T", tick.Events[10])
	}
}

// --- Marshal tests ---

func TestMarshalAuth(t *testing.T) {
	assertJSON(t, marshalAuth("key1"), `{"api_key":"key1","type":"auth"}`)
}

func TestMarshalHeartbeat(t *testing.T) {
	assertJSON(t, marshalHeartbeat(), `{"type":"heartbeat"}`)
}

func TestMarshalSetStatus(t *testing.T) {
	assertJSONField(t, marshalSetStatus("busy"), "status", "busy")
}

func TestMarshalSetZone(t *testing.T) {
	assertJSONField(t, marshalSetZone("mine"), "zone", "mine")
}

func TestMarshalGuildCreate(t *testing.T) {
	data := marshalGuildCreate("g1", "desc", "public")
	assertJSONField(t, data, "type", "guild_create")
	assertPayloadField(t, data, "name", "g1")
}

func TestMarshalGuildJoin(t *testing.T) {
	data := marshalGuildJoin("g1")
	assertJSONField(t, data, "type", "guild_join")
	assertPayloadField(t, data, "guild_name", "g1")
}

func TestMarshalGuildLeave(t *testing.T) {
	assertJSON(t, marshalGuildLeave(), `{"type":"guild_leave"}`)
}

func TestMarshalTaskPost(t *testing.T) {
	data := marshalTaskPost("Fix", "broken", "high")
	assertJSONField(t, data, "type", "task_post")
	assertPayloadField(t, data, "title", "Fix")
	assertPayloadField(t, data, "priority", "high")
}

func TestMarshalTaskAction_Claim(t *testing.T) {
	data := marshalTaskAction("task_claim", "t1", "")
	assertJSONField(t, data, "type", "task_claim")
	assertPayloadField(t, data, "task_id", "t1")
}

func TestMarshalTaskAction_Complete(t *testing.T) {
	data := marshalTaskAction("task_complete", "t1", "done")
	assertPayloadField(t, data, "result", "done")
}

func TestMarshalChatGuild(t *testing.T) {
	data := marshalChatGuild("hi")
	assertJSONField(t, data, "type", "message")
	assertPayloadField(t, data, "channel", "guild")
	assertPayloadField(t, data, "content", "hi")
}

func TestMarshalChatDirect(t *testing.T) {
	data := marshalChatDirect("a2", "hey")
	assertPayloadField(t, data, "channel", "direct")
	assertPayloadField(t, data, "to", "a2")
}

// --- Helpers ---

func assertJSON(t *testing.T, data []byte, expected string) {
	t.Helper()
	var got, want any
	json.Unmarshal(data, &got)
	json.Unmarshal([]byte(expected), &want)
	g, _ := json.Marshal(got)
	w, _ := json.Marshal(want)
	if string(g) != string(w) {
		t.Fatalf("got %s, want %s", g, w)
	}
}

func assertJSONField(t *testing.T, data []byte, key, expected string) {
	t.Helper()
	var m map[string]any
	json.Unmarshal(data, &m)
	if v, _ := m[key].(string); v != expected {
		t.Fatalf("field %q: got %q, want %q", key, v, expected)
	}
}

func assertPayloadField(t *testing.T, data []byte, key, expected string) {
	t.Helper()
	var m map[string]any
	json.Unmarshal(data, &m)
	payload, ok := m["payload"].(map[string]any)
	if !ok {
		t.Fatalf("no payload in %s", data)
	}
	if v, _ := payload[key].(string); v != expected {
		t.Fatalf("payload.%s: got %q, want %q", key, v, expected)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/lucasmeneses/mmoagens && go test ./pkg/woasdk/ -v -run TestParse`
Expected: FAIL — `parseServerMessage` not defined

- [ ] **Step 3: Implement `pkg/woasdk/protocol.go`**

Contains: `parseServerMessage`, `parseTick`, `parseTickEvent` (server→client parsing), and all `marshal*` functions (client→server). See spec "Wire Protocol Reference" for exact JSON formats.

Key: tick events use `{"type":"...","payload":{...}}` envelope. Presence messages (`heartbeat`, `set_status`, `set_zone`) do NOT use `payload` wrapper. Guild/task/chat DO use `payload` wrapper.

```go
package woasdk

import "encoding/json"

type serverMessage struct {
	Type  string
	Event Event
}

func parseServerMessage(data json.RawMessage) (serverMessage, error) {
	var envelope struct{ Type string `json:"type"` }
	if err := json.Unmarshal(data, &envelope); err != nil {
		return serverMessage{}, err
	}
	msg := serverMessage{Type: envelope.Type}
	switch envelope.Type {
	case "auth_required":
		// no payload
	case "welcome":
		var evt WelcomeEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return msg, err
		}
		msg.Event = &evt
	case "error":
		var evt ErrorEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return msg, err
		}
		msg.Event = &evt
	case "tick":
		evt, err := parseTick(data)
		if err != nil {
			return msg, err
		}
		msg.Event = evt
	}
	return msg, nil
}

func parseTick(data json.RawMessage) (*TickEvent, error) {
	var raw struct {
		Number uint64            `json:"number"`
		Events []json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	tick := &TickEvent{Number: raw.Number}
	for _, ed := range raw.Events {
		evt, err := parseTickEvent(ed)
		if err != nil || evt == nil {
			continue
		}
		tick.Events = append(tick.Events, evt)
	}
	return tick, nil
}

func parseTickEvent(data json.RawMessage) (Event, error) {
	var env struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	var evt Event
	switch env.Type {
	case "guild_created":
		var e GuildCreatedEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "member_joined":
		var e MemberJoinedEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "member_left":
		var e MemberLeftEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "task_created":
		var e TaskCreatedEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "task_claimed":
		var e TaskClaimedEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "task_completed":
		var e TaskCompletedEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "task_abandoned":
		var e TaskAbandonedEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "task_failed":
		var e TaskFailedEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "task_cancelled":
		var e TaskCancelledEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "message":
		var e MessageEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "agent_online":
		var e AgentOnlineEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "agent_offline":
		var e AgentOfflineEvent; json.Unmarshal(env.Payload, &e); evt = &e
	case "agent_status":
		var e AgentStatusEvent; json.Unmarshal(env.Payload, &e); evt = &e
	}
	return evt, nil
}

// --- Client-to-server marshalers ---

func marshalAuth(apiKey string) []byte {
	d, _ := json.Marshal(map[string]string{"type": "auth", "api_key": apiKey})
	return d
}

func marshalHeartbeat() []byte {
	d, _ := json.Marshal(map[string]string{"type": "heartbeat"})
	return d
}

// set_status and set_zone: flat (no payload wrapper)
func marshalSetStatus(status string) []byte {
	d, _ := json.Marshal(map[string]string{"type": "set_status", "status": status})
	return d
}

func marshalSetZone(zone string) []byte {
	d, _ := json.Marshal(map[string]string{"type": "set_zone", "zone": zone})
	return d
}

// Guild/task/chat: use payload wrapper
func marshalGuildCreate(name, description, visibility string) []byte {
	d, _ := json.Marshal(map[string]any{"type": "guild_create", "payload": map[string]string{
		"name": name, "description": description, "visibility": visibility,
	}})
	return d
}

func marshalGuildJoin(guildName string) []byte {
	d, _ := json.Marshal(map[string]any{"type": "guild_join", "payload": map[string]string{"guild_name": guildName}})
	return d
}

func marshalGuildLeave() []byte {
	d, _ := json.Marshal(map[string]string{"type": "guild_leave"})
	return d
}

func marshalTaskPost(title, description, priority string) []byte {
	d, _ := json.Marshal(map[string]any{"type": "task_post", "payload": map[string]string{
		"title": title, "description": description, "priority": priority,
	}})
	return d
}

func marshalTaskAction(actionType, taskID, result string) []byte {
	payload := map[string]string{"task_id": taskID}
	if result != "" {
		payload["result"] = result
	}
	d, _ := json.Marshal(map[string]any{"type": actionType, "payload": payload})
	return d
}

func marshalChatGuild(content string) []byte {
	d, _ := json.Marshal(map[string]any{"type": "message", "payload": map[string]string{
		"channel": "guild", "content": content,
	}})
	return d
}

func marshalChatDirect(toAgentID, content string) []byte {
	d, _ := json.Marshal(map[string]any{"type": "message", "payload": map[string]string{
		"channel": "direct", "content": content, "to": toAgentID,
	}})
	return d
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/lucasmeneses/mmoagens && go test ./pkg/woasdk/ -v`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/woasdk/protocol.go pkg/woasdk/protocol_test.go
git commit -m "feat(sdk): add protocol encoding/decoding with tests"
```

---

### Task 3: Client + Actions

**Files:**
- Create: `pkg/woasdk/client.go`
- Create: `pkg/woasdk/actions.go`

These two files are tightly coupled (client uses action interfaces, action implementations need `sender`). Created together, verified together.

- [ ] **Step 1: Create `pkg/woasdk/actions.go`**

Action interfaces + implementations using the internal `sender` interface.

```go
package woasdk

// GuildActions provides guild-related actions.
type GuildActions interface {
	Create(name, description, visibility string) error
	Join(guildName string) error
	Leave() error
}

type guildActions struct{ s sender }

func (g *guildActions) Create(name, desc, vis string) error { return g.s.send(marshalGuildCreate(name, desc, vis)) }
func (g *guildActions) Join(name string) error              { return g.s.send(marshalGuildJoin(name)) }
func (g *guildActions) Leave() error                        { return g.s.send(marshalGuildLeave()) }

// TaskActions provides task-related actions.
type TaskActions interface {
	Post(title, description, priority string) error
	Claim(taskID string) error
	Complete(taskID, result string) error
	Abandon(taskID string) error
	Fail(taskID string) error
	Cancel(taskID string) error
}

type taskActions struct{ s sender }

func (t *taskActions) Post(title, desc, pri string) error   { return t.s.send(marshalTaskPost(title, desc, pri)) }
func (t *taskActions) Claim(id string) error                { return t.s.send(marshalTaskAction("task_claim", id, "")) }
func (t *taskActions) Complete(id, result string) error     { return t.s.send(marshalTaskAction("task_complete", id, result)) }
func (t *taskActions) Abandon(id string) error              { return t.s.send(marshalTaskAction("task_abandon", id, "")) }
func (t *taskActions) Fail(id string) error                 { return t.s.send(marshalTaskAction("task_fail", id, "")) }
func (t *taskActions) Cancel(id string) error               { return t.s.send(marshalTaskAction("task_cancel", id, "")) }

// ChatActions provides chat-related actions.
type ChatActions interface {
	SendGuild(content string) error
	SendDirect(toAgentID, content string) error
}

type chatActions struct{ s sender }

func (c *chatActions) SendGuild(content string) error          { return c.s.send(marshalChatGuild(content)) }
func (c *chatActions) SendDirect(to, content string) error     { return c.s.send(marshalChatDirect(to, content)) }

// PresenceActions provides presence-related actions.
type PresenceActions interface {
	SetStatus(status string) error
	SetZone(zone string) error
	Heartbeat() error
}

type presenceActions struct{ s sender }

func (p *presenceActions) SetStatus(s string) error { return p.s.send(marshalSetStatus(s)) }
func (p *presenceActions) SetZone(z string) error   { return p.s.send(marshalSetZone(z)) }
func (p *presenceActions) Heartbeat() error         { return p.s.send(marshalHeartbeat()) }
```

- [ ] **Step 2: Create `pkg/woasdk/client.go`**

Handles: WebSocket connection, auth handshake (wait for `auth_required` → send auth → wait for `welcome`), background read/write/heartbeat loops, thread-safe event channel.

```go
package woasdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Config struct {
	ServerURL string // e.g. "ws://localhost:8083/ws"
	APIKey    string // e.g. "woa_abc123..."
}

type Client struct {
	conn      *websocket.Conn
	writeCh   chan []byte
	eventCh   chan Event
	agentID   string
	done      chan struct{}
	closeOnce sync.Once
	Guild     GuildActions
	Task      TaskActions
	Chat      ChatActions
	Presence  PresenceActions
}

// sender is the internal interface for sending messages over the WebSocket.
type sender interface {
	send(data []byte) error
}

func Connect(ctx context.Context, cfg Config) (*Client, error) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, cfg.ServerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("woasdk: dial: %w", err)
	}

	// 1. Wait for auth_required
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("woasdk: waiting for auth_required: %w", err)
	}
	msg, err := parseServerMessage(json.RawMessage(raw))
	if err != nil || msg.Type != "auth_required" {
		conn.Close()
		return nil, fmt.Errorf("woasdk: expected auth_required, got %q", msg.Type)
	}

	// 2. Send auth
	if err := conn.WriteMessage(websocket.TextMessage, marshalAuth(cfg.APIKey)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("woasdk: send auth: %w", err)
	}

	// 3. Wait for welcome or error
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err = conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("woasdk: waiting for welcome: %w", err)
	}
	msg, err = parseServerMessage(json.RawMessage(raw))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("woasdk: parse welcome: %w", err)
	}
	if msg.Type == "error" {
		conn.Close()
		evt := msg.Event.(*ErrorEvent)
		return nil, fmt.Errorf("woasdk: auth failed: [%s] %s", evt.Code, evt.Message)
	}
	if msg.Type != "welcome" {
		conn.Close()
		return nil, fmt.Errorf("woasdk: expected welcome, got %q", msg.Type)
	}

	welcome := msg.Event.(*WelcomeEvent)
	conn.SetReadDeadline(time.Time{})

	c := &Client{
		conn:    conn,
		writeCh: make(chan []byte, 64),
		eventCh: make(chan Event, 256),
		agentID: welcome.AgentID,
		done:    make(chan struct{}),
	}
	c.Guild = &guildActions{s: c}
	c.Task = &taskActions{s: c}
	c.Chat = &chatActions{s: c}
	c.Presence = &presenceActions{s: c}

	go c.readLoop()
	go c.writeLoop()
	go c.heartbeatLoop()
	return c, nil
}

func (c *Client) Events() <-chan Event { return c.eventCh }
func (c *Client) AgentID() string      { return c.agentID }

func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.done)
		err = c.conn.Close()
	})
	return err
}

func (c *Client) send(data []byte) error {
	select {
	case c.writeCh <- data:
		return nil
	case <-c.done:
		return errors.New("woasdk: client closed")
	}
}

func (c *Client) readLoop() {
	defer c.Close()
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			select {
			case <-c.done:
			default:
				c.pushEvent(&DisconnectEvent{Err: err})
			}
			return
		}
		msg, err := parseServerMessage(json.RawMessage(raw))
		if err != nil {
			slog.Warn("woasdk: parse error", "err", err)
			continue
		}
		switch msg.Type {
		case "tick":
			for _, evt := range msg.Event.(*TickEvent).Events {
				c.pushEvent(evt)
			}
		case "error":
			c.pushEvent(msg.Event)
		}
	}
}

func (c *Client) writeLoop() {
	for {
		select {
		case data := <-c.writeCh:
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				slog.Warn("woasdk: write error", "err", err)
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *Client) heartbeatLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = c.send(marshalHeartbeat())
		case <-c.done:
			return
		}
	}
}

func (c *Client) pushEvent(evt Event) {
	select {
	case c.eventCh <- evt:
	default:
		// Buffer full — drop oldest
		select {
		case <-c.eventCh:
		default:
		}
		select {
		case c.eventCh <- evt:
		default:
		}
	}
}
```

- [ ] **Step 3: Verify the SDK compiles**

Run: `cd /Users/lucasmeneses/mmoagens && go build ./pkg/woasdk/`
Expected: no errors

- [ ] **Step 4: Run all tests**

Run: `cd /Users/lucasmeneses/mmoagens && go test ./pkg/woasdk/ -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/woasdk/client.go pkg/woasdk/actions.go
git commit -m "feat(sdk): add client connection, auth handshake, and action methods"
```

---

### Task 4: Client integration tests

**Files:**
- Create: `pkg/woasdk/client_test.go`

Tests use external test package (`woasdk_test`) with a mock WebSocket server that simulates the full auth handshake and event streaming. Covers: successful connect, bad key rejection, event reception, and action wire format verification.

- [ ] **Step 1: Create `pkg/woasdk/client_test.go`**

```go
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

	// Should receive 2 events
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

	// Test multiple actions
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
```

- [ ] **Step 2: Run all SDK tests**

Run: `cd /Users/lucasmeneses/mmoagens && go test ./pkg/woasdk/ -v -count=1`
Expected: all PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/woasdk/client_test.go
git commit -m "test(sdk): add client integration tests with mock WebSocket server"
```

---

## Chunk 2: MCP Server

### Task 5: Event buffer

**Files:**
- Create: `cmd/woa-mcp/eventbuf.go`
- Create: `cmd/woa-mcp/eventbuf_test.go`

- [ ] **Step 1: Write `cmd/woa-mcp/eventbuf_test.go`**

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/lucasmeneses/mmoagens && go test ./cmd/woa-mcp/ -v -run TestEventBuf`
Expected: FAIL — `newEventBuf` not defined

- [ ] **Step 3: Implement `cmd/woa-mcp/eventbuf.go`**

```go
package main

import (
	"sync"

	"github.com/lucasmeneses/world-of-agents/pkg/woasdk"
)

type eventBuf struct {
	mu    sync.Mutex
	buf   []woasdk.Event
	cap   int
	start int
	count int
}

func newEventBuf(capacity int) *eventBuf {
	return &eventBuf{buf: make([]woasdk.Event, capacity), cap: capacity}
}

func (b *eventBuf) Push(evt woasdk.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	idx := (b.start + b.count) % b.cap
	b.buf[idx] = evt
	if b.count == b.cap {
		b.start = (b.start + 1) % b.cap
	} else {
		b.count++
	}
}

func (b *eventBuf) Drain() []woasdk.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.count == 0 {
		return nil
	}
	result := make([]woasdk.Event, b.count)
	for i := range b.count {
		result[i] = b.buf[(b.start+i)%b.cap]
	}
	b.start, b.count = 0, 0
	return result
}

func (b *eventBuf) Recent(n int) []woasdk.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.count == 0 {
		return nil
	}
	take := min(n, b.count)
	result := make([]woasdk.Event, take)
	off := b.count - take
	for i := range take {
		result[i] = b.buf[(b.start+off+i)%b.cap]
	}
	return result
}

func (b *eventBuf) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}
```

- [ ] **Step 4: Run tests**

Run: `cd /Users/lucasmeneses/mmoagens && go test ./cmd/woa-mcp/ -v -run TestEventBuf`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/woa-mcp/eventbuf.go cmd/woa-mcp/eventbuf_test.go
git commit -m "feat(mcp): add event ring buffer with tests"
```

---

### Task 6: WoAClient interface + handlers + MCP server

**Files:**
- Create: `cmd/woa-mcp/woaclient.go`
- Create: `cmd/woa-mcp/handlers.go`
- Create: `cmd/woa-mcp/tools.go`
- Create: `cmd/woa-mcp/main.go`

Handler logic is extracted into standalone functions in `handlers.go` that take `WoAClient` + `*eventBuf` and return `(string, error)`. This makes them testable without the MCP framework. `tools.go` wraps them as MCP tool handlers.

- [ ] **Step 1: Create `cmd/woa-mcp/woaclient.go`**

```go
package main

import "github.com/lucasmeneses/world-of-agents/pkg/woasdk"

// WoAClient abstracts the woasdk.Client for testability.
type WoAClient interface {
	AgentID() string
	Events() <-chan woasdk.Event
	Close() error

	GuildCreate(name, description, visibility string) error
	GuildJoin(guildName string) error
	GuildLeave() error

	TaskPost(title, description, priority string) error
	TaskClaim(taskID string) error
	TaskComplete(taskID, result string) error
	TaskAbandon(taskID string) error
	TaskFail(taskID string) error
	TaskCancel(taskID string) error

	SendGuild(content string) error
	SendDirect(toAgentID, content string) error

	SetStatus(status string) error
	SetZone(zone string) error
}

// sdkClient wraps a real woasdk.Client to satisfy WoAClient.
type sdkClient struct{ c *woasdk.Client }

func newSDKClient(c *woasdk.Client) WoAClient { return &sdkClient{c: c} }

func (s *sdkClient) AgentID() string                  { return s.c.AgentID() }
func (s *sdkClient) Events() <-chan woasdk.Event       { return s.c.Events() }
func (s *sdkClient) Close() error                     { return s.c.Close() }
func (s *sdkClient) GuildCreate(n, d, v string) error  { return s.c.Guild.Create(n, d, v) }
func (s *sdkClient) GuildJoin(name string) error       { return s.c.Guild.Join(name) }
func (s *sdkClient) GuildLeave() error                 { return s.c.Guild.Leave() }
func (s *sdkClient) TaskPost(t, d, p string) error     { return s.c.Task.Post(t, d, p) }
func (s *sdkClient) TaskClaim(id string) error         { return s.c.Task.Claim(id) }
func (s *sdkClient) TaskComplete(id, r string) error   { return s.c.Task.Complete(id, r) }
func (s *sdkClient) TaskAbandon(id string) error       { return s.c.Task.Abandon(id) }
func (s *sdkClient) TaskFail(id string) error          { return s.c.Task.Fail(id) }
func (s *sdkClient) TaskCancel(id string) error        { return s.c.Task.Cancel(id) }
func (s *sdkClient) SendGuild(content string) error    { return s.c.Chat.SendGuild(content) }
func (s *sdkClient) SendDirect(to, c string) error     { return s.c.Chat.SendDirect(to, c) }
func (s *sdkClient) SetStatus(status string) error     { return s.c.Presence.SetStatus(status) }
func (s *sdkClient) SetZone(zone string) error         { return s.c.Presence.SetZone(zone) }
```

- [ ] **Step 2: Create `cmd/woa-mcp/handlers.go`**

Testable handler functions — each takes WoAClient + eventBuf + params, returns (string, error).

```go
package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// formatEvents formats events as a JSON string for tool responses.
func formatEvents(buf *eventBuf) string {
	events := buf.Recent(20)
	if len(events) == 0 {
		return "[]"
	}
	items := make([]map[string]any, len(events))
	for i, evt := range events {
		items[i] = map[string]any{"type": evt.EventType(), "event": evt}
	}
	data, _ := json.MarshalIndent(items, "", "  ")
	return string(data)
}

func handleGuildCreate(wc WoAClient, buf *eventBuf, name, desc, vis string) (string, error) {
	if err := wc.GuildCreate(name, desc, vis); err != nil {
		return "", err
	}
	return fmt.Sprintf("Guild '%s' creation requested.\n\n## Recent Events\n```json\n%s\n```", name, formatEvents(buf)), nil
}

func handleGuildJoin(wc WoAClient, buf *eventBuf, name string) (string, error) {
	if err := wc.GuildJoin(name); err != nil {
		return "", err
	}
	return fmt.Sprintf("Join guild '%s' requested.\n\n## Recent Events\n```json\n%s\n```", name, formatEvents(buf)), nil
}

func handleGuildLeave(wc WoAClient, buf *eventBuf) (string, error) {
	if err := wc.GuildLeave(); err != nil {
		return "", err
	}
	return fmt.Sprintf("Leave guild requested.\n\n## Recent Events\n```json\n%s\n```", formatEvents(buf)), nil
}

func handleTaskPost(wc WoAClient, buf *eventBuf, title, desc, priority string) (string, error) {
	if err := wc.TaskPost(title, desc, priority); err != nil {
		return "", err
	}
	return fmt.Sprintf("Task '%s' posted.\n\n## Recent Events\n```json\n%s\n```", title, formatEvents(buf)), nil
}

func handleTaskAction(wc WoAClient, buf *eventBuf, action, taskID, result string) (string, error) {
	var err error
	switch action {
	case "claim":
		err = wc.TaskClaim(taskID)
	case "complete":
		err = wc.TaskComplete(taskID, result)
	case "abandon":
		err = wc.TaskAbandon(taskID)
	case "fail":
		err = wc.TaskFail(taskID)
	case "cancel":
		err = wc.TaskCancel(taskID)
	default:
		return "", fmt.Errorf("unknown task action: %s", action)
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Task %s %s requested.\n\n## Recent Events\n```json\n%s\n```", taskID, action, formatEvents(buf)), nil
}

func handleSendMessage(wc WoAClient, buf *eventBuf, channel, content, to string) (string, error) {
	switch channel {
	case "guild":
		if err := wc.SendGuild(content); err != nil {
			return "", err
		}
		return fmt.Sprintf("Guild message sent.\n\n## Recent Events\n```json\n%s\n```", formatEvents(buf)), nil
	case "direct":
		if to == "" {
			return "", fmt.Errorf("'to' parameter is required for direct messages")
		}
		if err := wc.SendDirect(to, content); err != nil {
			return "", err
		}
		return fmt.Sprintf("Direct message sent to %s.\n\n## Recent Events\n```json\n%s\n```", to, formatEvents(buf)), nil
	default:
		return "", fmt.Errorf("channel must be 'guild' or 'direct'")
	}
}

func handleGetEvents(buf *eventBuf) string {
	events := buf.Drain()
	if len(events) == 0 {
		return "No events buffered."
	}
	items := make([]map[string]any, len(events))
	for i, evt := range events {
		items[i] = map[string]any{"type": evt.EventType(), "event": evt}
	}
	data, _ := json.MarshalIndent(items, "", "  ")
	return string(data)
}

func handleWaitForEvents(buf *eventBuf, timeoutSec float64) string {
	if timeoutSec > 60 {
		timeoutSec = 60
	}
	if timeoutSec < 1 {
		timeoutSec = 1
	}
	deadline := time.After(time.Duration(timeoutSec) * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			events := buf.Drain()
			if len(events) == 0 {
				return "No events received within timeout."
			}
			return handleGetEvents_fromSlice(events)
		case <-ticker.C:
			if buf.Len() > 0 {
				return handleGetEvents_fromSlice(buf.Drain())
			}
		}
	}
}

func handleGetEvents_fromSlice(events []interface{ EventType() string }) string {
	items := make([]map[string]any, len(events))
	for i, evt := range events {
		items[i] = map[string]any{"type": evt.EventType(), "event": evt}
	}
	data, _ := json.MarshalIndent(items, "", "  ")
	return string(data)
}

// handleGetStatus returns agent status info.
// Note: v1 does not track guild/online-agents client-side (see Known Limitations in spec).
func handleGetStatus(wc WoAClient, buf *eventBuf) string {
	status := map[string]any{
		"agent_id":        wc.AgentID(),
		"events_buffered": buf.Len(),
	}
	data, _ := json.MarshalIndent(status, "", "  ")
	return fmt.Sprintf("```json\n%s\n```\n\n## Recent Events\n```json\n%s\n```", string(data), formatEvents(buf))
}
```

- [ ] **Step 3: Create `cmd/woa-mcp/tools.go`**

Thin MCP wrappers around the handler functions.

```go
package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func buildMCPServer(wc WoAClient, buf *eventBuf) *server.MCPServer {
	s := server.NewMCPServer("World of Agents", "1.0.0", server.WithToolCapabilities(false))
	registerGuildTools(s, wc, buf)
	registerTaskTools(s, wc, buf)
	registerChatTools(s, wc, buf)
	registerEventTools(s, wc, buf)
	registerStatusTools(s, wc, buf)
	return s
}

func registerGuildTools(s *server.MCPServer, wc WoAClient, buf *eventBuf) {
	s.AddTool(mcp.NewTool("guild_create",
		mcp.WithDescription("Create a new guild and join it"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Guild name")),
		mcp.WithString("description", mcp.Description("Guild description")),
		mcp.WithString("visibility", mcp.Description("Guild visibility"), mcp.Enum("public", "private")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, _ := req.RequireString("name")
		desc := req.GetString("description", "")
		vis := req.GetString("visibility", "public")
		text, err := handleGuildCreate(wc, buf, name, desc, vis)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(text), nil
	})

	s.AddTool(mcp.NewTool("guild_join",
		mcp.WithDescription("Join an existing guild"),
		mcp.WithString("guild_name", mcp.Required(), mcp.Description("Name of guild to join")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, _ := req.RequireString("guild_name")
		text, err := handleGuildJoin(wc, buf, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(text), nil
	})

	s.AddTool(mcp.NewTool("guild_leave",
		mcp.WithDescription("Leave the current guild"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text, err := handleGuildLeave(wc, buf)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(text), nil
	})
}

func registerTaskTools(s *server.MCPServer, wc WoAClient, buf *eventBuf) {
	s.AddTool(mcp.NewTool("task_post",
		mcp.WithDescription("Post a new task to the current guild"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
		mcp.WithString("description", mcp.Description("Task description")),
		mcp.WithString("priority", mcp.Description("Task priority"), mcp.Enum("low", "normal", "high", "urgent")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title, _ := req.RequireString("title")
		desc := req.GetString("description", "")
		pri := req.GetString("priority", "normal")
		text, err := handleTaskPost(wc, buf, title, desc, pri)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(text), nil
	})

	for _, tc := range []struct {
		name, desc string
		hasResult  bool
	}{
		{"task_claim", "Claim an open task", false},
		{"task_complete", "Complete a claimed task with a result", true},
		{"task_abandon", "Abandon a claimed task (reverts to open)", false},
		{"task_fail", "Mark a claimed task as failed", false},
		{"task_cancel", "Cancel a task (only task poster can cancel)", false},
	} {
		tc := tc // capture
		action := tc.name[5:] // strip "task_" prefix
		opts := []mcp.ToolOption{
			mcp.WithDescription(tc.desc),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID")),
		}
		if tc.hasResult {
			opts = append(opts, mcp.WithString("result", mcp.Required(), mcp.Description("Task result summary")))
		}
		s.AddTool(mcp.NewTool(tc.name, opts...), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id, _ := req.RequireString("task_id")
			result := ""
			if tc.hasResult {
				result, _ = req.RequireString("result")
			}
			text, err := handleTaskAction(wc, buf, action, id, result)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(text), nil
		})
	}
}

func registerChatTools(s *server.MCPServer, wc WoAClient, buf *eventBuf) {
	s.AddTool(mcp.NewTool("send_message",
		mcp.WithDescription("Send a message to the guild or directly to an agent"),
		mcp.WithString("channel", mcp.Required(), mcp.Description("Message channel"), mcp.Enum("guild", "direct")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Message content")),
		mcp.WithString("to", mcp.Description("Agent ID for direct messages (required if channel is 'direct')")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ch, _ := req.RequireString("channel")
		content, _ := req.RequireString("content")
		to := req.GetString("to", "")
		text, err := handleSendMessage(wc, buf, ch, content, to)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(text), nil
	})
}

func registerEventTools(s *server.MCPServer, wc WoAClient, buf *eventBuf) {
	s.AddTool(mcp.NewTool("get_events",
		mcp.WithDescription("Get all buffered events since last call and clear the buffer"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(handleGetEvents(buf)), nil
	})

	s.AddTool(mcp.NewTool("wait_for_events",
		mcp.WithDescription("Block until new events arrive or timeout expires"),
		mcp.WithNumber("timeout_seconds", mcp.Description("How long to wait (default 30, max 60)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		timeout := 30.0
		if v, err := req.RequireFloat("timeout_seconds"); err == nil {
			timeout = v
		}
		return mcp.NewToolResultText(handleWaitForEvents(buf, timeout)), nil
	})
}

func registerStatusTools(s *server.MCPServer, wc WoAClient, buf *eventBuf) {
	s.AddTool(mcp.NewTool("get_status",
		mcp.WithDescription("Get this agent's current status and connection info"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(handleGetStatus(wc, buf)), nil
	})
}
```

- [ ] **Step 4: Create `cmd/woa-mcp/main.go`**

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lucasmeneses/world-of-agents/pkg/woasdk"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	serverURL := os.Getenv("WOA_SERVER_URL")
	if serverURL == "" {
		serverURL = "ws://localhost:8083/ws"
	}
	apiKey := os.Getenv("WOA_API_KEY")
	if apiKey == "" {
		log.Fatal("WOA_API_KEY environment variable is required")
	}

	client, err := woasdk.Connect(context.Background(), woasdk.Config{
		ServerURL: serverURL, APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to connect to WoA server: %v", err)
	}
	defer client.Close()

	wc := newSDKClient(client)
	buf := newEventBuf(1000)

	go func() {
		for evt := range wc.Events() {
			buf.Push(evt)
		}
	}()

	mcpServer := buildMCPServer(wc, buf)
	fmt.Fprintf(os.Stderr, "woa-mcp: connected as %s\n", wc.AgentID())
	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
```

- [ ] **Step 5: Add mcp-go dependency and verify compilation**

```bash
cd /Users/lucasmeneses/mmoagens
go get github.com/mark3labs/mcp-go@latest
go build ./cmd/woa-mcp/
```

Expected: binary compiles successfully

- [ ] **Step 6: Commit**

```bash
git add cmd/woa-mcp/ go.mod go.sum
git commit -m "feat(mcp): add MCP server with all tool handlers"
```

---

### Task 7: Handler + tool tests

**Files:**
- Create: `cmd/woa-mcp/handlers_test.go`
- Create: `cmd/woa-mcp/tools_test.go`

Tests exercise the extracted handler functions directly with a mock WoAClient. `tools_test.go` verifies tool registration.

- [ ] **Step 1: Create `cmd/woa-mcp/handlers_test.go`**

```go
package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lucasmeneses/world-of-agents/pkg/woasdk"
)

type mockWoAClient struct {
	agentID   string
	eventCh   chan woasdk.Event
	lastCall  string
	lastArgs  []string
	returnErr error
}

func newMockClient() *mockWoAClient {
	return &mockWoAClient{agentID: "mock-001", eventCh: make(chan woasdk.Event, 10)}
}

func (m *mockWoAClient) AgentID() string            { return m.agentID }
func (m *mockWoAClient) Events() <-chan woasdk.Event { return m.eventCh }
func (m *mockWoAClient) Close() error                { return nil }

func (m *mockWoAClient) GuildCreate(n, d, v string) error {
	m.lastCall = "guild_create"; m.lastArgs = []string{n, d, v}; return m.returnErr
}
func (m *mockWoAClient) GuildJoin(n string) error {
	m.lastCall = "guild_join"; m.lastArgs = []string{n}; return m.returnErr
}
func (m *mockWoAClient) GuildLeave() error {
	m.lastCall = "guild_leave"; return m.returnErr
}
func (m *mockWoAClient) TaskPost(t, d, p string) error {
	m.lastCall = "task_post"; m.lastArgs = []string{t, d, p}; return m.returnErr
}
func (m *mockWoAClient) TaskClaim(id string) error {
	m.lastCall = "task_claim"; m.lastArgs = []string{id}; return m.returnErr
}
func (m *mockWoAClient) TaskComplete(id, r string) error {
	m.lastCall = "task_complete"; m.lastArgs = []string{id, r}; return m.returnErr
}
func (m *mockWoAClient) TaskAbandon(id string) error {
	m.lastCall = "task_abandon"; m.lastArgs = []string{id}; return m.returnErr
}
func (m *mockWoAClient) TaskFail(id string) error {
	m.lastCall = "task_fail"; m.lastArgs = []string{id}; return m.returnErr
}
func (m *mockWoAClient) TaskCancel(id string) error {
	m.lastCall = "task_cancel"; m.lastArgs = []string{id}; return m.returnErr
}
func (m *mockWoAClient) SendGuild(c string) error {
	m.lastCall = "send_guild"; m.lastArgs = []string{c}; return m.returnErr
}
func (m *mockWoAClient) SendDirect(to, c string) error {
	m.lastCall = "send_direct"; m.lastArgs = []string{to, c}; return m.returnErr
}
func (m *mockWoAClient) SetStatus(s string) error {
	m.lastCall = "set_status"; m.lastArgs = []string{s}; return m.returnErr
}
func (m *mockWoAClient) SetZone(z string) error {
	m.lastCall = "set_zone"; m.lastArgs = []string{z}; return m.returnErr
}

func TestHandleGuildCreate_Success(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	text, err := handleGuildCreate(mc, buf, "demo", "A guild", "public")
	if err != nil {
		t.Fatal(err)
	}
	if mc.lastCall != "guild_create" {
		t.Fatalf("expected guild_create, got %s", mc.lastCall)
	}
	if !strings.Contains(text, "demo") {
		t.Fatal("response should mention guild name")
	}
	if !strings.Contains(text, "Recent Events") {
		t.Fatal("response should include recent events")
	}
}

func TestHandleGuildCreate_Error(t *testing.T) {
	mc := newMockClient()
	mc.returnErr = fmt.Errorf("guild already exists")
	buf := newEventBuf(100)
	_, err := handleGuildCreate(mc, buf, "demo", "", "public")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHandleTaskPost_Success(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	text, err := handleTaskPost(mc, buf, "Fix bug", "Login broken", "high")
	if err != nil {
		t.Fatal(err)
	}
	if mc.lastArgs[0] != "Fix bug" || mc.lastArgs[2] != "high" {
		t.Fatalf("unexpected args: %v", mc.lastArgs)
	}
	if !strings.Contains(text, "Fix bug") {
		t.Fatal("response should mention task title")
	}
}

func TestHandleTaskAction_AllActions(t *testing.T) {
	for _, action := range []string{"claim", "complete", "abandon", "fail", "cancel"} {
		mc := newMockClient()
		buf := newEventBuf(100)
		result := ""
		if action == "complete" {
			result = "done"
		}
		text, err := handleTaskAction(mc, buf, action, "task-1", result)
		if err != nil {
			t.Fatalf("%s: %v", action, err)
		}
		if !strings.Contains(text, action) {
			t.Fatalf("%s: response should mention action", action)
		}
	}
}

func TestHandleSendMessage_Guild(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	text, err := handleSendMessage(mc, buf, "guild", "hello", "")
	if err != nil {
		t.Fatal(err)
	}
	if mc.lastCall != "send_guild" {
		t.Fatalf("expected send_guild, got %s", mc.lastCall)
	}
	if !strings.Contains(text, "Guild message sent") {
		t.Fatal("unexpected response")
	}
}

func TestHandleSendMessage_Direct(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	_, err := handleSendMessage(mc, buf, "direct", "hey", "agent-2")
	if err != nil {
		t.Fatal(err)
	}
	if mc.lastCall != "send_direct" || mc.lastArgs[0] != "agent-2" {
		t.Fatalf("expected send_direct to agent-2")
	}
}

func TestHandleSendMessage_DirectMissingTo(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	_, err := handleSendMessage(mc, buf, "direct", "hey", "")
	if err == nil {
		t.Fatal("expected error for missing 'to'")
	}
}

func TestHandleGetEvents_Empty(t *testing.T) {
	buf := newEventBuf(100)
	text := handleGetEvents(buf)
	if text != "No events buffered." {
		t.Fatalf("expected 'No events buffered.', got: %s", text)
	}
}

func TestHandleGetEvents_WithEvents(t *testing.T) {
	buf := newEventBuf(100)
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a1", AgentName: "Bot"})
	text := handleGetEvents(buf)
	if !strings.Contains(text, "agent_online") {
		t.Fatal("should contain event type")
	}
	// Should have drained
	if buf.Len() != 0 {
		t.Fatal("buffer should be empty after drain")
	}
}

func TestHandleGetStatus(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a1"})
	text := handleGetStatus(mc, buf)
	if !strings.Contains(text, "mock-001") {
		t.Fatal("should contain agent_id")
	}
	if !strings.Contains(text, "events_buffered") {
		t.Fatal("should contain events_buffered")
	}
}

func TestFormatEvents_IncludesRecentEvents(t *testing.T) {
	buf := newEventBuf(100)
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a1", AgentName: "Bot", AgentType: "explorer"})
	text := formatEvents(buf)
	if !strings.Contains(text, "agent_online") {
		t.Fatal("should contain event type")
	}
}
```

- [ ] **Step 2: Create `cmd/woa-mcp/tools_test.go`**

```go
package main

import "testing"

func TestBuildMCPServer_NotNil(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	s := buildMCPServer(mc, buf)
	if s == nil {
		t.Fatal("buildMCPServer returned nil")
	}
}
```

- [ ] **Step 3: Run all MCP tests**

Run: `cd /Users/lucasmeneses/mmoagens && go test ./cmd/woa-mcp/ -v -count=1`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/woa-mcp/handlers_test.go cmd/woa-mcp/tools_test.go
git commit -m "test(mcp): add handler unit tests and tool registration tests"
```

---

### Task 8: Build binary + final verification

**Files:**
- Modify: `.gitignore` (add `bin/`)

- [ ] **Step 1: Add `bin/` to `.gitignore`**

```bash
echo "bin/" >> /Users/lucasmeneses/mmoagens/.gitignore
```

- [ ] **Step 2: Build the MCP server binary**

Run: `cd /Users/lucasmeneses/mmoagens && go build -o bin/woa-mcp ./cmd/woa-mcp/`
Expected: binary created at `bin/woa-mcp`

- [ ] **Step 3: Verify it fails gracefully without WOA_API_KEY**

Run: `cd /Users/lucasmeneses/mmoagens && ./bin/woa-mcp 2>&1; true`
Expected output: `WOA_API_KEY environment variable is required`

- [ ] **Step 4: Run all tests (SDK + MCP)**

Run: `cd /Users/lucasmeneses/mmoagens && go test ./pkg/woasdk/ ./cmd/woa-mcp/ -v -count=1`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add .gitignore go.mod go.sum
git commit -m "feat: Phase 3 complete — Go SDK + MCP Server"
```
