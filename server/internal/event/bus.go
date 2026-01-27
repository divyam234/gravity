package event

import (
	"sync"
	"sync/atomic"
	"time"
)

// Bus is a typed event bus with separate channels for different event types
type Bus struct {
	// Typed channel subscribers
	progressSubs  sync.Map // map[chan ProgressEvent]struct{}
	lifecycleSubs sync.Map // map[chan LifecycleEvent]struct{}
	statsSubs     sync.Map // map[chan StatsEvent]struct{}

	// Unified subscribers (for SSE - receives all events as Event)
	unifiedSubs sync.Map // map[chan Event]struct{}

	// Subscriber count for lazy polling
	subscriberCount atomic.Int32
}

// NewBus creates a new event bus
func NewBus() *Bus {
	return &Bus{}
}

// HasSubscribers returns true if there are any active subscribers
func (b *Bus) HasSubscribers() bool {
	return b.subscriberCount.Load() > 0
}

// SubscriberCount returns the number of active subscribers
func (b *Bus) SubscriberCount() int {
	return int(b.subscriberCount.Load())
}

// --- Progress Channel ---

// SubscribeProgress subscribes to progress events only
func (b *Bus) SubscribeProgress() <-chan ProgressEvent {
	ch := make(chan ProgressEvent, 100)
	b.progressSubs.Store(ch, struct{}{})
	b.subscriberCount.Add(1)
	return ch
}

// UnsubscribeProgress unsubscribes from progress events
func (b *Bus) UnsubscribeProgress(ch <-chan ProgressEvent) {
	b.progressSubs.Range(func(key, value any) bool {
		s := key.(chan ProgressEvent)
		if (<-chan ProgressEvent)(s) == ch {
			b.progressSubs.Delete(s)
			b.subscriberCount.Add(-1)
			close(s)
			return false
		}
		return true
	})
}

// PublishProgress publishes a progress event to all progress subscribers
func (b *Bus) PublishProgress(e ProgressEvent) {
	// Publish to typed progress subscribers
	b.progressSubs.Range(func(key, value any) bool {
		ch := key.(chan ProgressEvent)
		select {
		case ch <- e:
		default:
			// Subscriber slow, drop to prevent blocking
		}
		return true
	})

	// Also publish to unified subscribers (SSE)
	eventType := DownloadProgress
	if e.Type == "upload" {
		eventType = UploadProgress
	}
	b.publishUnified(Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data: map[string]any{
			"id":               e.ID,
			"downloaded":       e.Downloaded,
			"uploaded":         e.Uploaded,
			"size":             e.Size,
			"speed":            e.Speed,
			"eta":              e.ETA,
			"seeders":          e.Seeders,
			"peers":            e.Peers,
			"metadataFetching": e.MetadataFetching,
		},
	})
}

// --- Lifecycle Channel ---

// SubscribeLifecycle subscribes to lifecycle events only
func (b *Bus) SubscribeLifecycle() <-chan LifecycleEvent {
	ch := make(chan LifecycleEvent, 50)
	b.lifecycleSubs.Store(ch, struct{}{})
	b.subscriberCount.Add(1)
	return ch
}

// UnsubscribeLifecycle unsubscribes from lifecycle events
func (b *Bus) UnsubscribeLifecycle(ch <-chan LifecycleEvent) {
	b.lifecycleSubs.Range(func(key, value any) bool {
		s := key.(chan LifecycleEvent)
		if (<-chan LifecycleEvent)(s) == ch {
			b.lifecycleSubs.Delete(s)
			b.subscriberCount.Add(-1)
			close(s)
			return false
		}
		return true
	})
}

// PublishLifecycle publishes a lifecycle event to all lifecycle subscribers
func (b *Bus) PublishLifecycle(e LifecycleEvent) {
	// Publish to typed lifecycle subscribers
	b.lifecycleSubs.Range(func(key, value any) bool {
		ch := key.(chan LifecycleEvent)
		select {
		case ch <- e:
		default:
			// Subscriber slow, drop to prevent blocking
		}
		return true
	})

	// Also publish to unified subscribers (SSE)
	b.publishUnified(Event{
		Type:      e.Type,
		Timestamp: e.Timestamp,
		Data:      e.Data,
	})
}

// --- Stats Channel ---

// SubscribeStats subscribes to stats events only
func (b *Bus) SubscribeStats() <-chan StatsEvent {
	ch := make(chan StatsEvent, 10)
	b.statsSubs.Store(ch, struct{}{})
	b.subscriberCount.Add(1)
	return ch
}

// UnsubscribeStats unsubscribes from stats events
func (b *Bus) UnsubscribeStats(ch <-chan StatsEvent) {
	b.statsSubs.Range(func(key, value any) bool {
		s := key.(chan StatsEvent)
		if (<-chan StatsEvent)(s) == ch {
			b.statsSubs.Delete(s)
			b.subscriberCount.Add(-1)
			close(s)
			return false
		}
		return true
	})
}

// PublishStats publishes a stats event to all stats subscribers
func (b *Bus) PublishStats(e StatsEvent) {
	// Publish to typed stats subscribers
	b.statsSubs.Range(func(key, value any) bool {
		ch := key.(chan StatsEvent)
		select {
		case ch <- e:
		default:
			// Subscriber slow, drop to prevent blocking
		}
		return true
	})

	// Also publish to unified subscribers (SSE)
	b.publishUnified(Event{
		Type:      StatsUpdate,
		Timestamp: time.Now(),
		Data:      e,
	})
}

// --- Unified Channel (for SSE) ---

// SubscribeAll returns a channel that receives all events as unified Event type
// This is used by SSE handlers that need to forward all events to clients
func (b *Bus) SubscribeAll() <-chan Event {
	ch := make(chan Event, 100)
	b.unifiedSubs.Store(ch, struct{}{})
	b.subscriberCount.Add(1)
	return ch
}

// UnsubscribeAll removes a subscriber from the unified channel
func (b *Bus) UnsubscribeAll(ch <-chan Event) {
	b.unifiedSubs.Range(func(key, value any) bool {
		s := key.(chan Event)
		if (<-chan Event)(s) == ch {
			b.unifiedSubs.Delete(s)
			b.subscriberCount.Add(-1)
			close(s)
			return false
		}
		return true
	})
}

// publishUnified sends an event to all unified subscribers
func (b *Bus) publishUnified(event Event) {
	b.unifiedSubs.Range(func(key, value any) bool {
		ch := key.(chan Event)
		select {
		case ch <- event:
		default:
			// Subscriber slow, drop to prevent blocking
		}
		return true
	})
}
