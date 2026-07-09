package http

import (
	"sync"
	"sync/atomic"
)

type eventPayload struct {
	ID   uint64      `json:"id"`
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

type eventBroker struct {
	mu      sync.Mutex
	nextID  atomic.Uint64
	clients map[chan eventPayload]struct{}
}

func newEventBroker() *eventBroker {
	return &eventBroker{clients: map[chan eventPayload]struct{}{}}
}

func (b *eventBroker) subscribe() (chan eventPayload, func()) {
	ch := make(chan eventPayload, 32)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.clients, ch)
		close(ch)
		b.mu.Unlock()
	}
}

func (b *eventBroker) emit(name string, data interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	payload := eventPayload{ID: b.nextID.Add(1), Name: name, Data: data}
	for ch := range b.clients {
		select {
		case ch <- payload:
		default:
		}
	}
}
