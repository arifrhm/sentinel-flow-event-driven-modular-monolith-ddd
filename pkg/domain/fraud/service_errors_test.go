package fraud

import (
	"context"
	"errors"
	"testing"
	"time"

	"sentinel-flow/pkg/broker"
)

type mockErrBroker struct {
	broker.Broker
	subErr bool
	pubErr bool
}

func (m *mockErrBroker) SubscribeRaw(ctx context.Context) (<-chan *broker.TrackingEvent, error) {
	if m.subErr {
		return nil, errors.New("mock subscribe error")
	}
	return m.Broker.SubscribeRaw(ctx)
}

func (m *mockErrBroker) PublishScreened(ctx context.Context, event *broker.ScreenedEvent) error {
	if m.pubErr {
		return errors.New("mock publish error")
	}
	return m.Broker.PublishScreened(ctx, event)
}

type mockErrRepo struct {
	FraudRepository
	saveErr bool
}

func (m *mockErrRepo) SaveEvent(event *broker.ScreenedEvent) error {
	if m.saveErr {
		return errors.New("mock save error")
	}
	return m.FraudRepository.SaveEvent(event)
}

func TestFraudServiceErrors(t *testing.T) {
	// 1. SubscribeRaw error
	b := &mockErrBroker{subErr: true}
	service := NewFraudService(b, nil, nil)
	err := service.Start(context.Background())
	if err == nil || err.Error() != "mock subscribe error" {
		t.Errorf("expected mock subscribe error, got %v", err)
	}

	// 2. SaveEvent DB failure and PublishScreened broker failure
	baseBroker := broker.NewInMemoryBroker(1)
	defer baseBroker.Close()
	b2 := &mockErrBroker{Broker: baseBroker, pubErr: true}
	repo := &mockErrRepo{FraudRepository: NewMemoryFraudRepository(nil), saveErr: true}
	tracker := &mockFraudTelemetryTracker{}
	
	service2 := NewFraudService(b2, repo, tracker)
	
	evt := &broker.TrackingEvent{
		EventID: "ev_srv_err",
		UserID:  "usr_srv_err",
	}
	
	service2.ScreenEvent(context.Background(), evt)

	// 3. Test rule evaluation error inside ScreenEvent
	service2.Rules = []FraudRule{&mockErrRule{}}
	service2.ScreenEvent(context.Background(), evt)

	// 4. Test Broker channel closed inside Start
	b3 := broker.NewInMemoryBroker(1)
	service3 := NewFraudService(b3, repo, tracker)
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel3()

	errChan := make(chan error, 1)
	go func() {
		errChan <- service3.Start(ctx3)
	}()

	time.Sleep(50 * time.Millisecond)
	_ = b3.Close()

	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("expected nil error on broker close, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for Start to exit on broker close")
	}
}

type mockErrRule struct{}

func (m *mockErrRule) Name() string { return "mock_error_rule" }
func (m *mockErrRule) Evaluate(ctx context.Context, event *broker.TrackingEvent, state *RuleState) (float64, string, error) {
	return 0.0, "", errors.New("rule eval error")
}
