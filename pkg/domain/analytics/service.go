package analytics

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type AnalyticsService struct {
	tracker TelemetryTracker
}

func NewAnalyticsService(t TelemetryTracker) *AnalyticsService {
	return &AnalyticsService{
		tracker: t,
	}
}

func (s *AnalyticsService) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.tracker.GetMetrics()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

func (s *AnalyticsService) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.tracker.GetMetrics()

	var fraudPct float64
	totalProcessed := metrics.LegitimateEvents + metrics.FraudulentEvents
	if totalProcessed > 0 {
		fraudPct = (float64(metrics.FraudulentEvents) / float64(totalProcessed)) * 100
	}

	var crmSuccessRate float64
	if metrics.CRMAttempts > 0 {
		crmSuccessRate = (float64(metrics.CRMSuccesses) / float64(metrics.CRMAttempts)) * 100
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "===================================================\n")
	fmt.Fprintf(w, "          SENTINEL-FLOW DASHBOARD TELEMETRY        \n")
	fmt.Fprintf(w, "===================================================\n")
	fmt.Fprintf(w, "SYSTEM THROUGHPUT\n")
	fmt.Fprintf(w, "  Total Events Ingested:      %d events\n", metrics.TotalEventsReceived)
	fmt.Fprintf(w, "  Current Ingest Velocity:    %.2f events/sec\n\n", metrics.EventsPerSecond)

	fmt.Fprintf(w, "FRAUD OPERATIONS DETECTION\n")
	fmt.Fprintf(w, "  Legitimate Events (Clean):  %d\n", metrics.LegitimateEvents)
	fmt.Fprintf(w, "  Fraudulent Events (Blocked):%d\n", metrics.FraudulentEvents)
	fmt.Fprintf(w, "  Fraud Incidence Rate:       %.2f%%\n\n", fraudPct)

	fmt.Fprintf(w, "MARKETING CRM AUTOMATION SYNC (HubSpot/Mailchimp)\n")
	fmt.Fprintf(w, "  Total Sync Actions Logged:  %d\n", metrics.CRMAttempts)
	fmt.Fprintf(w, "  Successful Synced Contacts: %d\n", metrics.CRMSuccesses)
	fmt.Fprintf(w, "  Failed Sync Contacts:       %d\n", metrics.CRMFailures)
	fmt.Fprintf(w, "  CRM Sync Resilience Rate:   %.2f%%\n", crmSuccessRate)
	fmt.Fprintf(w, "===================================================\n")
}

func (s *AnalyticsService) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/metrics", s.HandleMetrics)
	mux.HandleFunc("/v1/dashboard", s.HandleDashboard)
}
