package broker

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryBrokerRawFlow(t *testing.T) {
	b := NewInMemoryBroker(0) // test <= 0 buffer size fallback
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	ch, err := b.SubscribeRaw(ctx)
	if err != nil {
		t.Fatalf("failed to subscribe raw: %v", err)
	}

	event := &TrackingEvent{
		EventID: "evt_raw_test",
		UserID:  "usr_test",
	}

	if err := b.PublishRaw(ctx, event); err != nil {
		t.Fatalf("failed to publish raw: %v", err)
	}

	select {
	case received := <-ch:
		if received.EventID != "evt_raw_test" {
			t.Errorf("expected event ID evt_raw_test, got %s", received.EventID)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for raw event")
	}
}

func TestInMemoryBrokerScreenedFlow(t *testing.T) {
	b := NewInMemoryBroker(10)
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	ch, err := b.SubscribeScreened(ctx)
	if err != nil {
		t.Fatalf("failed to subscribe screened: %v", err)
	}

	event := &ScreenedEvent{
		TrackingEvent: TrackingEvent{
			EventID: "evt_screened_test",
		},
		IsFraudulent: true,
	}

	if err := b.PublishScreened(ctx, event); err != nil {
		t.Fatalf("failed to publish screened: %v", err)
	}

	select {
	case received := <-ch:
		if received.EventID != "evt_screened_test" || !received.IsFraudulent {
			t.Errorf("unexpected event received: %+v", received)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for screened event")
	}
}

func TestInMemoryBrokerClosed(t *testing.T) {
	b := NewInMemoryBroker(1)
	_ = b.Close()
	_ = b.Close() // test double close

	ctx := context.Background()

	if err := b.PublishRaw(ctx, &TrackingEvent{}); err == nil {
		t.Error("expected error publishing raw to closed broker")
	}

	if _, err := b.SubscribeRaw(ctx); err == nil {
		t.Error("expected error subscribing raw on closed broker")
	}

	if err := b.PublishScreened(ctx, &ScreenedEvent{}); err == nil {
		t.Error("expected error publishing screened to closed broker")
	}

	if _, err := b.SubscribeScreened(ctx); err == nil {
		t.Error("expected error subscribing screened on closed broker")
	}
}

func TestInMemoryBrokerPublishContextCancel(t *testing.T) {
	b := NewInMemoryBroker(1)
	defer b.Close()

	ctx, cancel := context.WithCancel(context.Background())
	_, _ = b.SubscribeRaw(ctx)

	// Fill buffer so sub <- event blocks
	_ = b.PublishRaw(ctx, &TrackingEvent{})

	// cancel context
	cancel()

	// Publish in a loop to ensure select chooses <-ctx.Done() case
	for i := 0; i < 100; i++ {
		if err := b.PublishRaw(ctx, &TrackingEvent{}); err != nil {
			break
		}
	}

	ctxScreened, cancelScreened := context.WithCancel(context.Background())
	_, _ = b.SubscribeScreened(ctxScreened)
	_ = b.PublishScreened(ctxScreened, &ScreenedEvent{})
	cancelScreened()

	for i := 0; i < 100; i++ {
		if err := b.PublishScreened(ctxScreened, &ScreenedEvent{}); err != nil {
			break
		}
	}
}

func TestInMemoryBrokerBufferFull(t *testing.T) {
	b := NewInMemoryBroker(1)
	defer b.Close()

	ctx := context.Background()
	_, _ = b.SubscribeRaw(ctx)

	// Publish twice to exceed buffer (buffer size is 1)
	_ = b.PublishRaw(ctx, &TrackingEvent{EventID: "1"})
	_ = b.PublishRaw(ctx, &TrackingEvent{EventID: "2"}) // should drop on select default

	_, _ = b.SubscribeScreened(ctx)
	_ = b.PublishScreened(ctx, &ScreenedEvent{})
	_ = b.PublishScreened(ctx, &ScreenedEvent{}) // should drop
}
