package analytics

import (
	"database/sql"
	"log/slog"
	"time"
)

type PostgresTelemetryTracker struct {
	db        *sql.DB
	startTime time.Time
}

func NewPostgresTelemetryTracker(db *sql.DB) *PostgresTelemetryTracker {
	return &PostgresTelemetryTracker{
		db:        db,
		startTime: time.Now(),
	}
}

func (p *PostgresTelemetryTracker) IncrementMetric(name string, delta int64) {
	query := `
	INSERT INTO metrics_counters (metric_name, counter_value)
	VALUES ($1, $2)
	ON CONFLICT (metric_name) DO UPDATE SET
		counter_value = metrics_counters.counter_value + EXCLUDED.counter_value;
	`
	_, err := p.db.Exec(query, name, delta)
	if err != nil {
		slog.Error("[PostgresTelemetryTracker] Failed to increment metric", "metric", name, "error", err)
	}
}

func (p *PostgresTelemetryTracker) GetMetrics() *SystemMetrics {
	elapsed := time.Since(p.startTime).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}

	metrics := &SystemMetrics{}

	rows, err := p.db.Query("SELECT metric_name, counter_value FROM metrics_counters")
	if err != nil {
		slog.Error("[PostgresTelemetryTracker] Failed to fetch metrics counters", "error", err)
		return metrics
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var val int64
		if err := rows.Scan(&name, &val); err != nil {
			slog.Error("[PostgresTelemetryTracker] Error scanning metric row", "error", err)
			continue
		}
		switch name {
		case "total_received":
			metrics.TotalEventsReceived = val
		case "legitimate":
			metrics.LegitimateEvents = val
		case "fraudulent":
			metrics.FraudulentEvents = val
		case "crm_attempts":
			metrics.CRMAttempts = val
		case "crm_successes":
			metrics.CRMSuccesses = val
		case "crm_failures":
			metrics.CRMFailures = val
		}
	}

	metrics.EventsPerSecond = float64(metrics.TotalEventsReceived) / elapsed
	return metrics
}
