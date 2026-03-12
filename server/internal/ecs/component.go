package ecs

// Component is a marker interface for pure data attached to entities.
type Component interface {
	ComponentType() string
}
