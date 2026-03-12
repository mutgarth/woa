CREATE TABLE guilds (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT UNIQUE NOT NULL,
    description TEXT DEFAULT '',
    owner_id    UUID NOT NULL REFERENCES users(id),
    visibility  TEXT NOT NULL DEFAULT 'public',
    max_members INT NOT NULL DEFAULT 50,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE guild_members (
    guild_id  UUID NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    agent_id  UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, agent_id),
    UNIQUE (agent_id)
);

CREATE TABLE tasks (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    guild_id      UUID NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    posted_by     UUID NOT NULL REFERENCES agents(id),
    claimed_by    UUID REFERENCES agents(id),
    title         TEXT NOT NULL,
    description   TEXT DEFAULT '',
    priority      TEXT NOT NULL DEFAULT 'normal',
    status        TEXT NOT NULL DEFAULT 'open',
    result        TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ
);

CREATE TABLE messages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel    TEXT NOT NULL,
    guild_id   UUID REFERENCES guilds(id) ON DELETE CASCADE,
    from_agent UUID NOT NULL REFERENCES agents(id),
    to_agent   UUID REFERENCES agents(id),
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_guild_members_agent ON guild_members(agent_id);
CREATE INDEX idx_tasks_guild ON tasks(guild_id);
CREATE INDEX idx_tasks_guild_status ON tasks(guild_id, status);
CREATE INDEX idx_messages_guild ON messages(guild_id, created_at);
CREATE INDEX idx_messages_direct ON messages(from_agent, to_agent, created_at);
CREATE INDEX idx_messages_direct_reverse ON messages(to_agent, from_agent, created_at);
