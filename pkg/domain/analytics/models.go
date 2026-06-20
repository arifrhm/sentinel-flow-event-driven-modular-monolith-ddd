package analytics

// SystemMetrics represents aggregated system performance and operations telemetry.
type SystemMetrics struct {
	TotalEventsReceived int64   `json:"total_events_received"`
	LegitimateEvents    int64   `json:"legitimate_events"`
	FraudulentEvents    int64   `json:"fraudulent_events"`
	CRMAttempts         int64   `json:"crm_attempts"`
	CRMSuccesses        int64   `json:"crm_successes"`
	CRMFailures         int64   `json:"crm_failures"`
	EventsPerSecond     float64 `json:"events_per_second"`
}
