package broker

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRedisBroker(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	rb, err := NewRedisBroker(redisURL, 100)
	if err != nil {
		t.Skipf("Skipping Redis test: broker not available: %v", err)
	}
	defer rb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Subscribe to raw events
	ch, err := rb.SubscribeRaw(ctx)
	if err != nil {
		t.Fatalf("SubscribeRaw failed: %v", err)
	}

	// Publish raw event
	event := &TrackingEvent{
		EventID:   "test_redis_ev_1",
		UserID:    "test_redis_user",
		EventType: "click",
		Timestamp: time.Now().UTC(),
	}

	// Wait briefly to ensure subscription is established in Redis
	time.Sleep(100 * time.Millisecond)

	err = rb.PublishRaw(ctx, event)
	if err != nil {
		t.Fatalf("PublishRaw failed: %v", err)
	}

	// Read event from channel
	select {
	case received, ok := <-ch:
		if !ok {
			t.Fatal("Channel closed prematurely")
		}
		if received.EventID != event.EventID {
			t.Errorf("Expected event ID %s, got %s", event.EventID, received.EventID)
		}
	case <-ctx.Done():
		t.Fatal("Timed out waiting for raw event from Redis Pub/Sub")
	}
}

func TestRedisBrokerScreened(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	rb, err := NewRedisBroker(redisURL, 100)
	if err != nil {
		t.Skipf("Skipping Redis test: broker not available: %v", err)
	}
	defer rb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := rb.SubscribeScreened(ctx)
	if err != nil {
		t.Fatalf("SubscribeScreened failed: %v", err)
	}

	event := &ScreenedEvent{
		TrackingEvent: TrackingEvent{
			EventID:   "test_redis_screened_1",
			UserID:    "test_redis_user",
			EventType: "click",
			Timestamp: time.Now().UTC(),
		},
		IsFraudulent: false,
		FraudScore:   0.0,
		FraudReason:  "",
		ScreenedAt:   time.Now().UTC(),
	}

	time.Sleep(100 * time.Millisecond)

	err = rb.PublishScreened(ctx, event)
	if err != nil {
		t.Fatalf("PublishScreened failed: %v", err)
	}

	select {
	case received, ok := <-ch:
		if !ok {
			t.Fatal("Channel closed prematurely")
		}
		if received.EventID != event.EventID {
			t.Errorf("Expected event ID %s, got %s", event.EventID, received.EventID)
		}
	case <-ctx.Done():
		t.Fatal("Timed out waiting for screened event from Redis Pub/Sub")
	}
}

func TestRedisBrokerDefaultBuffer(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	rb, err := NewRedisBroker(redisURL, 0)
	if err != nil {
		t.Skip("Redis is not available")
	}
	defer rb.Close()
	
	if rb.bufferSize != 10000 {
		t.Errorf("expected default buffer size 10000, got %d", rb.bufferSize)
	}
}
