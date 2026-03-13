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
	TaskID  string `json:"task_id"`
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
}

func (e TaskClaimedEvent) EventType() string { return "task_claimed" }

type TaskCompletedEvent struct {
	TaskID  string `json:"task_id"`
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
	Result  string `json:"result"`
}

func (e TaskCompletedEvent) EventType() string { return "task_completed" }

type TaskAbandonedEvent struct {
	TaskID  string `json:"task_id"`
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
}

func (e TaskAbandonedEvent) EventType() string { return "task_abandoned" }

type TaskFailedEvent struct {
	TaskID  string `json:"task_id"`
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
}

func (e TaskFailedEvent) EventType() string { return "task_failed" }

type TaskCancelledEvent struct {
	TaskID  string `json:"task_id"`
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
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
