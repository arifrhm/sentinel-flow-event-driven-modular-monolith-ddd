package marketing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sentinel-flow/pkg/broker"
)

type mockMarketingTelemetryTracker struct {
	metrics map[string]int64
}

func (m *mockMarketingTelemetryTracker) IncrementMetric(name string, delta int64) {
	if m.metrics == nil {
		m.metrics = make(map[string]int64)
	}
	m.metrics[name] += delta
}

type mockCRMAdapter struct {
	synced map[string]*CRMContact
	err    error
}

func (m *mockCRMAdapter) SyncContact(ctx context.Context, contact *CRMContact) error {
	if m.err != nil {
		return m.err
	}
	if m.synced == nil {
		m.synced = make(map[string]*CRMContact)
	}
	m.synced[contact.UserID] = contact
	return nil
}

type mockFraudEventService struct {
	events   map[string][]*broker.ScreenedEvent
	delCount int
	err      error
}

func (m *mockFraudEventService) GetEventsByUserID(userID string) ([]*broker.ScreenedEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events[userID], nil
}

func (m *mockFraudEventService) DeleteUserEvents(userID string) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.delCount, nil
}

type mockFailedBroker struct {
	broker.Broker
}

func (m *mockFailedBroker) SubscribeScreened(ctx context.Context) (<-chan *broker.ScreenedEvent, error) {
	return nil, errors.New("mock subscribe error")
}

func TestMarketingServiceWorkflow(t *testing.T) {
	b := broker.NewInMemoryBroker(10)
	defer b.Close()

	repo := NewMemoryMarketingRepository()
	tracker := &mockMarketingTelemetryTracker{}
	crm := &mockCRMAdapter{}
	fraudSrv := &mockFraudEventService{}

	service := NewMarketingService(b, repo, tracker, fraudSrv, crm)

	// Signup workflow
	evt := &broker.ScreenedEvent{
		TrackingEvent: broker.TrackingEvent{
			EventID:   "evt_signup",
			UserID:    "usr_signup",
			EventType: "signup",
			Payload:   map[string]interface{}{"email": "signup@example.com"},
		},
		IsFraudulent: false,
	}

	service.ProcessMarketingWorkflow(context.Background(), evt)

	if crm.synced["usr_signup"] == nil {
		t.Error("expected user to be synced with CRM")
	}
	contact, _ := repo.GetCRMContact("usr_signup")
	if contact == nil || contact.SyncStatus != "synced" {
		t.Errorf("unexpected saved contact state: %+v", contact)
	}

	// Unhandled workflow event
	crm.synced = nil
	evtUnhandled := &broker.ScreenedEvent{
		TrackingEvent: broker.TrackingEvent{
			UserID:    "usr_unhandled",
			EventType: "unhandled_action",
		},
	}
	service.ProcessMarketingWorkflow(context.Background(), evtUnhandled)
	if len(crm.synced) > 0 {
		t.Error("unexpected sync for unhandled workflow action")
	}

	// Failed CRM sync
	crm.err = errors.New("crm connection timeout")
	evtFail := &broker.ScreenedEvent{
		TrackingEvent: broker.TrackingEvent{
			EventID:   "evt_fail",
			UserID:    "usr_fail",
			EventType: "signup",
		},
	}
	service.ProcessMarketingWorkflow(context.Background(), evtFail)
	contactFail, _ := repo.GetCRMContact("usr_fail")
	if contactFail == nil || contactFail.SyncStatus != "failed" {
		t.Error("expected CRM sync status to be failed")
	}
}

