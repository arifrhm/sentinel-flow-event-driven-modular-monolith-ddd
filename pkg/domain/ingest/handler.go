package ingest

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"sentinel-flow/pkg/broker"
)

// TelemetryTracker defines the port interface for updating ingestion statistics.
type TelemetryTracker interface {
	IncrementMetric(name string, delta int64)
}

type IngestServer struct {
	broker  broker.Broker
	tracker TelemetryTracker
}

func NewIngestServer(b broker.Broker, t TelemetryTracker) *IngestServer {
	return &IngestServer{
		broker:  b,
		tracker: t,
	}
}

func (s *IngestServer) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event broker.TrackingEvent
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		http.Error(w, `{"error": "invalid json payload"}`, http.StatusBadRequest)
		return
	}

	if event.EventID == "" || event.UserID == "" || event.EventType == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "event_id, user_id, and event_type are required"}`))
		return
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Domain logic check for GDPR consent
	err = CheckGDPRConsent(event.EventType, event.GDPRConsent)
	if err != nil {
		s.tracker.IncrementMetric("total_received", 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "GDPR compliance check failed: consent required for behavioral tracking"}`))
		return
	}

	// Domain logic IP anonymization
	event.IPAddress = AnonymizeIP(event.IPAddress)

	// Publish to raw events integration channel
	err = s.broker.PublishRaw(r.Context(), &event)
	if err != nil {
		slog.Error("Failed to publish event to broker", "event_id", event.EventID, "error", err)
		http.Error(w, `{"error": "internal broker error"}`, http.StatusInternalServerError)
		return
	}

	s.tracker.IncrementMetric("total_received", 1)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "accepted", "message": "event queued for screening"}`))
}

func (s *IngestServer) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/events", s.HandleIngest)
}
