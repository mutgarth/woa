package ecs

import "sync"

type World struct {
	mu       sync.RWMutex
	entities map[EntityID]*Entity
	systems  []System
}

func NewWorld() *World {
	return &World{entities: make(map[EntityID]*Entity)}
}

func (w *World) AddEntity(e *Entity) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.entities[e.ID] = e
}

func (w *World) RemoveEntity(id EntityID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.entities, id)
}

func (w *World) Entity(id EntityID) *Entity {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.entities[id]
}

func (w *World) Each(fn func(e *Entity)) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, e := range w.entities {
		fn(e)
	}
}

func (w *World) AddSystem(s System) {
	w.systems = append(w.systems, s)
}

func (w *World) Tick(tickNum uint64) {
	for _, s := range w.systems {
		s.Update(w, tickNum)
	}
}

func (w *World) EntityCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.entities)
}
