package woasdk

type GuildActions interface {
	Create(name, description, visibility string) error
	Join(guildName string) error
	Leave() error
}

type guildActions struct{ s sender }

func (g *guildActions) Create(name, desc, vis string) error { return g.s.send(marshalGuildCreate(name, desc, vis)) }
func (g *guildActions) Join(name string) error              { return g.s.send(marshalGuildJoin(name)) }
func (g *guildActions) Leave() error                        { return g.s.send(marshalGuildLeave()) }

type TaskActions interface {
	Post(title, description, priority string) error
	Claim(taskID string) error
	Complete(taskID, result string) error
	Abandon(taskID string) error
	Fail(taskID string) error
	Cancel(taskID string) error
}

type taskActions struct{ s sender }

func (t *taskActions) Post(title, desc, pri string) error { return t.s.send(marshalTaskPost(title, desc, pri)) }
func (t *taskActions) Claim(id string) error              { return t.s.send(marshalTaskAction("task_claim", id, "")) }
func (t *taskActions) Complete(id, result string) error   { return t.s.send(marshalTaskAction("task_complete", id, result)) }
func (t *taskActions) Abandon(id string) error            { return t.s.send(marshalTaskAction("task_abandon", id, "")) }
func (t *taskActions) Fail(id string) error               { return t.s.send(marshalTaskAction("task_fail", id, "")) }
func (t *taskActions) Cancel(id string) error             { return t.s.send(marshalTaskAction("task_cancel", id, "")) }

type ChatActions interface {
	SendGuild(content string) error
	SendDirect(toAgentID, content string) error
}

type chatActions struct{ s sender }

func (c *chatActions) SendGuild(content string) error        { return c.s.send(marshalChatGuild(content)) }
func (c *chatActions) SendDirect(to, content string) error   { return c.s.send(marshalChatDirect(to, content)) }

type PresenceActions interface {
	SetStatus(status string) error
	SetZone(zone string) error
	Heartbeat() error
}

type presenceActions struct{ s sender }

func (p *presenceActions) SetStatus(s string) error { return p.s.send(marshalSetStatus(s)) }
func (p *presenceActions) SetZone(z string) error   { return p.s.send(marshalSetZone(z)) }
func (p *presenceActions) Heartbeat() error         { return p.s.send(marshalHeartbeat()) }
