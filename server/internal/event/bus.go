package event

import (
	"sync"
)

type Bus struct {
	subscribers sync.Map // map[chan Event]struct{}
}

func NewBus() *Bus {
	return &Bus{}
}

func (b *Bus) Subscribe() <-chan Event {
	ch := make(chan Event, 100)
	b.subscribers.Store(ch, struct{}{})
	return ch
}

func (b *Bus) Unsubscribe(ch <-chan Event) {
	// We need to find the specific sender channel that matches the receiver
	b.subscribers.Range(func(key, value any) bool {
		s := key.(chan Event)
		if (<-chan Event)(s) == ch {
			b.subscribers.Delete(s)
			close(s)
			return false
		}
		return true
	})
}

func (b *Bus) Publish(event Event) {
	b.subscribers.Range(func(key, value any) bool {
		ch := key.(chan Event)
		select {
		case ch <- event:
		default:
			// Subscriber slow, drop to prevent blocking
		}
		return true
	})
}
