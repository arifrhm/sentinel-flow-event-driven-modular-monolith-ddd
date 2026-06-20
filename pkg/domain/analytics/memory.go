package analytics

import (
	"sync/atomic"
	"time"
)

type MemoryTelemetryTracker struct {
	startTime     time.Time
	totalReceived int64
	legitimate    int64
	fraudulent    int64
	crmAttempts   int64
	crmSuccesses  int64
	crmFailures   int64
}

func NewMemoryTelemetryTracker() *MemoryTelemetryTracker {
	return &MemoryTelemetryTracker{
		startTime: time.Now(),
	}
}

func (m *MemoryTelemetryTracker) IncrementMetric(name string, delta int64) {
	switch name {
	case "total_received":
		atomic.AddInt64(&m.totalReceived, delta)
	case "legitimate":
		atomic.AddInt64(&m.legitimate, delta)
	case "fraudulent":
		atomic.AddInt64(&m.fraudulent, delta)
	case "crm_attempts":
		atomic.AddInt64(&m.crmAttempts, delta)
	case "crm_successes":
		atomic.AddInt64(&m.crmSuccesses, delta)
	case "crm_failures":
		atomic.AddInt64(&m.crmFailures, delta)
	}
}

func (m *MemoryTelemetryTracker) GetMetrics() *SystemMetrics {
	elapsed := time.Since(m.startTime).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}

	totalReceived := atomic.LoadInt64(&m.totalReceived)

	return &SystemMetrics{
		TotalEventsReceived: totalReceived,
		LegitimateEvents:    atomic.LoadInt64(&m.legitimate),
		FraudulentEvents:    atomic.LoadInt64(&m.fraudulent),
		CRMAttempts:         atomic.LoadInt64(&m.crmAttempts),
		CRMSuccesses:        atomic.LoadInt64(&m.crmSuccesses),
		CRMFailures:         atomic.LoadInt64(&m.crmFailures),
		EventsPerSecond:     float64(totalReceived) / elapsed,
	}
}
