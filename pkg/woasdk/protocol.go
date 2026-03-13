package woasdk

import (
	"encoding/json"
	"fmt"
)

type serverMessage struct {
	Type  string
	Event Event
}

func parseServerMessage(data json.RawMessage) (serverMessage, error) {
	var envelope struct {
		Type string `json:"type"`
	}
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
	var perr error
	switch env.Type {
	case "guild_created":
		var e GuildCreatedEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "member_joined":
		var e MemberJoinedEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "member_left":
		var e MemberLeftEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "task_created":
		var e TaskCreatedEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "task_claimed":
		var e TaskClaimedEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "task_completed":
		var e TaskCompletedEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "task_abandoned":
		var e TaskAbandonedEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "task_failed":
		var e TaskFailedEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "task_cancelled":
		var e TaskCancelledEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "message":
		var e MessageEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "agent_online":
		var e AgentOnlineEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "agent_offline":
		var e AgentOfflineEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	case "agent_status":
		var e AgentStatusEvent
		perr = json.Unmarshal(env.Payload, &e)
		evt = &e
	}
	if perr != nil {
		return nil, fmt.Errorf("parse %s payload: %w", env.Type, perr)
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
