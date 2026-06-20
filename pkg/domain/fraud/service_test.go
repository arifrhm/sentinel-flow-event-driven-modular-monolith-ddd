package fraud

import (
	"context"
	"testing"
	"time"

	"sentinel-flow/pkg/broker"
)

type mockFraudTelemetryTracker struct {
	metrics map[string]int64
}

func (m *mockFraudTelemetryTracker) IncrementMetric(name string, delta int64) {
	if m.metrics == nil {
		m.metrics = make(map[string]int64)
	}
	m.metrics[name] += delta
}

func TestFraudServiceScreenEvent(t *testing.T) {
	b := broker.NewInMemoryBroker(10)
	defer b.Close()

	repo := NewMemoryFraudRepository(nil)
	tracker := &mockFraudTelemetryTracker{}
	service := NewFraudService(b, repo, tracker)

	// Clean event
	cleanEv := &broker.TrackingEvent{
		EventID:   "clean_1",
		UserID:    "usr_1",
		UserAgent: "Mozilla/5.0",
		IPAddress: "192.168.1.1",
	}

	service.ScreenEvent(context.Background(), cleanEv)

	res, err := repo.GetEventsByUserID("usr_1")
	if err != nil || len(res) != 1 {
		t.Fatalf("failed to retrieve event: %v", err)
	}
	if res[0].IsFraudulent {
		t.Error("expected clean event, screened as fraudulent")
	}

	// Fraud bot event
	fraudEv := &broker.TrackingEvent{
		EventID:   "fraud_1",
		UserID:    "bot_1",
		UserAgent: "Googlebot",
		IPAddress: "8.8.8.8",
	}

	service.ScreenEvent(context.Background(), fraudEv)

	resFraud, _ := repo.GetEventsByUserID("bot_1")
	if len(resFraud) != 1 || !resFraud[0].IsFraudulent {
		t.Error("expected event to be flagged as fraudulent")
	}
}

func TestFraudServiceDaemon(t *testing.T) {
	b := broker.NewInMemoryBroker(10)
	defer b.Close()

	repo := NewMemoryFraudRepository(nil)
	tracker := &mockFraudTelemetryTracker{}
	service := NewFraudService(b, repo, tracker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = service.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Publish an event
	evt := &broker.TrackingEvent{
		EventID:   "evt_daemon_ok",
		UserID:    "usr_daemon",
		EventType: "signup",
	}

	_ = b.PublishRaw(ctx, evt)
	time.Sleep(50 * time.Millisecond)

	events, _ := repo.GetEventsByUserID("usr_daemon")
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	// Test Get/Delete operations via service
	got, _ := service.GetEventsByUserID("usr_daemon")
	if len(got) != 1 {
		t.Error("GetEventsByUserID failed")
	}

	deleted, _ := service.DeleteUserEvents("usr_daemon")
	if deleted != 1 {
		t.Error("DeleteUserEvents failed")
	}
}

func TestMemoryFraudRepositoryEdgeCases(t *testing.T) {
	repo := NewMemoryFraudRepository(nil)
	evs, err := repo.GetEventsByUserID("non_existent")
	if err != nil || len(evs) != 0 {
		t.Errorf("expected empty results for non-existent user, got: %v", evs)
	}

	count, err := repo.DeleteUserEvents("non_existent")
	if err != nil || count != 0 {
		t.Errorf("expected 0 deleted events, got %d", count)
	}

	// Test with crmDeleter
	called := false
	repoWithDeleter := NewMemoryFraudRepository(func(userID string) {
		called = true
	})
	cnt, err := repoWithDeleter.DeleteUserEvents("some_user")
	if err != nil || cnt != 1 || !called {
		t.Errorf("expected deleter to be called, got count=%d, called=%t", cnt, called)
	}
}

func TestFraudServiceHighFraudScore(t *testing.T) {
	b := broker.NewInMemoryBroker(10)
	defer b.Close()

	repo := NewMemoryFraudRepository(nil)
	tracker := &mockFraudTelemetryTracker{}
	service := NewFraudService(b, repo, tracker)

	// Trigger bot UA (0.8) and rate limit rule (0.7) by hitting 11 times
	ip := "9.9.9.9"
	for i := 0; i < 11; i++ {
		evt := &broker.TrackingEvent{
			EventID:   "ev_bot_rate_limit",
			UserID:    "bot_spammer",
			UserAgent: "Googlebot", // Bot detection rule
			IPAddress: ip,          // IP rate limit rule
		}
		service.ScreenEvent(context.Background(), evt)
	}

	res, _ := repo.GetEventsByUserID("bot_spammer")
	if len(res) == 0 {
		t.Fatal("expected events saved")
	}

	lastScore := res[len(res)-1].FraudScore
	if lastScore != 1.0 {
		t.Errorf("expected fraud score capped at 1.0, got %f", lastScore)
	}
}
