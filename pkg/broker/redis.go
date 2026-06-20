package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	PingRetries = 10
	PingDelay   = 1 * time.Second
)

// RedisBroker implements Broker interface using Redis Pub/Sub.
type RedisBroker struct {
	client     *redis.Client
	mu         sync.Mutex
	pubsubs    []*redis.PubSub
	closed     bool
	bufferSize int
}

// NewRedisBroker instantiates a new connection to Redis and ping-checks it.
func NewRedisBroker(redisURL string, bufferSize int) (*RedisBroker, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Ping with retry to tolerate transient boot latency
	var pingErr error
	for attempt := 0; attempt < PingRetries; attempt++ {
		pingErr = client.Ping(context.Background()).Err()
		if pingErr == nil {
			break
		}
		log.Printf("[Redis] Waiting for Redis connection... (attempt %d/%d)", attempt+1, PingRetries)
		time.Sleep(PingDelay)
	}
	if pingErr != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", pingErr)
	}

	if bufferSize <= 0 {
		bufferSize = 10000
	}

	log.Println("[Redis] Broker connection established successfully.")
	return &RedisBroker{
		client:     client,
		bufferSize: bufferSize,
		pubsubs:    make([]*redis.PubSub, 0),
	}, nil
}

// PublishRaw serializes and publishes a raw tracking event.
func (r *RedisBroker) PublishRaw(ctx context.Context, event *TrackingEvent) error {
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return fmt.Errorf("broker is closed")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal raw event: %w", err)
	}

	return r.client.Publish(ctx, "sentinel_flow_raw", data).Err()
}

// SubscribeRaw subscribes to raw events, forwarding received records to a Go channel.
func (r *RedisBroker) SubscribeRaw(ctx context.Context) (<-chan *TrackingEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, fmt.Errorf("broker is closed")
	}

	// We use the background context here to prevent prematurely closing the subscription if the HTTP request context finishes
	pubsub := r.client.Subscribe(context.Background(), "sentinel_flow_raw")
	r.pubsubs = append(r.pubsubs, pubsub)

	ch := make(chan *TrackingEvent, r.bufferSize)
	go func() {
		defer pubsub.Close()
		defer close(ch)

		redisCh := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-redisCh:
				if !ok {
					return
				}
				var ev TrackingEvent
				if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
					log.Printf("[RedisBroker] Error unmarshalling raw event: %v", err)
					continue
				}
				select {
				case ch <- &ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// PublishScreened serializes and publishes a fraud-screened event.
func (r *RedisBroker) PublishScreened(ctx context.Context, event *ScreenedEvent) error {
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return fmt.Errorf("broker is closed")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal screened event: %w", err)
	}

	return r.client.Publish(ctx, "sentinel_flow_screened", data).Err()
}

// SubscribeScreened subscribes to screened events, forwarding received records to a Go channel.
func (r *RedisBroker) SubscribeScreened(ctx context.Context) (<-chan *ScreenedEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, fmt.Errorf("broker is closed")
	}

	pubsub := r.client.Subscribe(context.Background(), "sentinel_flow_screened")
	r.pubsubs = append(r.pubsubs, pubsub)

	ch := make(chan *ScreenedEvent, r.bufferSize)
	go func() {
		defer pubsub.Close()
		defer close(ch)

		redisCh := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-redisCh:
				if !ok {
					return
				}
				var ev ScreenedEvent
				if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
					log.Printf("[RedisBroker] Error unmarshalling screened event: %v", err)
					continue
				}
				select {
				case ch <- &ev:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// Close unsubscribes and closes the redis connection client.
func (r *RedisBroker) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true

	for _, ps := range r.pubsubs {
		ps.Close()
	}
	return r.client.Close()
}
