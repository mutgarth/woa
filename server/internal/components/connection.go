package components

import "github.com/gorilla/websocket"

const ConnectionType = "connection"

type Connection struct {
	Conn      *websocket.Conn
	SessionID string
	Send      chan []byte
}

func (c *Connection) ComponentType() string { return ConnectionType }
