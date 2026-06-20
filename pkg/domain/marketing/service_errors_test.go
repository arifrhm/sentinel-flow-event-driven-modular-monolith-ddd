package marketing

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sentinel-flow/pkg/broker"
	"sentinel-flow/pkg/resilience"
)

// errDeleteCRMRepo simulates a local marketing repository where DeleteCRMContact fails.
type errDeleteCRMRepo struct {
	MarketingRepository
}

func (e *errDeleteCRMRepo) DeleteCRMContact(userID string) error {
	return errors.New("mock delete crm contact error")
}

func TestHTTPCRMAdapterRequestCreationError(t *testing.T) {
	rc := resilience.NewResilientClient(100*time.Millisecond, 3, 50*time.Millisecond, 1, 1*time.Millisecond, nil)
	// Passing an invalid URL with control characters makes http.NewRequestWithContext fail.
	adapter := NewHTTPCRMAdapter("\x7f", rc)

	contact := &CRMContact{
		UserID: "usr_err",
		Email:  "err@example.com",
	}

	err := adapter.SyncContact(context.Background(), contact)
	if err == nil {
		t.Error("expected error from SyncContact due to invalid URL, got nil")
	}
}

func TestMarketingServiceHandleGDPRDeleteRepoFailure(t *testing.T) {
	repo := &errDeleteCRMRepo{MarketingRepository: NewMemoryMarketingRepository()}
	fraudSrv := &mockFraudEventService{delCount: 1}
	
	service := NewMarketingService(nil, repo, nil, fraudSrv, nil)

	reqBody := []byte(`{"user_id": "usr_delete_fail"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/privacy/gdpr-delete", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	service.HandleGDPRDelete(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected StatusOK even if local CRM delete fails, got %d", rr.Code)
	}
}

func TestHTTPCRMAdapterMarshalError(t *testing.T) {
	rc := resilience.NewResilientClient(100*time.Millisecond, 3, 50*time.Millisecond, 1, 1*time.Millisecond, nil)
	adapter := NewHTTPCRMAdapter("http://localhost", rc)
	contact := &CRMContact{
		SyncedAt: time.Date(-5000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	err := adapter.SyncContact(context.Background(), contact)
	if err == nil {
		t.Error("expected JSON marshal error, got nil")
	}
}

func TestMemoryMarketingRepositoryGetPrivacyLogsNotFound(t *testing.T) {
	repo := NewMemoryMarketingRepository()
	logs, err := repo.GetPrivacyLogsByUserID("nonexistent_user")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected empty logs, got %v", logs)
	}
}

func TestMarketingServiceStartConsumerChannelClosed(t *testing.T) {
	// Import broker package if not present or reference it
	b := broker.NewInMemoryBroker(1)
	repo := NewMemoryMarketingRepository()
	service := NewMarketingService(b, repo, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- service.StartConsumer(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	_ = b.Close()

	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("expected nil error on broker close, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for StartConsumer to exit on broker close")
	}
}

func TestHTTPCRMAdapterDoError(t *testing.T) {
	rc := resilience.NewResilientClient(100*time.Millisecond, 3, 50*time.Millisecond, 1, 1*time.Millisecond, nil)
	adapter := NewHTTPCRMAdapter("http://localhost", rc)
	contact := &CRMContact{
		UserID: "usr_sync_test",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := adapter.SyncContact(ctx, contact)
	if err == nil {
		t.Fatal("expected error with canceled context, got nil")
	}
}

func TestMarketingConsumerWorkerFraudulentEvent(t *testing.T) {
	b := broker.NewInMemoryBroker(1)
	defer b.Close()

	repo := NewMemoryMarketingRepository()
	crm := &mockCRMAdapter{}
	service := NewMarketingService(b, repo, nil, nil, crm)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = service.StartConsumer(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	_ = b.PublishScreened(ctx, &broker.ScreenedEvent{
		TrackingEvent: broker.TrackingEvent{
			EventID:   "evt_worker_fraud",
			UserID:    "usr_worker_fraud",
			EventType: "signup",
		},
		IsFraudulent: true,
	})

	time.Sleep(50 * time.Millisecond)

	if crm.synced != nil && crm.synced["usr_worker_fraud"] != nil {
		t.Error("expected fraudulent event to be skipped, but it was synced to CRM")
	}
}
