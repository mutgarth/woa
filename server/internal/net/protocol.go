package net

import "encoding/json"

type Envelope struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"-"`
}

func UnmarshalEnvelope(data []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return env, err
	}
	env.Raw = data
	return env, nil
}

type AuthMessage struct {
	Type   string `json:"type"`
	APIKey string `json:"api_key,omitempty"`
	Token  string `json:"token,omitempty"`
}

type HeartbeatMessage struct {
	Type   string `json:"type"`
	Status string `json:"status,omitempty"`
	Zone   string `json:"zone,omitempty"`
}

type SetStatusMessage struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type SetZoneMessage struct {
	Type string `json:"type"`
	Zone string `json:"zone"`
}

type AuthRequiredMessage struct {
	Type string `json:"type"`
}

type WelcomeMessage struct {
	Type            string `json:"type"`
	AgentID         string `json:"agent_id"`
	ServerTick      uint64 `json:"server_tick"`
	ProtocolVersion int    `json:"protocol_version"`
}

type AgentOnlineMessage struct {
	Type  string    `json:"type"`
	Agent AgentInfo `json:"agent"`
}

type AgentOfflineMessage struct {
	Type    string `json:"type"`
	AgentID string `json:"agent_id"`
	Reason  string `json:"reason"`
}

type AgentStatusMessage struct {
	Type    string `json:"type"`
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
	Zone    string `json:"zone"`
}

type ErrorMessage struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TickMessage struct {
	Type   string `json:"type"`
	Number uint64 `json:"number"`
	Events []any  `json:"events"`
}

type AgentInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Guild actions (client → server)
type GuildCreateMessage struct {
	Type    string `json:"type"`
	Payload struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Visibility  string `json:"visibility,omitempty"`
	} `json:"payload"`
}

type GuildJoinMessage struct {
	Type    string `json:"type"`
	Payload struct {
		GuildName string `json:"guild_name"`
	} `json:"payload"`
}

type GuildLeaveMessage struct {
	Type string `json:"type"`
}

// Guild events (server → client)
type GuildCreatedEvent struct {
	Type  string    `json:"type"`
	Guild GuildInfo `json:"guild"`
}

type GuildInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type MemberJoinedEvent struct {
	Type    string    `json:"type"`
	GuildID string    `json:"guild_id"`
	Agent   AgentInfo `json:"agent"`
}

type MemberLeftEvent struct {
	Type    string `json:"type"`
	GuildID string `json:"guild_id"`
	AgentID string `json:"agent_id"`
}

// --- Task messages ---

type TaskPostMessage struct {
	Type    string `json:"type"`
	Payload struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
	} `json:"payload"`
}

type TaskActionMessage struct {
	Type    string `json:"type"`
	Payload struct {
		TaskID string `json:"task_id"`
		Result string `json:"result,omitempty"` // only for task_complete
	} `json:"payload"`
}

// --- Chat messages ---

type ChatMessage struct {
	Type    string `json:"type"`
	Payload struct {
		Channel string `json:"channel"`
		Content string `json:"content"`
		To      string `json:"to,omitempty"` // for direct messages
	} `json:"payload"`
}
