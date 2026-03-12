package task

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain"
)

func TestClaim(t *testing.T) {
	guildID := uuid.New()
	poster := uuid.New()
	claimer := uuid.New()
	task := NewTask(guildID, poster, "Fix bug", "desc", PriorityNormal)
	if err := task.Claim(claimer); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if task.Status != StatusClaimed {
		t.Fatalf("expected claimed, got %s", task.Status)
	}
	if *task.ClaimedBy != claimer {
		t.Fatal("wrong claimer")
	}
}

func TestClaimAlreadyClaimed(t *testing.T) {
	task := NewTask(uuid.New(), uuid.New(), "T", "", PriorityNormal)
	task.Claim(uuid.New())
	if err := task.Claim(uuid.New()); err != domain.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestComplete(t *testing.T) {
	claimer := uuid.New()
	task := NewTask(uuid.New(), uuid.New(), "T", "", PriorityNormal)
	task.Claim(claimer)
	if err := task.Complete(claimer, "done"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if task.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", task.Status)
	}
	if task.Result != "done" {
		t.Fatal("wrong result")
	}
}

func TestCompleteNotClaimer(t *testing.T) {
	claimer := uuid.New()
	task := NewTask(uuid.New(), uuid.New(), "T", "", PriorityNormal)
	task.Claim(claimer)
	if err := task.Complete(uuid.New(), "done"); err != domain.ErrNotClaimer {
		t.Fatalf("expected ErrNotClaimer, got %v", err)
	}
}

func TestAbandon(t *testing.T) {
	claimer := uuid.New()
	task := NewTask(uuid.New(), uuid.New(), "T", "", PriorityNormal)
	task.Claim(claimer)
	if err := task.Abandon(claimer); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if task.Status != StatusOpen {
		t.Fatalf("expected open, got %s", task.Status)
	}
	if task.ClaimedBy != nil {
		t.Fatal("expected nil claimer")
	}
}

func TestFail(t *testing.T) {
	claimer := uuid.New()
	task := NewTask(uuid.New(), uuid.New(), "T", "", PriorityNormal)
	task.Claim(claimer)
	if err := task.Fail(claimer); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if task.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", task.Status)
	}
}

func TestCancel(t *testing.T) {
	poster := uuid.New()
	task := NewTask(uuid.New(), poster, "T", "", PriorityNormal)
	if err := task.Cancel(poster); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if task.Status != StatusCancelled {
		t.Fatalf("expected cancelled, got %s", task.Status)
	}
}

func TestCancelNotPoster(t *testing.T) {
	task := NewTask(uuid.New(), uuid.New(), "T", "", PriorityNormal)
	if err := task.Cancel(uuid.New()); err != domain.ErrPermissionDenied {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestCancelNotOpen(t *testing.T) {
	poster := uuid.New()
	task := NewTask(uuid.New(), poster, "T", "", PriorityNormal)
	task.Claim(uuid.New())
	if err := task.Cancel(poster); err != domain.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestTerminalStatesRejectTransitions(t *testing.T) {
	for _, status := range []Status{StatusCompleted, StatusFailed, StatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			task := &Task{Status: status, PostedBy: uuid.New()}
			claimer := uuid.New()
			task.ClaimedBy = &claimer
			if err := task.Claim(uuid.New()); err != domain.ErrInvalidTransition {
				t.Errorf("Claim on %s: expected ErrInvalidTransition", status)
			}
			if err := task.Complete(claimer, "r"); err != domain.ErrInvalidTransition {
				t.Errorf("Complete on %s: expected ErrInvalidTransition", status)
			}
			if err := task.Abandon(claimer); err != domain.ErrInvalidTransition {
				t.Errorf("Abandon on %s: expected ErrInvalidTransition", status)
			}
			if err := task.Fail(claimer); err != domain.ErrInvalidTransition {
				t.Errorf("Fail on %s: expected ErrInvalidTransition", status)
			}
			if err := task.Cancel(task.PostedBy); err != domain.ErrInvalidTransition {
				t.Errorf("Cancel on %s: expected ErrInvalidTransition", status)
			}
		})
	}
}
