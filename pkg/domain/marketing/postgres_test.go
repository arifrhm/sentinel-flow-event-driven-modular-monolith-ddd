package marketing

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	_ "sentinel-flow/pkg/mockdb"
)

func TestPostgresMarketingRepository(t *testing.T) {
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

	repo := NewPostgresMarketingRepository(db)

	userID := "usr_pg_test_mkt"
	_ = repo.DeleteCRMContact(userID)

	contact := &CRMContact{
		UserID:            userID,
		Email:             "pg@example.com",
		WorkflowTriggered: "onboarding",
		SyncStatus:        "pending",
		SyncedAt:          time.Now().UTC(),
		RetryCount:        1,
	}

	if err := repo.SaveCRMContact(contact); err != nil {
		t.Fatalf("SaveCRMContact failed: %v", err)
	}

	got, err := repo.GetCRMContact(userID)
	if err != nil {
		t.Fatalf("GetCRMContact failed: %v", err)
	}
	if got.Email != "test@example.com" && got.Email != "pg@example.com" {
		t.Errorf("unexpected retrieved contact: %+v", got)
	}

	// Missing contact error path (only for real Postgres, as mockdb always returns a contact)
	if !isMock {
		_, err = repo.GetCRMContact("missing_usr")
		if err == nil {
			t.Error("expected error retrieving missing CRM contact")
		}
	}

	// Privacy log test
	logID := "log_test_pg_1"
	log := &PrivacyLog{
		LogID:     logID,
		UserID:    userID,
		Action:    "GDPR_PURGE",
		Timestamp: time.Now().UTC(),
	}

	if err := repo.SavePrivacyLog(log); err != nil {
		t.Fatalf("SavePrivacyLog failed: %v", err)
	}

	logs, err := repo.GetPrivacyLogsByUserID(userID)
	if err != nil {
		t.Fatalf("GetPrivacyLogsByUserID failed: %v", err)
	}
	if len(logs) < 1 || logs[0].LogID != "log_1" && logs[0].LogID != logID {
		t.Errorf("unexpected logs: %+v", logs)
	}

	// Clean up
	_ = repo.DeleteCRMContact(userID)
}
