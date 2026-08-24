package events

import (
	"sync"
	"sync/atomic"

	"gorocksdb/internal/clock"
)

type Event struct {
	Type      string         `json:"type"`
	Time      string         `json:"time"`
	Payload   map[string]any `json:"payload"`
}

type Bus struct {
	ch      chan Event
	dropped atomic.Int64
	mu      sync.RWMutex
	closed  bool
}

func New(buffer int) *Bus {
	if buffer < 1 {
		buffer = 16
	}
	return &Bus{ch: make(chan Event, buffer)}
}

func (b *Bus) Publish(typ string, payload map[string]any) {
	if payload == nil {
		payload = map[string]any{}
	}
	ev := Event{Type: typ, Time: clock.FormatNow(), Payload: payload}
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}
	select {
	case b.ch <- ev:
	default:
		b.dropped.Add(1)
	}
}

func (b *Bus) Chan() <-chan Event { return b.ch }

func (b *Bus) Dropped() int64 { return b.dropped.Load() }

func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	close(b.ch)
}
