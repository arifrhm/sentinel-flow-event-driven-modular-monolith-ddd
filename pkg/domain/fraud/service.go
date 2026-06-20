package fraud

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"sentinel-flow/pkg/broker"
)

// TelemetryTracker defines the port interface for updating fraud analysis statistics.
type TelemetryTracker interface {
	IncrementMetric(name string, delta int64)
}

type FraudService struct {
	broker  broker.Broker
	repo    FraudRepository
	tracker TelemetryTracker
	state   *RuleState
	Rules   []FraudRule
}

func NewFraudService(b broker.Broker, r FraudRepository, t TelemetryTracker) *FraudService {
	return &FraudService{
		broker:  b,
		repo:    r,
		tracker: t,
		state:   NewRuleState(),
		Rules: []FraudRule{
			&BotRule{},
			&RateLimitRule{},
			&GeoVelocityRule{},
		},
	}
}

func (s *FraudService) Start(ctx context.Context) error {
	ch, err := s.broker.SubscribeRaw(ctx)
	if err != nil {
		return err
	}

	slog.Info("[Fraud] Worker listening for raw events...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			s.ScreenEvent(ctx, event)
		}
	}
}

func (s *FraudService) ScreenEvent(ctx context.Context, event *broker.TrackingEvent) {
	var fraudScore float64
	var reasons []string

	for _, rule := range s.Rules {
		score, reason, err := rule.Evaluate(ctx, event, s.state)
		if err != nil {
			slog.Error("Failed to evaluate rule", "rule", rule.Name(), "error", err)
			continue
		}
		if score > 0 {
			fraudScore += score
			reasons = append(reasons, reason)
		}
	}

	isFraud := fraudScore >= 0.5
	if fraudScore > 1.0 {
		fraudScore = 1.0
	}

	reasonStr := strings.Join(reasons, ", ")
	screened := &broker.ScreenedEvent{
		TrackingEvent: *event,
		IsFraudulent:  isFraud,
		FraudScore:    fraudScore,
		FraudReason:   reasonStr,
		ScreenedAt:    time.Now(),
	}

	if err := s.repo.SaveEvent(screened); err != nil {
		slog.Error("[Fraud] Error saving event to DB", "error", err)
	}

	if isFraud {
		s.tracker.IncrementMetric("fraudulent", 1)
		slog.Warn("[Fraud] ALERT: Fraud detected", "user_id", event.UserID, "score", fraudScore, "reasons", reasonStr)
	} else {
		s.tracker.IncrementMetric("legitimate", 1)
	}

	if err := s.broker.PublishScreened(ctx, screened); err != nil {
		slog.Error("[Fraud] Error publishing screened event", "error", err)
	}
}

func (s *FraudService) GetEventsByUserID(userID string) ([]*broker.ScreenedEvent, error) {
	return s.repo.GetEventsByUserID(userID)
}

func (s *FraudService) DeleteUserEvents(userID string) (int, error) {
	return s.repo.DeleteUserEvents(userID)
}
