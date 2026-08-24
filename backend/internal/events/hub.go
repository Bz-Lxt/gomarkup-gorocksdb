package events

import "sync"

type Hub struct {
	mu      sync.Mutex
	clients map[chan Event]struct{}
}

func NewHub(bus *Bus) *Hub {
	h := &Hub{clients: make(map[chan Event]struct{})}
	go h.loop(bus)
	return h
}

func (h *Hub) loop(bus *Bus) {
	for ev := range bus.Chan() {
		h.mu.Lock()
		for ch := range h.clients {
			select {
			case ch <- ev:
			default:
			}
		}
		h.mu.Unlock()
	}
}

func (h *Hub) Subscribe() chan Event {
	ch := make(chan Event, 64)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan Event) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}
