package ecs_test

import (
	"testing"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
)

type stubComponent struct{ Value string }
func (s *stubComponent) ComponentType() string { return "stub" }

type tickCounter struct {
	Count     int
	LastTick  uint64
	SeenCount int
}
func (t *tickCounter) Name() string { return "tick_counter" }
func (t *tickCounter) Update(w *ecs.World, tick uint64) {
	t.Count++
	t.LastTick = tick
	t.SeenCount = 0
	w.Each(func(e *ecs.Entity) {
		if e.Has("stub") {
			t.SeenCount++
		}
	})
}

func TestWorld_AddEntityAndQuery(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.NewEntity()
	e.Add(&stubComponent{Value: "hello"})
	w.AddEntity(e)
	got := w.Entity(e.ID)
	if got == nil { t.Fatal("expected to find entity") }
	c := got.Get("stub").(*stubComponent)
	if c.Value != "hello" { t.Fatalf("expected 'hello', got %q", c.Value) }
}

func TestWorld_RemoveEntity(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.NewEntity()
	w.AddEntity(e)
	w.RemoveEntity(e.ID)
	if w.Entity(e.ID) != nil { t.Fatal("expected entity to be removed") }
}

func TestWorld_Tick(t *testing.T) {
	w := ecs.NewWorld()
	counter := &tickCounter{}
	w.AddSystem(counter)
	e := ecs.NewEntity()
	e.Add(&stubComponent{Value: "x"})
	w.AddEntity(e)
	w.Tick(42)
	if counter.Count != 1 { t.Fatalf("expected 1 tick, got %d", counter.Count) }
	if counter.LastTick != 42 { t.Fatalf("expected tick 42, got %d", counter.LastTick) }
	if counter.SeenCount != 1 { t.Fatalf("expected 1 entity, got %d", counter.SeenCount) }
}
