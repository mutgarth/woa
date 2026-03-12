package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/guild"
)

// --- Mock TaskRepository ---
type mockTaskRepo struct {
	tasks map[uuid.UUID]*Task
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{tasks: make(map[uuid.UUID]*Task)}
}

func (m *mockTaskRepo) Create(_ context.Context, t *Task) error {
	m.tasks[t.ID] = t
	return nil
}

func (m *mockTaskRepo) GetByID(_ context.Context, id uuid.UUID) (*Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func (m *mockTaskRepo) Update(_ context.Context, t *Task) error {
	m.tasks[t.ID] = t
	return nil
}

func (m *mockTaskRepo) ListByGuild(_ context.Context, guildID uuid.UUID, status *Status, limit, offset int) ([]Task, error) {
	var result []Task
	for _, t := range m.tasks {
		if t.GuildID == guildID {
			if status == nil || t.Status == *status {
				result = append(result, *t)
			}
		}
	}
	return result, nil
}

// --- Mock GuildRepository (only GetMembership needed) ---
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
	key := guildID.String() + ":" + agentID.String()
	if !m.members[key] {
		return nil, domain.ErrNotMember
	}
	return &guild.Membership{GuildID: guildID, AgentID: agentID, Role: guild.RoleMember}, nil
}

// Unused interface methods
func (m *mockGuildRepo) Create(_ context.Context, _ *guild.Guild) error        { return nil }
func (m *mockGuildRepo) GetByID(_ context.Context, _ uuid.UUID) (*guild.Guild, error) {
	return nil, nil
}
func (m *mockGuildRepo) GetByName(_ context.Context, _ string) (*guild.Guild, error) {
	return nil, nil
}
func (m *mockGuildRepo) List(_ context.Context, _, _ int) ([]guild.Guild, error) { return nil, nil }
func (m *mockGuildRepo) AddMember(_ context.Context, _ *guild.Membership) error  { return nil }
func (m *mockGuildRepo) RemoveMember(_ context.Context, _, _ uuid.UUID) error    { return nil }
func (m *mockGuildRepo) ListMembers(_ context.Context, _ uuid.UUID) ([]guild.Membership, error) {
	return nil, nil
}
func (m *mockGuildRepo) CountMembers(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (m *mockGuildRepo) GetGuildByAgent(_ context.Context, _ uuid.UUID) (*guild.Guild, *guild.Membership, error) {
	return nil, nil, nil
}

// --- Tests ---
func TestPostTask(t *testing.T) {
	ctx := context.Background()
	taskRepo := newMockTaskRepo()
	guildRepo := newMockGuildRepo()
	guildID := uuid.New()
	agentID := uuid.New()
	guildRepo.addMember(guildID, agentID)
	svc := NewService(taskRepo, guildRepo)
	task, err := svc.Post(ctx, guildID, agentID, "Fix deploy", "It's broken", PriorityHigh)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if task.Title != "Fix deploy" {
		t.Fatal("wrong title")
	}
	if task.Status != StatusOpen {
		t.Fatal("expected open")
	}
	if task.GuildID != guildID {
		t.Fatal("wrong guild")
	}
}

func TestPostTaskNotMember(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newMockTaskRepo(), newMockGuildRepo())
	_, err := svc.Post(ctx, uuid.New(), uuid.New(), "T", "", PriorityNormal)
	if err != domain.ErrNotMember {
		t.Fatalf("expected ErrNotMember, got %v", err)
	}
}

func TestClaimTask(t *testing.T) {
	ctx := context.Background()
	taskRepo := newMockTaskRepo()
	guildRepo := newMockGuildRepo()
	guildID := uuid.New()
	poster := uuid.New()
	claimer := uuid.New()
	guildRepo.addMember(guildID, poster)
	guildRepo.addMember(guildID, claimer)
	svc := NewService(taskRepo, guildRepo)
	task, _ := svc.Post(ctx, guildID, poster, "T", "", PriorityNormal)
	claimed, err := svc.Claim(ctx, task.ID, claimer)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if claimed.Status != StatusClaimed {
		t.Fatal("expected claimed")
	}
}

func TestClaimTaskNotMember(t *testing.T) {
	ctx := context.Background()
	taskRepo := newMockTaskRepo()
	guildRepo := newMockGuildRepo()
	guildID := uuid.New()
	poster := uuid.New()
	guildRepo.addMember(guildID, poster)
	svc := NewService(taskRepo, guildRepo)
	task, _ := svc.Post(ctx, guildID, poster, "T", "", PriorityNormal)
	_, err := svc.Claim(ctx, task.ID, uuid.New())
	if err != domain.ErrNotMember {
		t.Fatalf("expected ErrNotMember, got %v", err)
	}
}

func TestFullTaskLifecycle(t *testing.T) {
	ctx := context.Background()
	taskRepo := newMockTaskRepo()
	guildRepo := newMockGuildRepo()
	guildID := uuid.New()
	poster := uuid.New()
	claimer := uuid.New()
	guildRepo.addMember(guildID, poster)
	guildRepo.addMember(guildID, claimer)
	svc := NewService(taskRepo, guildRepo)
	task, _ := svc.Post(ctx, guildID, poster, "T", "", PriorityNormal)
	task, _ = svc.Claim(ctx, task.ID, claimer)
	task, err := svc.Complete(ctx, task.ID, claimer, "Done!")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if task.Status != StatusCompleted {
		t.Fatal("expected completed")
	}
	if task.Result != "Done!" {
		t.Fatal("wrong result")
	}
}
