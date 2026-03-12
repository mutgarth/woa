package systems

import (
	"encoding/json"

	"github.com/lucasmeneses/world-of-agents/server/internal/components"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
)

// ActionHandler is implemented by GuildSystem, TaskSystem, ChatSystem.
// ActionRouter dispatches to these via this interface so it doesn't
// need concrete type references at compile time.
type ActionHandler interface {
	HandleAction(world *ecs.World, action wonet.IncomingAction)
}

func sendError(entity *ecs.Entity, code, message string) {
	c := entity.Get(components.ConnectionType)
	if c == nil {
		return
	}
	conn := c.(*components.Connection)
	errMsg := wonet.ErrorMessage{Type: "error", Code: code, Message: message}
	data, _ := json.Marshal(errMsg)
	select {
	case conn.Send <- data:
	default:
	}
}
