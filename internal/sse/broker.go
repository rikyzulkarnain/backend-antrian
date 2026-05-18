package sse

import (
	"sync"
)

type Event struct {
	Name string
	Data []byte
}

type Broker struct {
	mu      sync.RWMutex
	clients map[chan Event]struct{}
}

func NewBroker() *Broker {
	return &Broker{clients: make(map[chan Event]struct{})}
}

func (b *Broker) Subscribe() chan Event {
	ch := make(chan Event, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Broker) Unsubscribe(ch chan Event) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}

func (b *Broker) Publish(ev Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- ev:
		default:
		}
	}
}
