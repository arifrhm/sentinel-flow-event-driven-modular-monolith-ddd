package broker

import "time"

// TrackingEvent is an integration event representing a raw telemetry event from client SDKs.
type TrackingEvent struct {
	EventID     string                 `json:"event_id"`
	UserID      string                 `json:"user_id"`
	EventType   string                 `json:"event_type"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	Payload     map[string]interface{} `json:"payload"`
	Timestamp   time.Time              `json:"timestamp"`
	GDPRConsent bool                   `json:"gdpr_consent"`
}

// ScreenedEvent is an integration event containing the result of fraud screening.
type ScreenedEvent struct {
	TrackingEvent
	IsFraudulent bool      `json:"is_fraudulent"`
	FraudScore   float64   `json:"fraud_score"`
	FraudReason  string    `json:"fraud_reason"`
	ScreenedAt   time.Time `json:"screened_at"`
}
