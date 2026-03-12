package engine

import (
	"sync/atomic"
	"time"
	"github.com/lucasmeneses/world-of-agents/server/internal/ecs"
)

type TickEngine struct {
	world    *ecs.World
	bus      *EventBus
	interval time.Duration
	tick     atomic.Uint64
	stop     chan struct{}
}

func NewTickEngine(world *ecs.World, bus *EventBus, interval time.Duration) *TickEngine {
	return &TickEngine{
		world:    world,
		bus:      bus,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (e *TickEngine) Start() {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	for {
		select {
		case <-e.stop:
			return
		case <-ticker.C:
			next := e.tick.Add(1)
			e.world.Tick(next)
		}
	}
}

func (e *TickEngine) Stop() { close(e.stop) }
func (e *TickEngine) CurrentTick() uint64 { return e.tick.Load() }
func (e *TickEngine) Bus() *EventBus { return e.bus }
