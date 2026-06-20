package marketing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sentinel-flow/pkg/resilience"
)

func TestHTTPCRMAdapterSyncSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"synced"}`))
	}))
	defer server.Close()

	rc := resilience.NewResilientClient(100*time.Millisecond, 3, 50*time.Millisecond, 1, 1*time.Millisecond, nil)
	adapter := NewHTTPCRMAdapter(server.URL, rc)

	contact := &CRMContact{
		UserID:            "usr_sync_test",
		Email:             "sync@example.com",
		WorkflowTriggered: "onboarding",
	}

	err := adapter.SyncContact(context.Background(), contact)
	if err != nil {
		t.Fatalf("SyncContact failed: %v", err)
	}
}

func TestHTTPCRMAdapterSyncFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`error detail`))
	}))
	defer server.Close()

	rc := resilience.NewResilientClient(100*time.Millisecond, 3, 50*time.Millisecond, 1, 1*time.Millisecond, nil)
	adapter := NewHTTPCRMAdapter(server.URL, rc)

	contact := &CRMContact{
		UserID: "usr_sync_test",
	}

	err := adapter.SyncContact(context.Background(), contact)
	if err == nil {
		t.Fatal("expected error from SyncContact, got nil")
	}
}
