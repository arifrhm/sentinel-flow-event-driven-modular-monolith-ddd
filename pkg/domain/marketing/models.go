package marketing

import "time"

// CRMContact represents user details synced with external systems.
type CRMContact struct {
	UserID            string    `json:"user_id"`
	Email             string    `json:"email"`
	WorkflowTriggered string    `json:"workflow_triggered"`
	SyncStatus        string    `json:"sync_status"` // "pending", "synced", "failed"
	SyncedAt          time.Time `json:"synced_at"`
	RetryCount        int       `json:"retry_count"`
}

// PrivacyLog stores audit records for GDPR/CCPA compliance actions.
type PrivacyLog struct {
	LogID     string    `json:"log_id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"` // "GDPR_PURGE", "CCPA_EXPORT"
	Timestamp time.Time `json:"timestamp"`
}
