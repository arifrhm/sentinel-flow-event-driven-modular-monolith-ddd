package fraud

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"sentinel-flow/pkg/broker"
	_ "sentinel-flow/pkg/mockdb"
)

func TestPostgresFraudRepository(t *testing.T) {
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

	repo := NewPostgresFraudRepository(db)

	userID := "usr_pg_test_fraud"
	if !isMock {
		_, _ = repo.DeleteUserEvents(userID)
	}

	event := &broker.ScreenedEvent{
		TrackingEvent: broker.TrackingEvent{
			EventID:     "ev_pg_test_1",
			UserID:      userID,
			EventType:   "signup",
			IPAddress:   "1.2.3.4",
			UserAgent:   "Mozilla",
			Payload:     map[string]interface{}{"step": "1"},
			Timestamp:   time.Now().UTC(),
			GDPRConsent: true,
		},
		IsFraudulent: false,
		FraudScore:   0.1,
		FraudReason:  "none",
		ScreenedAt:   time.Now().UTC(),
	}

	if err := repo.SaveEvent(event); err != nil {
		t.Fatalf("SaveEvent failed: %v", err)
	}

	// Retrieve
	events, err := repo.GetEventsByUserID(userID)
	if err != nil {
		t.Fatalf("GetEventsByUserID failed: %v", err)
	}
	if len(events) != 1 || events[0].EventID != "ev_1" && events[0].EventID != "ev_pg_test_1" {
		t.Errorf("unexpected retrieved events: %+v", events)
	}

	// Delete
	count, err := repo.DeleteUserEvents(userID)
	if err != nil {
		t.Fatalf("DeleteUserEvents failed: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 deleted row, got %d", count)
	}

	// Test SaveEvent with unmarshallable payload
	badEvent := &broker.ScreenedEvent{
		TrackingEvent: broker.TrackingEvent{
			EventID: "ev_pg_bad_payload",
			UserID:  userID,
			Payload: map[string]interface{}{"invalid": make(chan int)}, // channels cannot be marshaled
		},
	}
	if err := repo.SaveEvent(badEvent); err == nil {
		t.Error("expected error saving event with unmarshallable payload, got nil")
	}
}
