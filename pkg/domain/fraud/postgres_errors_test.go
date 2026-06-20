package fraud

import (
	"database/sql"
	"errors"
	"testing"

	"sentinel-flow/pkg/broker"
	"sentinel-flow/pkg/mockdb"
)

func TestPostgresFraudRepositoryErrors(t *testing.T) {
	db, err := sql.Open("mockdb", "")
	if err != nil {
		t.Fatalf("failed to open mockdb: %v", err)
	}
	defer db.Close()

	repo := NewPostgresFraudRepository(db)

	event := &broker.ScreenedEvent{
		TrackingEvent: broker.TrackingEvent{
			EventID: "ev_err_1",
			UserID:  "usr_err",
		},
	}

	// 1. SaveEvent query error
	mockdb.SimError = errors.New("save error")
	if err := repo.SaveEvent(event); err == nil {
		t.Error("expected error from SaveEvent, got nil")
	}
	mockdb.SimError = nil

	// 2. GetEventsByUserID query error
	mockdb.SimError = errors.New("query error")
	if _, err := repo.GetEventsByUserID("usr_err"); err == nil {
		t.Error("expected error from GetEventsByUserID, got nil")
	}
	mockdb.SimError = nil

	// 3. GetEventsByUserID scan error
	mockdb.TriggerScanError = true
	if _, err := repo.GetEventsByUserID("usr_err"); err == nil {
		t.Error("expected scan error from GetEventsByUserID, got nil")
	}
	mockdb.TriggerScanError = false

	// 4. DeleteUserEvents Begin transaction error
	mockdb.SimError = errors.New("begin error")
	if _, err := repo.DeleteUserEvents("usr_err"); err == nil {
		t.Error("expected begin error from DeleteUserEvents, got nil")
	}
	mockdb.SimError = nil

	// 5. GetEventsByUserID Next Error
	mockdb.NextError = errors.New("mock rows next error")
	if _, err := repo.GetEventsByUserID("usr_err"); err == nil {
		t.Error("expected error from GetEventsByUserID on rows next error, got nil")
	}
	mockdb.NextError = nil

	// 6. DeleteUserEvents Exec CRM Delete Error
	mockdb.FailCRMDelete = true
	if _, err := repo.DeleteUserEvents("usr_err"); err == nil {
		t.Error("expected error from DeleteUserEvents on CRM delete failure, got nil")
	}
	mockdb.FailCRMDelete = false

	// 7. DeleteUserEvents Commit Error
	mockdb.CommitError = errors.New("mock commit error")
	if _, err := repo.DeleteUserEvents("usr_err"); err == nil {
		t.Error("expected error from DeleteUserEvents on Commit failure, got nil")
	}
	mockdb.CommitError = nil

	// 8. GetEventsByUserID JSON unmarshal error
	mockdb.InvalidJSONPayload = true
	if _, err := repo.GetEventsByUserID("usr_err"); err == nil {
		t.Error("expected JSON unmarshal error, got nil")
	}
	mockdb.InvalidJSONPayload = false

	// 9. DeleteUserEvents Exec error on events table deletion
	mockdb.ExecError = errors.New("mock exec error on events delete")
	if _, err := repo.DeleteUserEvents("usr_err"); err == nil {
		t.Error("expected Exec error on events deletion, got nil")
	}
	mockdb.ExecError = nil
}
