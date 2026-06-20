package broker

import (
	"context"
	"errors"
	"sync"
)

// Broker defines the interface for publishing and subscribing to events.
type Broker interface {
	PublishRaw(ctx context.Context, event *TrackingEvent) error
	SubscribeRaw(ctx context.Context) (<-chan *TrackingEvent, error)

	PublishScreened(ctx context.Context, event *ScreenedEvent) error
	SubscribeScreened(ctx context.Context) (<-chan *ScreenedEvent, error)
	
	Close() error
}

// InMemoryBroker implements Broker using Go channels and fan-out distribution.
type InMemoryBroker struct {
	mu              sync.RWMutex
	rawSubs         []chan *TrackingEvent
	screenedSubs    []chan *ScreenedEvent
	closed          bool
	bufferSize      int
}

// NewInMemoryBroker creates a new thread-safe in-memory pub-sub broker.
func NewInMemoryBroker(bufferSize int) *InMemoryBroker {
	if bufferSize <= 0 {
		bufferSize = 10000 // Default high-capacity buffer
	}
	return &InMemoryBroker{
		rawSubs:      make([]chan *TrackingEvent, 0),
		screenedSubs: make([]chan *ScreenedEvent, 0),
		bufferSize:   bufferSize,
	}
}

// PublishRaw broadcasts an incoming raw tracking event to all active raw subscribers.
func (b *InMemoryBroker) PublishRaw(ctx context.Context, event *TrackingEvent) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return errors.New("broker is closed")
	}

	for _, sub := range b.rawSubs {
		select {
		case sub <- event:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Buffer full, drop or block. For high-throughput simulation, we log/drop to prevent blocking ingestion.
			// In production, backpressure or persistent queues like Kafka/Redis are used.
		}
	}
	return nil
}

// SubscribeRaw registers a subscriber for raw tracking events.
func (b *InMemoryBroker) SubscribeRaw(ctx context.Context) (<-chan *TrackingEvent, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, errors.New("broker is closed")
	}

	ch := make(chan *TrackingEvent, b.bufferSize)
	b.rawSubs = append(b.rawSubs, ch)
	return ch, nil
}

// PublishScreened broadcasts a fraud-screened event to all active screened subscribers.
func (b *InMemoryBroker) PublishScreened(ctx context.Context, event *ScreenedEvent) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return errors.New("broker is closed")
	}

	for _, sub := range b.screenedSubs {
		select {
		case sub <- event:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Drop to avoid blocking the broker thread if one subscriber is slow
		}
	}
	return nil
}

// SubscribeScreened registers a subscriber for fraud-screened events.
func (b *InMemoryBroker) SubscribeScreened(ctx context.Context) (<-chan *ScreenedEvent, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, errors.New("broker is closed")
	}

	ch := make(chan *ScreenedEvent, b.bufferSize)
	b.screenedSubs = append(b.screenedSubs, ch)
	return ch, nil
}

// Close closes all subscribers and stops publishing.
func (b *InMemoryBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	for _, ch := range b.rawSubs {
		close(ch)
	}
	for _, ch := range b.screenedSubs {
		close(ch)
	}
	return nil
}
