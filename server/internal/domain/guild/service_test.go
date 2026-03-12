package guild_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/guild"
)

type mockGuildRepo struct {
	guilds     map[uuid.UUID]*guild.Guild
	members    map[uuid.UUID][]guild.Membership
	agentGuild map[uuid.UUID]uuid.UUID
}

func newMockRepo() *mockGuildRepo {
	return &mockGuildRepo{
		guilds:     make(map[uuid.UUID]*guild.Guild),
		members:    make(map[uuid.UUID][]guild.Membership),
		agentGuild: make(map[uuid.UUID]uuid.UUID),
	}
}

func (m *mockGuildRepo) Create(_ context.Context, g *guild.Guild) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	for _, existing := range m.guilds {
		if existing.Name == g.Name {
			return domain.ErrAlreadyExists
		}
	}
	m.guilds[g.ID] = g
	return nil
}

func (m *mockGuildRepo) GetByID(_ context.Context, id uuid.UUID) (*guild.Guild, error) {
	g, ok := m.guilds[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return g, nil
}

func (m *mockGuildRepo) GetByName(_ context.Context, name string) (*guild.Guild, error) {
	for _, g := range m.guilds {
		if g.Name == name {
			return g, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockGuildRepo) List(_ context.Context, limit, offset int) ([]guild.Guild, error) {
	var result []guild.Guild
	for _, g := range m.guilds {
		result = append(result, *g)
	}
	return result, nil
}

func (m *mockGuildRepo) AddMember(_ context.Context, mem *guild.Membership) error {
	if _, exists := m.agentGuild[mem.AgentID]; exists {
		return domain.ErrAlreadyMember
	}
	m.members[mem.GuildID] = append(m.members[mem.GuildID], *mem)
	m.agentGuild[mem.AgentID] = mem.GuildID
	return nil
}

func (m *mockGuildRepo) RemoveMember(_ context.Context, guildID, agentID uuid.UUID) error {
	members := m.members[guildID]
	for i, mem := range members {
		if mem.AgentID == agentID {
			m.members[guildID] = append(members[:i], members[i+1:]...)
			delete(m.agentGuild, agentID)
			return nil
		}
	}
	return domain.ErrNotMember
}

func (m *mockGuildRepo) GetMembership(_ context.Context, guildID, agentID uuid.UUID) (*guild.Membership, error) {
	for _, mem := range m.members[guildID] {
		if mem.AgentID == agentID {
			return &mem, nil
		}
	}
	return nil, domain.ErrNotMember
}

func (m *mockGuildRepo) ListMembers(_ context.Context, guildID uuid.UUID) ([]guild.Membership, error) {
	return m.members[guildID], nil
}

func (m *mockGuildRepo) CountMembers(_ context.Context, guildID uuid.UUID) (int, error) {
	return len(m.members[guildID]), nil
}

func (m *mockGuildRepo) GetGuildByAgent(_ context.Context, agentID uuid.UUID) (*guild.Guild, *guild.Membership, error) {
	guildID, ok := m.agentGuild[agentID]
	if !ok {
		return nil, nil, domain.ErrNotMember
	}
	g := m.guilds[guildID]
	for _, mem := range m.members[guildID] {
		if mem.AgentID == agentID {
			return g, &mem, nil
		}
	}
	return nil, nil, domain.ErrNotMember
}

func TestGuildService_CreateAndJoin(t *testing.T) {
	repo := newMockRepo()
	svc := guild.NewService(repo, nil)

	ownerID := uuid.New()
	agentID := uuid.New()

	g, err := svc.Create(context.Background(), "test-guild", "A test guild", "public", ownerID, agentID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if g.Name != "test-guild" {
		t.Errorf("expected name test-guild, got %s", g.Name)
	}

	mem, err := repo.GetMembership(context.Background(), g.ID, agentID)
	if err != nil {
		t.Fatalf("GetMembership: %v", err)
	}
	if mem.Role != guild.RoleOwner {
		t.Errorf("expected owner role, got %s", mem.Role)
	}

	agent2 := uuid.New()
	_, err = svc.Join(context.Background(), "test-guild", agent2)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	_, err = svc.Join(context.Background(), "test-guild", agent2)
	if err == nil {
		t.Fatal("expected error for double join")
	}
}

func TestGuildService_Leave(t *testing.T) {
	repo := newMockRepo()
	svc := guild.NewService(repo, nil)

	ownerID := uuid.New()
	creatorAgent := uuid.New()
	memberAgent := uuid.New()

	g, _ := svc.Create(context.Background(), "leave-guild", "", "public", ownerID, creatorAgent)
	svc.Join(context.Background(), "leave-guild", memberAgent)

	if err := svc.Leave(context.Background(), memberAgent); err != nil {
		t.Fatalf("Leave: %v", err)
	}

	if err := svc.Leave(context.Background(), creatorAgent); err == nil {
		t.Fatal("expected error when owner tries to leave")
	}

	_ = g
}

func TestGuildService_GuildFull(t *testing.T) {
	repo := newMockRepo()
	svc := guild.NewService(repo, nil)

	ownerID := uuid.New()
	creatorAgent := uuid.New()

	g, _ := svc.Create(context.Background(), "small-guild", "", "public", ownerID, creatorAgent)
	g.MaxMembers = 2
	repo.guilds[g.ID] = g

	agent2 := uuid.New()
	_, err := svc.Join(context.Background(), "small-guild", agent2)
	if err != nil {
		t.Fatalf("Join agent2: %v", err)
	}

	agent3 := uuid.New()
	_, err = svc.Join(context.Background(), "small-guild", agent3)
	if err == nil {
		t.Fatal("expected guild full error")
	}
}
