package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lucasmeneses/world-of-agents/pkg/woasdk"
)

type mockWoAClient struct {
	agentID   string
	eventCh   chan woasdk.Event
	lastCall  string
	lastArgs  []string
	returnErr error
}

func newMockClient() *mockWoAClient {
	return &mockWoAClient{agentID: "mock-001", eventCh: make(chan woasdk.Event, 10)}
}

func (m *mockWoAClient) AgentID() string            { return m.agentID }
func (m *mockWoAClient) Events() <-chan woasdk.Event { return m.eventCh }
func (m *mockWoAClient) Close() error                { return nil }

func (m *mockWoAClient) GuildCreate(n, d, v string) error {
	m.lastCall = "guild_create"; m.lastArgs = []string{n, d, v}; return m.returnErr
}
func (m *mockWoAClient) GuildJoin(n string) error {
	m.lastCall = "guild_join"; m.lastArgs = []string{n}; return m.returnErr
}
func (m *mockWoAClient) GuildLeave() error {
	m.lastCall = "guild_leave"; return m.returnErr
}
func (m *mockWoAClient) TaskPost(t, d, p string) error {
	m.lastCall = "task_post"; m.lastArgs = []string{t, d, p}; return m.returnErr
}
func (m *mockWoAClient) TaskClaim(id string) error {
	m.lastCall = "task_claim"; m.lastArgs = []string{id}; return m.returnErr
}
func (m *mockWoAClient) TaskComplete(id, r string) error {
	m.lastCall = "task_complete"; m.lastArgs = []string{id, r}; return m.returnErr
}
func (m *mockWoAClient) TaskAbandon(id string) error {
	m.lastCall = "task_abandon"; m.lastArgs = []string{id}; return m.returnErr
}
func (m *mockWoAClient) TaskFail(id string) error {
	m.lastCall = "task_fail"; m.lastArgs = []string{id}; return m.returnErr
}
func (m *mockWoAClient) TaskCancel(id string) error {
	m.lastCall = "task_cancel"; m.lastArgs = []string{id}; return m.returnErr
}
func (m *mockWoAClient) SendGuild(c string) error {
	m.lastCall = "send_guild"; m.lastArgs = []string{c}; return m.returnErr
}
func (m *mockWoAClient) SendDirect(to, c string) error {
	m.lastCall = "send_direct"; m.lastArgs = []string{to, c}; return m.returnErr
}
func (m *mockWoAClient) SetStatus(s string) error {
	m.lastCall = "set_status"; m.lastArgs = []string{s}; return m.returnErr
}
func (m *mockWoAClient) SetZone(z string) error {
	m.lastCall = "set_zone"; m.lastArgs = []string{z}; return m.returnErr
}

func TestHandleGuildCreate_Success(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	text, err := handleGuildCreate(mc, buf, "demo", "A guild", "public")
	if err != nil {
		t.Fatal(err)
	}
	if mc.lastCall != "guild_create" {
		t.Fatalf("expected guild_create, got %s", mc.lastCall)
	}
	if !strings.Contains(text, "demo") {
		t.Fatal("response should mention guild name")
	}
	if !strings.Contains(text, "Recent Events") {
		t.Fatal("response should include recent events")
	}
}

