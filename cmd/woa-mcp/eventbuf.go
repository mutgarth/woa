package main

import (
	"sync"

	"github.com/lucasmeneses/world-of-agents/pkg/woasdk"
)

type eventBuf struct {
	mu    sync.Mutex
	buf   []woasdk.Event
	cap   int
	start int
	count int
}

func newEventBuf(capacity int) *eventBuf {
	return &eventBuf{buf: make([]woasdk.Event, capacity), cap: capacity}
}

func (b *eventBuf) Push(evt woasdk.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	idx := (b.start + b.count) % b.cap
	b.buf[idx] = evt
	if b.count == b.cap {
		b.start = (b.start + 1) % b.cap
	} else {
		b.count++
	}
}

func (b *eventBuf) Drain() []woasdk.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.count == 0 {
		return nil
	}
	result := make([]woasdk.Event, b.count)
	for i := range b.count {
		result[i] = b.buf[(b.start+i)%b.cap]
	}
	b.start, b.count = 0, 0
	return result
}

func (b *eventBuf) Recent(n int) []woasdk.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.count == 0 {
		return nil
	}
	take := min(n, b.count)
	result := make([]woasdk.Event, take)
	off := b.count - take
	for i := range take {
		result[i] = b.buf[(b.start+off+i)%b.cap]
	}
	return result
}

func (b *eventBuf) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}
