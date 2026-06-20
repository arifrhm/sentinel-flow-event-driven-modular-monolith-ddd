package broker

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRedisBrokerPingExhaust(t *testing.T) {
	oldRetries := PingRetries
	oldDelay := PingDelay
	PingRetries = 2
	PingDelay = 1 * time.Millisecond
	defer func() {
		PingRetries = oldRetries
		PingDelay = oldDelay
	}()

	_, err := NewRedisBroker("redis://localhost:9999", 10)
	if err == nil {
		t.Error("expected error from NewRedisBroker due to offline server ping exhaustion, got nil")
	}
}

func TestRedisBrokerPublishClientError(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	rb, err := NewRedisBroker(redisURL, 10)
	if err != nil {
		t.Skip("Redis is not available")
	}

	_ = rb.client.Close()

	ctx := context.Background()
	err = rb.PublishRaw(ctx, &TrackingEvent{EventID: "1"})
	if err == nil {
		t.Error("expected raw publish error, got nil")
	}

	err = rb.PublishScreened(ctx, &ScreenedEvent{})
	if err == nil {
		t.Error("expected screened publish error, got nil")
	}
}

func TestRedisBrokerSubscribeRawBufferFullCancel(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	rb, err := NewRedisBroker(redisURL, 1)
	if err != nil {
		t.Skip("Redis is not available")
	}
	defer rb.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := rb.SubscribeRaw(ctx)
	if err != nil {
		t.Fatalf("SubscribeRaw failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	_ = rb.PublishRaw(context.Background(), &TrackingEvent{EventID: "1"})
	_ = rb.PublishRaw(context.Background(), &TrackingEvent{EventID: "2"})
	time.Sleep(50 * time.Millisecond)

	cancel()
	time.Sleep(50 * time.Millisecond)
	for range ch {
	}
}

func TestRedisBrokerSubscribeScreenedBufferFullCancel(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	rb, err := NewRedisBroker(redisURL, 1)
	if err != nil {
		t.Skip("Redis is not available")
	}
	defer rb.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := rb.SubscribeScreened(ctx)
	if err != nil {
		t.Fatalf("SubscribeScreened failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	_ = rb.PublishScreened(context.Background(), &ScreenedEvent{})
	_ = rb.PublishScreened(context.Background(), &ScreenedEvent{})
	time.Sleep(50 * time.Millisecond)

	cancel()
	time.Sleep(50 * time.Millisecond)
	for range ch {
	}
}

func TestRedisBrokerPublishMarshalError(t *testing.T) {
	rb := &RedisBroker{}
	badEvent := &TrackingEvent{
		Payload: map[string]interface{}{"ch": make(chan int)},
	}
	err := rb.PublishRaw(context.Background(), badEvent)
	if err == nil {
		t.Error("expected PublishRaw marshal error, got nil")
	}

	badScreened := &ScreenedEvent{
		TrackingEvent: TrackingEvent{
			Payload: map[string]interface{}{"ch": make(chan int)},
		},
	}
	err = rb.PublishScreened(context.Background(), badScreened)
	if err == nil {
		t.Error("expected PublishScreened marshal error, got nil")
	}
}
