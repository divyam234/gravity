package event

import (
	"sync"
)

type Bus struct {
	subscribers map[chan Event]struct{}
	mu          sync.RWMutex
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[chan Event]struct{}),
	}
}

func (b *Bus) Subscribe() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 100)
	b.subscribers[ch] = struct{}{}
	return ch
}

func (b *Bus) Unsubscribe(ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Convert receiver channel to sender channel for deletion
	for s := range b.subscribers {
		if (<-chan Event)(s) == ch {
			delete(b.subscribers, s)
			close(s)
			break
		}
	}
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			// Subscriber slow, skip or drop? For now just skip to avoid blocking bus
		}
	}
}
