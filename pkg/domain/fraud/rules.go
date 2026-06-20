package fraud

import (
	"context"
	"strings"
	"sync"
	"time"

	"sentinel-flow/pkg/broker"
)

// UserGeoHistory holds geographical travel state details for a specific user ID.
type UserGeoHistory struct {
	LastCountry string
	LastTime    time.Time
}

// RuleState aggregates history variables and provides synchronization across rules.
type RuleState struct {
	Mu          sync.Mutex
	IPHistory   map[string][]time.Time
	UserHistory map[string]*UserGeoHistory
}

// NewRuleState initializes an empty, synchronized state manager.
func NewRuleState() *RuleState {
	return &RuleState{
		IPHistory:   make(map[string][]time.Time),
		UserHistory: make(map[string]*UserGeoHistory),
	}
}

// FraudRule defines the strategy contract for evaluating incoming tracking events.
type FraudRule interface {
	Name() string
	Evaluate(ctx context.Context, event *broker.TrackingEvent, state *RuleState) (float64, string, error)
}

// BotRule inspects the User-Agent signature to detect crawlers and automated scraper bots.
type BotRule struct{}

func (b *BotRule) Name() string { return "Bot Detection" }
func (b *BotRule) Evaluate(ctx context.Context, event *broker.TrackingEvent, state *RuleState) (float64, string, error) {
	ua := strings.ToLower(event.UserAgent)
	if ua == "" {
		return 0.4, "empty user-agent", nil
	}
	if strings.Contains(ua, "bot") || strings.Contains(ua, "crawler") ||
		strings.Contains(ua, "headless") || strings.Contains(ua, "spider") ||
		strings.Contains(ua, "python-requests") || strings.Contains(ua, "curl") {
		return 0.8, "known bot/crawler user-agent signature", nil
	}
	return 0.0, "", nil
}

// RateLimitRule blocks high-frequency request spikes originating from a single IP address.
type RateLimitRule struct{}

func (r *RateLimitRule) Name() string { return "IP Rate Limiting" }
func (r *RateLimitRule) Evaluate(ctx context.Context, event *broker.TrackingEvent, state *RuleState) (float64, string, error) {
	state.Mu.Lock()
	defer state.Mu.Unlock()

	now := time.Now()
	history := state.IPHistory[event.IPAddress]

	var activeStamps []time.Time
	for _, t := range history {
		if now.Sub(t) < 5*time.Second {
			activeStamps = append(activeStamps, t)
		}
	}
	activeStamps = append(activeStamps, now)
	state.IPHistory[event.IPAddress] = activeStamps

	if len(activeStamps) > 10 {
		return 0.7, "high-frequency events from single IP (rate limit exceeded)", nil
	}
	return 0.0, "", nil
}

// GeoVelocityRule checks for impossible geographic travel speeds of a user between requests.
type GeoVelocityRule struct{}

func (g *GeoVelocityRule) Name() string { return "Geo-Velocity Anomaly" }
func (g *GeoVelocityRule) Evaluate(ctx context.Context, event *broker.TrackingEvent, state *RuleState) (float64, string, error) {
	countryVal, ok := event.Payload["country"]
	if !ok {
		return 0.0, "", nil
	}
	country, okStr := countryVal.(string)
	if !okStr || country == "" {
		return 0.0, "", nil
	}

	state.Mu.Lock()
	defer state.Mu.Unlock()

	now := time.Now()
	lastGeo, exists := state.UserHistory[event.UserID]
	if exists {
		if lastGeo.LastCountry != country && now.Sub(lastGeo.LastTime) < 10*time.Second {
			return 0.9, "geo-velocity anomaly: user IP location changed too fast", nil
		}
	}

	state.UserHistory[event.UserID] = &UserGeoHistory{
		LastCountry: country,
		LastTime:    now,
	}
	return 0.0, "", nil
}
