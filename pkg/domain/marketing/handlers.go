package marketing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// hashIdentifier hashes a sensitive identifier (like UserID) using SHA-256 with a salt.
func hashIdentifier(identifier string, salt string) string {
	hasher := sha256.New()
	hasher.Write([]byte(identifier + salt))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *MarketingService) HandleGDPRDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		http.Error(w, `{"error": "invalid json or user_id missing"}`, http.StatusBadRequest)
		return
	}

	// Delete user events through the FraudService port interface
	scrubbedCount, err := s.fraudService.DeleteUserEvents(req.UserID)
	if err != nil {
		slog.Error("[Marketing] Failed to scrub event logs via FraudService", "user_id", req.UserID, "error", err)
		http.Error(w, `{"error": "failed to scrub records"}`, http.StatusInternalServerError)
		return
	}

	// Clean up local CRM records as well
	if err := s.repo.DeleteCRMContact(req.UserID); err != nil {
		slog.Error("[Marketing] Failed to delete local CRM records", "user_id", req.UserID, "error", err)
	}

	hashedUserID := hashIdentifier(req.UserID, s.salt)
	auditLog := &PrivacyLog{
		LogID:     fmt.Sprintf("purge-%d", time.Now().UnixNano()),
		UserID:    hashedUserID,
		Action:    "GDPR_PURGE",
		Timestamp: time.Now(),
	}
	if err := s.repo.SavePrivacyLog(auditLog); err != nil {
		slog.Error("[Marketing] Failed to save GDPR audit log", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"status": "purged", "user_id_hashed": "%s", "records_scrubbed": %d}`, hashedUserID, scrubbedCount)))
}

func (s *MarketingService) HandleCCPAExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, `{"error": "user_id query parameter is required"}`, http.StatusBadRequest)
		return
	}

	// Fetch event logs using FraudService port interface
	events, err := s.fraudService.GetEventsByUserID(userID)
	if err != nil {
		slog.Error("[Marketing] Failed to fetch events from FraudService", "user_id", userID, "error", err)
		http.Error(w, `{"error": "failed to export data"}`, http.StatusInternalServerError)
		return
	}

	var crmContactInfo interface{}
	contact, err := s.repo.GetCRMContact(userID)
	if err == nil {
		crmContactInfo = contact
	} else {
		crmContactInfo = "no crm contact synced"
	}

	response := map[string]interface{}{
		"user_id":     userID,
		"events":      events,
		"crm_contact": crmContactInfo,
		"exported_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *MarketingService) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/privacy/gdpr-delete", s.HandleGDPRDelete)
	mux.HandleFunc("/v1/privacy/ccpa-export", s.HandleCCPAExport)
}
