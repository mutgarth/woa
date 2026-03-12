package ecs

import "github.com/google/uuid"

type EntityID = uuid.UUID

type Entity struct {
	ID         EntityID
	components map[string]Component
}

func NewEntity() *Entity {
	return &Entity{
		ID:         uuid.New(),
		components: make(map[string]Component),
	}
}

func NewEntityWithID(id EntityID) *Entity {
	return &Entity{
		ID:         id,
		components: make(map[string]Component),
	}
}

func (e *Entity) Add(c Component) {
	e.components[c.ComponentType()] = c
}

func (e *Entity) Remove(componentType string) {
	delete(e.components, componentType)
}

func (e *Entity) Get(componentType string) Component {
	return e.components[componentType]
}

func (e *Entity) Has(componentType string) bool {
	_, ok := e.components[componentType]
	return ok
}

func (e *Entity) HasAll(types ...string) bool {
	for _, t := range types {
		if !e.Has(t) {
			return false
		}
	}
	return true
}
