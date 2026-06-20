package broker

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewRedisBrokerInvalidURL(t *testing.T) {
	_, err := NewRedisBroker("::invalid-url::", 10)
	if err == nil {
		t.Error("expected error parsing invalid URL, got nil")
	}
}

func TestRedisBrokerClosedErrors(t *testing.T) {
	rb := &RedisBroker{closed: true}
	ctx := context.Background()

	if err := rb.PublishRaw(ctx, nil); err == nil || err.Error() != "broker is closed" {
		t.Errorf("expected broker is closed error, got %v", err)
	}

	if _, err := rb.SubscribeRaw(ctx); err == nil || err.Error() != "broker is closed" {
		t.Errorf("expected broker is closed error, got %v", err)
	}

	if err := rb.PublishScreened(ctx, nil); err == nil || err.Error() != "broker is closed" {
		t.Errorf("expected broker is closed error, got %v", err)
	}

	if _, err := rb.SubscribeScreened(ctx); err == nil || err.Error() != "broker is closed" {
		t.Errorf("expected broker is closed error, got %v", err)
	}
}

func TestRedisBrokerInvalidJSONPayload(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	rb, err := NewRedisBroker(redisURL, 10)
	if err != nil {
		t.Skip("Redis is not available")
	}
	defer rb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ch, err := rb.SubscribeRaw(ctx)
	if err != nil {
		t.Fatalf("SubscribeRaw failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	opts, _ := redis.ParseURL(redisURL)
	rawClient := redis.NewClient(opts)
	defer rawClient.Close()

	err = rawClient.Publish(ctx, "sentinel_flow_raw", "invalid payload json").Err()
	if err != nil {
		t.Fatalf("failed to publish invalid json: %v", err)
	}

	select {
	case ev, ok := <-ch:
		if ok {
			t.Errorf("unexpected event received: %+v", ev)
		}
	case <-time.After(100 * time.Millisecond):
	}
	rb.Close()
	time.Sleep(10 * time.Millisecond)
}

func TestRedisBrokerInvalidJSONScreened(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	rb, err := NewRedisBroker(redisURL, 10)
	if err != nil {
		t.Skip("Redis is not available")
	}
	defer rb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ch, err := rb.SubscribeScreened(ctx)
	if err != nil {
		t.Fatalf("SubscribeScreened failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	opts, _ := redis.ParseURL(redisURL)
	rawClient := redis.NewClient(opts)
	defer rawClient.Close()

	_ = rawClient.Publish(ctx, "sentinel_flow_screened", "invalid payload").Err()

	select {
	case ev, ok := <-ch:
		if ok {
			t.Errorf("unexpected event received: %+v", ev)
		}
	case <-time.After(100 * time.Millisecond):
	}
	rb.Close()
	time.Sleep(10 * time.Millisecond)
}
