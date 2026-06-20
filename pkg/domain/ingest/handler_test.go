package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"sentinel-flow/pkg/broker"
)

type mockTelemetryTracker struct {
	metrics map[string]int64
}

func (m *mockTelemetryTracker) IncrementMetric(name string, delta int64) {
	if m.metrics == nil {
		m.metrics = make(map[string]int64)
	}
	m.metrics[name] += delta
}

type mockFailedBroker struct {
	broker.Broker
}

func (m *mockFailedBroker) PublishRaw(ctx context.Context, event *broker.TrackingEvent) error {
	return errors.New("mock publish error")
}

func TestHandleIngest(t *testing.T) {
	b := broker.NewInMemoryBroker(10)
	defer b.Close()

	tracker := &mockTelemetryTracker{}
	server := NewIngestServer(b, tracker)

	// 1. Invalid method
	req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	rr := httptest.NewRecorder()
	server.HandleIngest(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}

	// 2. Invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBufferString("invalid json"))
	rr = httptest.NewRecorder()
	server.HandleIngest(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	// 3. Missing fields
	req = httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBufferString(`{"event_id": ""}`))
	rr = httptest.NewRecorder()
	server.HandleIngest(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	// 4. GDPR consent failure
	badConsentEvent := broker.TrackingEvent{
		EventID:     "ev_test_bad_consent",
		UserID:      "usr_consent_fail",
		EventType:   "click",
		GDPRConsent: false,
	}
	payload, _ := json.Marshal(badConsentEvent)
	req = httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBuffer(payload))
	rr = httptest.NewRecorder()
	server.HandleIngest(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	// 5. Success Flow
	successEvent := broker.TrackingEvent{
		EventID:     "ev_test_ok",
		UserID:      "usr_ok",
		EventType:   "signup",
		IPAddress:   "192.168.1.1",
		GDPRConsent: true,
	}
	payload, _ = json.Marshal(successEvent)
	req = httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBuffer(payload))
	rr = httptest.NewRecorder()
	server.HandleIngest(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, rr.Code)
	}

	// 6. Broker publish failure
	failedBrokerSrv := NewIngestServer(&mockFailedBroker{}, tracker)
	req = httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBuffer(payload))
	rr = httptest.NewRecorder()
	failedBrokerSrv.HandleIngest(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestSetupRoutes(t *testing.T) {
	b := broker.NewInMemoryBroker(10)
	defer b.Close()
	server := NewIngestServer(b, &mockTelemetryTracker{})
	mux := http.NewServeMux()
	server.SetupRoutes(mux)
}
