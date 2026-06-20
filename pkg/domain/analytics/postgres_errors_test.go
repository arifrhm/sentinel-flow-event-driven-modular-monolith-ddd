package analytics

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"sentinel-flow/pkg/mockdb"
)

func TestPostgresTelemetryTrackerErrors(t *testing.T) {
	db, err := sql.Open("mockdb", "")
	if err != nil {
		t.Fatalf("failed to open mockdb: %v", err)
	}
	defer db.Close()

	tracker := NewPostgresTelemetryTracker(db)

	// 1. Test IncrementMetric execution error
	mockdb.SimError = errors.New("exec error")
	tracker.IncrementMetric("total_received", 1) // should log error and return
	mockdb.SimError = nil

	// 2. Test GetMetrics query execution error
	mockdb.SimError = errors.New("query error")
	metrics := tracker.GetMetrics()
	if metrics == nil {
		t.Error("expected non-nil metrics even on error")
	}
	mockdb.SimError = nil

	// 3. Test GetMetrics scan error
	mockdb.TriggerScanError = true
	metricsScan := tracker.GetMetrics()
	if metricsScan == nil {
		t.Error("expected non-nil metrics even on scan error")
	}
	mockdb.TriggerScanError = false

	// 4. Test GetMetrics elapsed <= 0
	tracker.startTime = time.Now().Add(10 * time.Second)
	metricsFuture := tracker.GetMetrics()
	if metricsFuture == nil {
		t.Error("expected metrics on future start time")
	}

	// 5. Test GetMetrics positive elapsed
	tracker.startTime = time.Now().Add(-10 * time.Second)
	metricsPast := tracker.GetMetrics()
	if metricsPast == nil {
		t.Error("expected metrics on past start time")
	}
}
