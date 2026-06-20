package analytics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAnalyticsService(t *testing.T) {
	tracker := NewMemoryTelemetryTracker()
	tracker.IncrementMetric("total_received", 10)
	tracker.IncrementMetric("legitimate", 8)
	tracker.IncrementMetric("fraudulent", 2)
	tracker.IncrementMetric("crm_attempts", 5)
	tracker.IncrementMetric("crm_successes", 4)
	tracker.IncrementMetric("crm_failures", 1)

	service := NewAnalyticsService(tracker)

	// Setup routes test
	mux := http.NewServeMux()
	service.SetupRoutes(mux)

	// 1. HandleMetrics JSON Endpoint
	req := httptest.NewRequest(http.MethodGet, "/v1/metrics", nil)
	rr := httptest.NewRecorder()
	service.HandleMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected StatusOK, got %d", rr.Code)
	}

	var m SystemMetrics
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("failed to unmarshal metrics: %v", err)
	}

	if m.TotalEventsReceived != 10 || m.LegitimateEvents != 8 || m.FraudulentEvents != 2 {
		t.Errorf("unexpected metrics returned: %+v", m)
	}

	// Bad Method for Metrics
	reqBad := httptest.NewRequest(http.MethodPost, "/v1/metrics", nil)
	rrBad := httptest.NewRecorder()
	service.HandleMetrics(rrBad, reqBad)
	if rrBad.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected StatusMethodNotAllowed, got %d", rrBad.Code)
	}

	// 2. HandleDashboard Text Endpoint
	reqDash := httptest.NewRequest(http.MethodGet, "/v1/dashboard", nil)
	rrDash := httptest.NewRecorder()
	service.HandleDashboard(rrDash, reqDash)

	if rrDash.Code != http.StatusOK {
		t.Fatalf("expected StatusOK, got %d", rrDash.Code)
	}

	bodyStr := rrDash.Body.String()
	if !strings.Contains(bodyStr, "Total Events Ingested:      10 events") {
		t.Errorf("dashboard output missing total events: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Fraud Incidence Rate:       20.00%") {
		t.Errorf("dashboard output missing fraud incidence rate: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "CRM Sync Resilience Rate:   80.00%") {
		t.Errorf("dashboard output missing CRM resilience rate: %s", bodyStr)
	}

	// Bad Method for Dashboard
	reqDashBad := httptest.NewRequest(http.MethodPost, "/v1/dashboard", nil)
	rrDashBad := httptest.NewRecorder()
	service.HandleDashboard(rrDashBad, reqDashBad)
	if rrDashBad.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected StatusMethodNotAllowed, got %d", rrDashBad.Code)
	}
}

func TestMemoryTelemetryTrackerElapsed(t *testing.T) {
	tracker := NewMemoryTelemetryTracker()
	// Set start time to future to force elapsed <= 0 branch
	tracker.startTime = time.Now().Add(10 * time.Second)
	metrics := tracker.GetMetrics()
	if metrics.EventsPerSecond != 0 {
		t.Errorf("expected 0 events per second, got %f", metrics.EventsPerSecond)
	}
}
