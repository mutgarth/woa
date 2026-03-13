package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lucasmeneses/world-of-agents/pkg/woasdk"
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
	return handleGetEvents_fromSlice(events)
}

func handleWaitForEvents(ctx context.Context, buf *eventBuf, timeoutSec float64) string {
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
		case <-ctx.Done():
			return "Request cancelled."
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

func handleGetEvents_fromSlice(events []woasdk.Event) string {
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
