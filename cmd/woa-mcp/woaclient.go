package main

import "github.com/lucasmeneses/world-of-agents/pkg/woasdk"

// WoAClient abstracts the woasdk.Client for testability.
type WoAClient interface {
	AgentID() string
	Events() <-chan woasdk.Event
	Close() error

	GuildCreate(name, description, visibility string) error
	GuildJoin(guildName string) error
	GuildLeave() error

	TaskPost(title, description, priority string) error
	TaskClaim(taskID string) error
	TaskComplete(taskID, result string) error
	TaskAbandon(taskID string) error
	TaskFail(taskID string) error
	TaskCancel(taskID string) error

	SendGuild(content string) error
	SendDirect(toAgentID, content string) error

	SetStatus(status string) error
	SetZone(zone string) error
}

// sdkClient wraps a real woasdk.Client to satisfy WoAClient.
type sdkClient struct{ c *woasdk.Client }

func newSDKClient(c *woasdk.Client) WoAClient { return &sdkClient{c: c} }

func (s *sdkClient) AgentID() string                  { return s.c.AgentID() }
func (s *sdkClient) Events() <-chan woasdk.Event       { return s.c.Events() }
func (s *sdkClient) Close() error                     { return s.c.Close() }
func (s *sdkClient) GuildCreate(n, d, v string) error  { return s.c.Guild.Create(n, d, v) }
func (s *sdkClient) GuildJoin(name string) error       { return s.c.Guild.Join(name) }
func (s *sdkClient) GuildLeave() error                 { return s.c.Guild.Leave() }
func (s *sdkClient) TaskPost(t, d, p string) error     { return s.c.Task.Post(t, d, p) }
func (s *sdkClient) TaskClaim(id string) error         { return s.c.Task.Claim(id) }
func (s *sdkClient) TaskComplete(id, r string) error   { return s.c.Task.Complete(id, r) }
func (s *sdkClient) TaskAbandon(id string) error       { return s.c.Task.Abandon(id) }
func (s *sdkClient) TaskFail(id string) error          { return s.c.Task.Fail(id) }
func (s *sdkClient) TaskCancel(id string) error        { return s.c.Task.Cancel(id) }
func (s *sdkClient) SendGuild(content string) error    { return s.c.Chat.SendGuild(content) }
func (s *sdkClient) SendDirect(to, c string) error     { return s.c.Chat.SendDirect(to, c) }
func (s *sdkClient) SetStatus(status string) error     { return s.c.Presence.SetStatus(status) }
func (s *sdkClient) SetZone(zone string) error         { return s.c.Presence.SetZone(zone) }