func TestMarketingServiceGDPRDelete(t *testing.T) {
	repo := NewMemoryMarketingRepository()
	_ = repo.SaveCRMContact(&CRMContact{UserID: "usr_gdpr", Email: "gdpr@example.com"})

	fraudSrv := &mockFraudEventService{delCount: 2}
	service := NewMarketingService(nil, repo, nil, fraudSrv, nil)

	// Bad payload (missing UserID)
	req := httptest.NewRequest(http.MethodPost, "/v1/privacy/gdpr-delete", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()
	service.HandleGDPRDelete(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected StatusBadRequest, got %d", rr.Code)
	}

	// Success delete
	body, _ := json.Marshal(map[string]string{"user_id": "usr_gdpr"})
	req = httptest.NewRequest(http.MethodPost, "/v1/privacy/gdpr-delete", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	service.HandleGDPRDelete(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected StatusOK, got %d", rr.Code)
	}

	// CRM contact should be deleted
	_, err := repo.GetCRMContact("usr_gdpr")
	if err == nil {
		t.Error("expected CRM contact to be deleted")
	}

	// Error path
	fraudSrv.err = errors.New("db delete failure")
	req = httptest.NewRequest(http.MethodPost, "/v1/privacy/gdpr-delete", bytes.NewBuffer(body))
	rr = httptest.NewRecorder()
	service.HandleGDPRDelete(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected InternalServerError, got %d", rr.Code)
	}
}

func TestMarketingServiceCCPAExport(t *testing.T) {
	repo := NewMemoryMarketingRepository()
	_ = repo.SaveCRMContact(&CRMContact{UserID: "usr_ccpa", Email: "ccpa@example.com"})

	fraudSrv := &mockFraudEventService{
		events: map[string][]*broker.ScreenedEvent{
			"usr_ccpa": {
				{TrackingEvent: broker.TrackingEvent{EventID: "ev_1", UserID: "usr_ccpa"}},
			},
		},
	}
	service := NewMarketingService(nil, repo, nil, fraudSrv, nil)

	// Missing user_id param
	req := httptest.NewRequest(http.MethodGet, "/v1/privacy/ccpa-export", nil)
	rr := httptest.NewRecorder()
	service.HandleCCPAExport(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected StatusBadRequest, got %d", rr.Code)
	}

	// Success export
	req = httptest.NewRequest(http.MethodGet, "/v1/privacy/ccpa-export?user_id=usr_ccpa", nil)
	rr = httptest.NewRecorder()
	service.HandleCCPAExport(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected StatusOK, got %d", rr.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["user_id"] != "usr_ccpa" {
		t.Error("unexpected export payload")
	}

	// Error path
	fraudSrv.err = errors.New("db query failure")
	req = httptest.NewRequest(http.MethodGet, "/v1/privacy/ccpa-export?user_id=usr_ccpa", nil)
	rr = httptest.NewRecorder()
	service.HandleCCPAExport(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected InternalServerError, got %d", rr.Code)
	}
}

func TestMarketingConsumerWorker(t *testing.T) {
	b := broker.NewInMemoryBroker(10)
	defer b.Close()

	repo := NewMemoryMarketingRepository()
	tracker := &mockMarketingTelemetryTracker{}
	crm := &mockCRMAdapter{}
	fraudSrv := &mockFraudEventService{}

	service := NewMarketingService(b, repo, tracker, fraudSrv, crm)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = service.StartConsumer(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Publish non-fraud event
	_ = b.PublishScreened(ctx, &broker.ScreenedEvent{
		TrackingEvent: broker.TrackingEvent{
			EventID:   "evt_worker",
			UserID:    "usr_worker",
			EventType: "signup",
		},
		IsFraudulent: false,
	})

	time.Sleep(50 * time.Millisecond)

	if crm.synced["usr_worker"] == nil {
		t.Error("expected worker consumer to process and sync contact")
	}
}

type errMktRepo struct {
	MarketingRepository
	savePrivacyErr bool
	saveCRMErr     bool
}

func (e *errMktRepo) SavePrivacyLog(log *PrivacyLog) error {
	if e.savePrivacyErr {
		return errors.New("mock privacy save error")
	}
	return e.MarketingRepository.SavePrivacyLog(log)
}

func (e *errMktRepo) SaveCRMContact(contact *CRMContact) error {
	if e.saveCRMErr {
		return errors.New("mock crm save error")
	}
	return e.MarketingRepository.SaveCRMContact(contact)
}

func TestMarketingServiceEdgeCases(t *testing.T) {
	// 1. SetupRoutes call
	b := broker.NewInMemoryBroker(1)
	defer b.Close()
	repo := NewMemoryMarketingRepository()
	service := NewMarketingService(b, repo, nil, nil, nil)
	mux := http.NewServeMux()
	service.SetupRoutes(mux)

	// 2. GetPrivacyLogsByUserID cover
	_ = repo.SavePrivacyLog(&PrivacyLog{LogID: "l1", UserID: "usr_1"})
	logs, err := repo.GetPrivacyLogsByUserID("usr_1")
	if err != nil || len(logs) != 1 {
		t.Error("failed to get privacy logs")
	}

	// 3. GDPR delete log save failure
	errRepo := &errMktRepo{MarketingRepository: NewMemoryMarketingRepository(), savePrivacyErr: true}
	fraudSrv := &mockFraudEventService{delCount: 1}
	serviceErr := NewMarketingService(b, errRepo, nil, fraudSrv, nil)
	body, _ := json.Marshal(map[string]string{"user_id": "usr_gdpr"})
	req := httptest.NewRequest(http.MethodPost, "/v1/privacy/gdpr-delete", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	serviceErr.HandleGDPRDelete(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected OK even if audit log save fails, got %d", rr.Code)
	}

	// 4. HandleCCPAExport contact missing (no CRM contact synced)
	serviceCCPAEmpty := NewMarketingService(b, NewMemoryMarketingRepository(), nil, fraudSrv, nil)
	reqCCPA := httptest.NewRequest(http.MethodGet, "/v1/privacy/ccpa-export?user_id=usr_missing", nil)
	rrCCPA := httptest.NewRecorder()
	serviceCCPAEmpty.HandleCCPAExport(rrCCPA, reqCCPA)
	if rrCCPA.Code != http.StatusOK {
		t.Errorf("expected StatusOK, got %d", rrCCPA.Code)
	}

	// 5. Workflows mapping (checkout_completed, cart_abandoned)
	crm := &mockCRMAdapter{}
	tracker := &mockMarketingTelemetryTracker{}
	serviceWorkflows := NewMarketingService(b, repo, tracker, fraudSrv, crm)

	evtCheckout := &broker.ScreenedEvent{
		TrackingEvent: broker.TrackingEvent{
			UserID:    "usr_chk",
			EventType: "checkout_completed",
		},
	}
	serviceWorkflows.ProcessMarketingWorkflow(context.Background(), evtCheckout)
	if crm.synced["usr_chk"] == nil || crm.synced["usr_chk"].WorkflowTriggered != "Loyalty Rewards Activation" {
		t.Error("checkout workflow failed")
	}

	evtCart := &broker.ScreenedEvent{
		TrackingEvent: broker.TrackingEvent{
			UserID:    "usr_cart",
			EventType: "cart_abandoned",
		},
	}
	serviceWorkflows.ProcessMarketingWorkflow(context.Background(), evtCart)
	if crm.synced["usr_cart"] == nil || crm.synced["usr_cart"].WorkflowTriggered != "Recovery Email Sequence" {
		t.Error("cart workflow failed")
	}

	// Save CRM contact error in workflow
	errRepoCRM := &errMktRepo{MarketingRepository: NewMemoryMarketingRepository(), saveCRMErr: true}
	serviceErrCRM := NewMarketingService(b, errRepoCRM, tracker, fraudSrv, crm)
	serviceErrCRM.ProcessMarketingWorkflow(context.Background(), evtCart)

	// 6. Consumer worker error subscribe
	serviceBadWorker := NewMarketingService(&mockFailedBroker{}, repo, nil, nil, nil)
	err = serviceBadWorker.StartConsumer(context.Background())
	if err == nil {
		t.Error("expected subscribe error")
	}

	// 7. Bad request methods on GDPR and CCPA handlers
	reqGDPRBad := httptest.NewRequest(http.MethodGet, "/v1/privacy/gdpr-delete", nil)
	rrGDPRBad := httptest.NewRecorder()
	serviceCCPAEmpty.HandleGDPRDelete(rrGDPRBad, reqGDPRBad)
	if rrGDPRBad.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected MethodNotAllowed, got %d", rrGDPRBad.Code)
	}

	reqCCPABad := httptest.NewRequest(http.MethodPost, "/v1/privacy/ccpa-export", nil)
	rrCCPABad := httptest.NewRecorder()
	serviceCCPAEmpty.HandleCCPAExport(rrCCPABad, reqCCPABad)
	if rrCCPABad.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected MethodNotAllowed, got %d", rrCCPABad.Code)
	}
}
