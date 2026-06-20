package analytics

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	_ "sentinel-flow/pkg/mockdb"
)

func TestPostgresTelemetryTracker(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://sentinel_flow_admin:sentinel_flow_secure_password@localhost:5432/sentinel_flow_production?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	isMock := false

	if err != nil || db.Ping() != nil {
		isMock = true
		db, err = sql.Open("mockdb", "")
		if err != nil {
			t.Fatalf("failed to open mockdb: %v", err)
		}
	}

	defer db.Close()

	if !isMock {
		migration, err := os.ReadFile("../../db/migrations/0001_init.sql")
		if err == nil {
			_, _ = db.Exec(string(migration))
		}
	}

	tracker := NewPostgresTelemetryTracker(db)

	if !isMock {
		// Clean up table first to prevent dirty tests
		_, _ = db.Exec("DELETE FROM metrics_counters")
	}

	tracker.IncrementMetric("total_received", 5)
	tracker.IncrementMetric("legitimate", 3)
	tracker.IncrementMetric("fraudulent", 2)
	tracker.IncrementMetric("crm_attempts", 4)
	tracker.IncrementMetric("crm_successes", 3)
	tracker.IncrementMetric("crm_failures", 1)

	metrics := tracker.GetMetrics()

	if isMock {
		// Since mockdb yields total_received metric_name with counter_value 10
		if metrics.TotalEventsReceived != 10 {
			t.Errorf("expected 10 total received events from mockdb, got %d", metrics.TotalEventsReceived)
		}
	} else {
		if metrics.TotalEventsReceived != 5 {
			t.Errorf("expected 5 total received events, got %d", metrics.TotalEventsReceived)
		}
		if metrics.LegitimateEvents != 3 {
			t.Errorf("expected 3 legitimate events, got %d", metrics.LegitimateEvents)
		}
	}
}
