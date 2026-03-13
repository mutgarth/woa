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