func TestHandleGuildCreate_Error(t *testing.T) {
	mc := newMockClient()
	mc.returnErr = fmt.Errorf("guild already exists")
	buf := newEventBuf(100)
	_, err := handleGuildCreate(mc, buf, "demo", "", "public")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHandleTaskPost_Success(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	text, err := handleTaskPost(mc, buf, "Fix bug", "Login broken", "high")
	if err != nil {
		t.Fatal(err)
	}
	if mc.lastArgs[0] != "Fix bug" || mc.lastArgs[2] != "high" {
		t.Fatalf("unexpected args: %v", mc.lastArgs)
	}
	if !strings.Contains(text, "Fix bug") {
		t.Fatal("response should mention task title")
	}
}

func TestHandleTaskAction_AllActions(t *testing.T) {
	for _, action := range []string{"claim", "complete", "abandon", "fail", "cancel"} {
		mc := newMockClient()
		buf := newEventBuf(100)
		result := ""
		if action == "complete" {
			result = "done"
		}
		text, err := handleTaskAction(mc, buf, action, "task-1", result)
		if err != nil {
			t.Fatalf("%s: %v", action, err)
		}
		if !strings.Contains(text, action) {
			t.Fatalf("%s: response should mention action", action)
		}
	}
}

func TestHandleSendMessage_Guild(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	text, err := handleSendMessage(mc, buf, "guild", "hello", "")
	if err != nil {
		t.Fatal(err)
	}
	if mc.lastCall != "send_guild" {
		t.Fatalf("expected send_guild, got %s", mc.lastCall)
	}
	if !strings.Contains(text, "Guild message sent") {
		t.Fatal("unexpected response")
	}
}

func TestHandleSendMessage_Direct(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	_, err := handleSendMessage(mc, buf, "direct", "hey", "agent-2")
	if err != nil {
		t.Fatal(err)
	}
	if mc.lastCall != "send_direct" || mc.lastArgs[0] != "agent-2" {
		t.Fatalf("expected send_direct to agent-2")
	}
}

func TestHandleSendMessage_DirectMissingTo(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	_, err := handleSendMessage(mc, buf, "direct", "hey", "")
	if err == nil {
		t.Fatal("expected error for missing 'to'")
	}
}

func TestHandleGetEvents_Empty(t *testing.T) {
	buf := newEventBuf(100)
	text := handleGetEvents(buf)
	if text != "No events buffered." {
		t.Fatalf("expected 'No events buffered.', got: %s", text)
	}
}

func TestHandleGetEvents_WithEvents(t *testing.T) {
	buf := newEventBuf(100)
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a1", AgentName: "Bot"})
	text := handleGetEvents(buf)
	if !strings.Contains(text, "agent_online") {
		t.Fatal("should contain event type")
	}
	if buf.Len() != 0 {
		t.Fatal("buffer should be empty after drain")
	}
}

func TestHandleGetStatus(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a1"})
	text := handleGetStatus(mc, buf)
	if !strings.Contains(text, "mock-001") {
		t.Fatal("should contain agent_id")
	}
	if !strings.Contains(text, "events_buffered") {
		t.Fatal("should contain events_buffered")
	}
}

func TestFormatEvents_IncludesRecentEvents(t *testing.T) {
	buf := newEventBuf(100)
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a1", AgentName: "Bot", AgentType: "explorer"})
	text := formatEvents(buf)
	if !strings.Contains(text, "agent_online") {
		t.Fatal("should contain event type")
	}
}

func TestHandleWaitForEvents_ImmediateReturn(t *testing.T) {
	buf := newEventBuf(100)
	buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a1"})
	text := handleWaitForEvents(context.Background(), buf, 5)
	if !strings.Contains(text, "agent_online") {
		t.Fatal("should return buffered events immediately")
	}
}

func TestHandleWaitForEvents_Timeout(t *testing.T) {
	buf := newEventBuf(100)
	start := time.Now()
	text := handleWaitForEvents(context.Background(), buf, 1)
	elapsed := time.Since(start)
	if elapsed < 900*time.Millisecond {
		t.Fatalf("returned too early: %v", elapsed)
	}
	if text != "No events received within timeout." {
		t.Fatalf("expected timeout message, got: %s", text)
	}
}

func TestHandleWaitForEvents_ContextCancelled(t *testing.T) {
	buf := newEventBuf(100)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	text := handleWaitForEvents(ctx, buf, 60)
	elapsed := time.Since(start)
	if elapsed > 2*time.Second {
		t.Fatalf("should have cancelled quickly, took %v", elapsed)
	}
	if text != "Request cancelled." {
		t.Fatalf("expected cancelled message, got: %s", text)
	}
}

func TestHandleWaitForEvents_EventsDuringWait(t *testing.T) {
	buf := newEventBuf(100)
	go func() {
		time.Sleep(300 * time.Millisecond)
		buf.Push(&woasdk.AgentOnlineEvent{AgentID: "a1"})
	}()
	text := handleWaitForEvents(context.Background(), buf, 5)
	if !strings.Contains(text, "agent_online") {
		t.Fatal("should return events that arrived during wait")
	}
}
