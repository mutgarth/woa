package net

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/agent"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/auth"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	"github.com/lucasmeneses/world-of-agents/server/internal/engine"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	authTimeout = 5 * time.Second
	sendBufSize = 256
	writeWait   = 10 * time.Second
	pongWait    = 60 * time.Second
	pingPeriod  = (pongWait * 9) / 10
)

type Hub struct {
	mu          sync.RWMutex
	world       *ecs.World
	bus         *engine.EventBus
	authService *auth.Service
	ActionQueue chan IncomingAction
}

type IncomingAction struct {
	EntityID ecs.EntityID
	Envelope Envelope
	Raw      []byte
}

func NewHub(world *ecs.World, bus *engine.EventBus, authService *auth.Service) *Hub {
	return &Hub{
		world: world, bus: bus, authService: authService,
		ActionQueue: make(chan IncomingAction, 1024),
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	go h.handleConnection(conn)
}

func (h *Hub) handleConnection(conn *websocket.Conn) {
	authReq := AuthRequiredMessage{Type: "auth_required"}
	if err := conn.WriteJSON(authReq); err != nil {
		conn.Close()
		return
	}

	conn.SetReadDeadline(time.Now().Add(authTimeout))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		conn.WriteJSON(ErrorMessage{Type: "error", Code: "AUTH_TIMEOUT", Message: "auth timeout"})
		conn.Close()
		return
	}
	conn.SetReadDeadline(time.Time{})

	var authMsg AuthMessage
	if err := json.Unmarshal(msg, &authMsg); err != nil || authMsg.Type != "auth" {
		conn.WriteJSON(ErrorMessage{Type: "error", Code: "AUTH_FAILED", Message: "expected auth message"})
		conn.Close()
		return
	}

	ctx := context.Background()
	var agentEntity *agent.Agent
	if authMsg.APIKey != "" {
		agentEntity, err = h.authService.AuthenticateByAPIKey(ctx, authMsg.APIKey)
	} else if authMsg.Token != "" {
		_, err = h.authService.AuthenticateByToken(ctx, authMsg.Token)
		if err != nil {
			conn.WriteJSON(ErrorMessage{Type: "error", Code: "AUTH_FAILED", Message: "invalid token"})
			conn.Close()
			return
		}
		conn.WriteJSON(WelcomeMessage{Type: "welcome", AgentID: "viewer", ProtocolVersion: 1})
		h.readPumpViewer(conn)
		return
	}

	if err != nil || agentEntity == nil {
		conn.WriteJSON(ErrorMessage{Type: "error", Code: "AUTH_FAILED", Message: "invalid API key"})
		conn.Close()
		return
	}

	entity := ecs.NewEntityWithID(agentEntity.ID)
	entity.Add(&components.Identity{
		Name: agentEntity.Name, AgentType: string(agentEntity.AgentType),
		OwnerID: agentEntity.OwnerID, AgentDBID: agentEntity.ID,
	})
	entity.Add(&components.Presence{
		Status: components.StatusOnline, Zone: "", LastHeartbeat: time.Now(),
	})

	sendCh := make(chan []byte, sendBufSize)
	entity.Add(&components.Connection{
		Conn: conn, SessionID: uuid.New().String(), Send: sendCh,
	})

	h.world.AddEntity(entity)

	conn.WriteJSON(WelcomeMessage{
		Type: "welcome", AgentID: agentEntity.ID.String(), ProtocolVersion: 1,
	})

	h.bus.Publish(engine.Event{
		Type: "agent_online",
		Payload: map[string]any{
			"agent_id": agentEntity.ID.String(), "agent_name": agentEntity.Name,
			"agent_type": string(agentEntity.AgentType),
		},
		Scope: engine.GlobalScope(),
	})

	slog.Info("agent connected", "name", agentEntity.Name, "id", agentEntity.ID.String())

	go h.writePump(conn, sendCh)
	h.readPump(conn, entity.ID)

	h.world.RemoveEntity(entity.ID)
	h.bus.Publish(engine.Event{
		Type:    "agent_offline",
		Payload: map[string]any{"agent_id": agentEntity.ID.String(), "reason": "disconnect"},
		Scope:   engine.GlobalScope(),
	})
	slog.Info("agent disconnected", "name", agentEntity.Name)
}

func (h *Hub) readPump(conn *websocket.Conn, entityID ecs.EntityID) {
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		env, err := UnmarshalEnvelope(msg)
		if err != nil {
			continue
		}
		h.ActionQueue <- IncomingAction{EntityID: entityID, Envelope: env, Raw: msg}
	}
}

func (h *Hub) readPumpViewer(conn *websocket.Conn) {
	defer conn.Close()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writePump(conn *websocket.Conn, send chan []byte) {
	defer conn.Close()
	for msg := range send {
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (h *Hub) Broadcast(data []byte) {
	h.world.Each(func(e *ecs.Entity) {
		c := e.Get(components.ConnectionType)
		if c == nil {
			return
		}
		conn := c.(*components.Connection)
		select {
		case conn.Send <- data:
		default:
		}
	})
}
