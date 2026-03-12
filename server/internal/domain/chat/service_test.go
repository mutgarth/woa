package chat

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/guild"
)

// --- Mock MessageRepository ---

type mockMessageRepo struct {
	messages []*Message
}

func (m *mockMessageRepo) Create(_ context.Context, msg *Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockMessageRepo) ListByGuild(_ context.Context, guildID uuid.UUID, limit int) ([]Message, error) {
	var result []Message
	for _, msg := range m.messages {
		if msg.GuildID != nil && *msg.GuildID == guildID {
			result = append(result, *msg)
		}
	}
	if len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result, nil
}

func (m *mockMessageRepo) ListDirect(_ context.Context, agentA, agentB uuid.UUID, limit int) ([]Message, error) {
	var result []Message
	for _, msg := range m.messages {
		if msg.Channel == ChannelDirect {
			if (msg.FromAgent == agentA && msg.ToAgent != nil && *msg.ToAgent == agentB) ||
				(msg.FromAgent == agentB && msg.ToAgent != nil && *msg.ToAgent == agentA) {
				result = append(result, *msg)
			}
		}
	}
	return result, nil
}

// --- Mock GuildRepository (minimal) ---

type mockGuildRepo struct {
	members map[string]bool
}

func newMockGuildRepo() *mockGuildRepo {
	return &mockGuildRepo{members: make(map[string]bool)}
}

func (m *mockGuildRepo) addMember(guildID, agentID uuid.UUID) {
	m.members[guildID.String()+":"+agentID.String()] = true
}

func (m *mockGuildRepo) GetMembership(_ context.Context, guildID, agentID uuid.UUID) (*guild.Membership, error) {
	if !m.members[guildID.String()+":"+agentID.String()] {
		return nil, domain.ErrNotMember
	}
	return &guild.Membership{GuildID: guildID, AgentID: agentID, Role: guild.RoleMember}, nil
}

func (m *mockGuildRepo) Create(_ context.Context, _ *guild.Guild) error          { return nil }
func (m *mockGuildRepo) GetByID(_ context.Context, _ uuid.UUID) (*guild.Guild, error) { return nil, nil }
func (m *mockGuildRepo) GetByName(_ context.Context, _ string) (*guild.Guild, error)  { return nil, nil }
func (m *mockGuildRepo) List(_ context.Context, _, _ int) ([]guild.Guild, error)      { return nil, nil }
func (m *mockGuildRepo) AddMember(_ context.Context, _ *guild.Membership) error       { return nil }
func (m *mockGuildRepo) RemoveMember(_ context.Context, _, _ uuid.UUID) error         { return nil }
func (m *mockGuildRepo) ListMembers(_ context.Context, _ uuid.UUID) ([]guild.Membership, error) {
	return nil, nil
}
func (m *mockGuildRepo) CountMembers(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (m *mockGuildRepo) GetGuildByAgent(_ context.Context, _ uuid.UUID) (*guild.Guild, *guild.Membership, error) {
	return nil, nil, nil
}

// --- Tests ---

func TestSendGuildMessage(t *testing.T) {
	ctx := context.Background()
	msgRepo := &mockMessageRepo{}
	guildRepo := newMockGuildRepo()
	guildID := uuid.New()
	agentID := uuid.New()
	guildRepo.addMember(guildID, agentID)

	svc := NewService(msgRepo, guildRepo)
	msg, err := svc.SendGuild(ctx, guildID, agentID, "Hello guild!")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if msg.Channel != ChannelGuild {
		t.Fatal("expected guild channel")
	}
	if msg.Content != "Hello guild!" {
		t.Fatal("wrong content")
	}
}

func TestSendGuildMessageNotMember(t *testing.T) {
	ctx := context.Background()
	svc := NewService(&mockMessageRepo{}, newMockGuildRepo())
	_, err := svc.SendGuild(ctx, uuid.New(), uuid.New(), "Hello")
	if err != domain.ErrNotMember {
		t.Fatalf("expected ErrNotMember, got %v", err)
	}
}

func TestSendDirectMessage(t *testing.T) {
	ctx := context.Background()
	from := uuid.New()
	to := uuid.New()
	svc := NewService(&mockMessageRepo{}, newMockGuildRepo())
	msg, err := svc.SendDirect(ctx, from, to, "Hey!")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if msg.Channel != ChannelDirect {
		t.Fatal("expected direct channel")
	}
	if *msg.ToAgent != to {
		t.Fatal("wrong recipient")
	}
}

func TestGuildHistory(t *testing.T) {
	ctx := context.Background()
	msgRepo := &mockMessageRepo{}
	guildRepo := newMockGuildRepo()
	guildID := uuid.New()
	agentID := uuid.New()
	guildRepo.addMember(guildID, agentID)

	svc := NewService(msgRepo, guildRepo)
	svc.SendGuild(ctx, guildID, agentID, "msg1")
	svc.SendGuild(ctx, guildID, agentID, "msg2")

	history, err := svc.GuildHistory(ctx, guildID, 50)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
}
