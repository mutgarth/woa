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
