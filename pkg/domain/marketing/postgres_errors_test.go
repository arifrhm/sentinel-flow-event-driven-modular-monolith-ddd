package marketing

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"sentinel-flow/pkg/mockdb"
)

func TestPostgresMarketingRepositoryErrors(t *testing.T) {
	db, err := sql.Open("mockdb", "")
	if err != nil {
		t.Fatalf("failed to open mockdb: %v", err)
	}
	defer db.Close()

	repo := NewPostgresMarketingRepository(db)

	contact := &CRMContact{
		UserID: "usr_err",
		Email:  "err@example.com",
	}

	// 1. SaveCRMContact error
	mockdb.SimError = errors.New("save crm error")
	if err := repo.SaveCRMContact(contact); err == nil {
		t.Error("expected error from SaveCRMContact, got nil")
	}
	mockdb.SimError = nil

	// 2. GetCRMContact error (not sql.ErrNoRows)
	mockdb.SimError = errors.New("get crm error")
	if _, err := repo.GetCRMContact("usr_err"); err == nil {
		t.Error("expected error from GetCRMContact, got nil")
	}
	mockdb.SimError = nil

	// 3. GetCRMContact not found error (sql.ErrNoRows)
	mockdb.SimError = sql.ErrNoRows
	if _, err := repo.GetCRMContact("usr_err"); err == nil || err.Error() != "crm contact not found" {
		t.Errorf("expected crm contact not found error, got %v", err)
	}
	mockdb.SimError = nil

	// 4. DeleteCRMContact error
	mockdb.SimError = errors.New("delete error")
	if err := repo.DeleteCRMContact("usr_err"); err == nil {
		t.Error("expected error from DeleteCRMContact, got nil")
	}
	mockdb.SimError = nil

	// 5. SavePrivacyLog error
	log := &PrivacyLog{
		LogID:     "l_err",
		UserID:    "usr_err",
		Action:    "PURGE",
		Timestamp: time.Now(),
	}
	mockdb.SimError = errors.New("save log error")
	if err := repo.SavePrivacyLog(log); err == nil {
		t.Error("expected error from SavePrivacyLog, got nil")
	}
	mockdb.SimError = nil

	// 6. GetPrivacyLogsByUserID query error
	mockdb.SimError = errors.New("query logs error")
	if _, err := repo.GetPrivacyLogsByUserID("usr_err"); err == nil {
		t.Error("expected error from GetPrivacyLogsByUserID, got nil")
	}
	mockdb.SimError = nil

	// 7. GetPrivacyLogsByUserID scan error
	mockdb.TriggerScanError = true
	if _, err := repo.GetPrivacyLogsByUserID("usr_err"); err == nil {
		t.Error("expected scan error from GetPrivacyLogsByUserID, got nil")
	}
	mockdb.TriggerScanError = false
}
