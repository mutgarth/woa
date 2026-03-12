package ecs

// System processes entities each tick.
type System interface {
	Name() string
	Update(world *World, tick uint64)
}
