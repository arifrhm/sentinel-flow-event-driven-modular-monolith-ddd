package analytics

// TelemetryTracker defines the port interface for recording and retrieving telemetry metrics.
type TelemetryTracker interface {
	IncrementMetric(name string, delta int64)
	GetMetrics() *SystemMetrics
}
