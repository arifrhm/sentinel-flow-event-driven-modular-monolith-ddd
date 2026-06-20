package fraud

import (
	"context"
	"testing"

	"sentinel-flow/pkg/broker"
)

func TestBotRule(t *testing.T) {
	rule := &BotRule{}
	ctx := context.Background()
	state := NewRuleState()

	if rule.Name() != "Bot Detection" {
		t.Errorf("unexpected name: %s", rule.Name())
	}

	// 1. Legitimate UA
	score, reason, _ := rule.Evaluate(ctx, &broker.TrackingEvent{UserAgent: "Mozilla/5.0"}, state)
	if score != 0 {
		t.Errorf("expected 0, got %f (%s)", score, reason)
	}

	// 2. Headless Chrome UA
	score, reason, _ = rule.Evaluate(ctx, &broker.TrackingEvent{UserAgent: "HeadlessChrome/109.0.0.0 Safari/537.36"}, state)
	if score != 0.8 {
		t.Errorf("expected 0.8, got %f (%s)", score, reason)
	}

	// 3. Empty UA
	score, reason, _ = rule.Evaluate(ctx, &broker.TrackingEvent{UserAgent: ""}, state)
	if score != 0.4 {
		t.Errorf("expected 0.4, got %f (%s)", score, reason)
	}
}

func TestRateLimitRule(t *testing.T) {
	rule := &RateLimitRule{}
	ctx := context.Background()
	state := NewRuleState()

	if rule.Name() != "IP Rate Limiting" {
		t.Errorf("unexpected name: %s", rule.Name())
	}

	ip := "1.2.3.4"
	// Under 10 hits in window
	for i := 0; i < 10; i++ {
		score, _, _ := rule.Evaluate(ctx, &broker.TrackingEvent{IPAddress: ip}, state)
		if score != 0 {
			t.Errorf("expected 0 on hit %d, got %f", i, score)
		}
	}

	// Over 10 hits
	for i := 0; i < 3; i++ {
		score, reason, _ := rule.Evaluate(ctx, &broker.TrackingEvent{IPAddress: ip}, state)
		if score != 0.7 {
			t.Errorf("expected 0.7 on hit %d, got %f (%s)", i, score, reason)
		}
	}
}

func TestGeoVelocityRule(t *testing.T) {
	rule := &GeoVelocityRule{}
	ctx := context.Background()
	state := NewRuleState()

	if rule.Name() != "Geo-Velocity Anomaly" {
		t.Errorf("unexpected name: %s", rule.Name())
	}

	// 1. Missing country payload
	score, _, _ := rule.Evaluate(ctx, &broker.TrackingEvent{UserID: "usr_1"}, state)
	if score != 0 {
		t.Errorf("expected 0, got %f", score)
	}

	// 2. Non-string country type
	score, _, _ = rule.Evaluate(ctx, &broker.TrackingEvent{
		UserID:  "usr_1",
		Payload: map[string]interface{}{"country": 123},
	}, state)
	if score != 0 {
		t.Errorf("expected 0, got %f", score)
	}

	// 3. Empty country string
	score, _, _ = rule.Evaluate(ctx, &broker.TrackingEvent{
		UserID:  "usr_1",
		Payload: map[string]interface{}{"country": ""},
	}, state)
	if score != 0 {
		t.Errorf("expected 0, got %f", score)
	}

	// 4. Normal trip (first time travel entry - exists is false)
	score, _, _ = rule.Evaluate(ctx, &broker.TrackingEvent{
		UserID:  "usr_1",
		Payload: map[string]interface{}{"country": "ID"},
	}, state)
	if score != 0 {
		t.Errorf("expected 0, got %f", score)
	}

	// 5. Normal trip (same country)
	score, _, _ = rule.Evaluate(ctx, &broker.TrackingEvent{
		UserID:  "usr_1",
		Payload: map[string]interface{}{"country": "ID"},
	}, state)
	if score != 0 {
		t.Errorf("expected 0, got %f", score)
	}

	// 6. Fast travel (impossible speed)
	score, reason, _ := rule.Evaluate(ctx, &broker.TrackingEvent{
		UserID:  "usr_1",
		Payload: map[string]interface{}{"country": "JP"},
	}, state)
	if score != 0.9 {
		t.Errorf("expected 0.9 for impossible travel speed, got %f (%s)", score, reason)
	}
}
