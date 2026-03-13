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
	ServerURL string
	APIKey    string
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
