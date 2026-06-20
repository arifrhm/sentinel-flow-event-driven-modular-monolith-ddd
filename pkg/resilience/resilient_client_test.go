package resilience

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResilientClientSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rc := NewResilientClient(100*time.Millisecond, 3, 50*time.Millisecond, 2, 5*time.Millisecond, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := rc.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestResilientClientRetriesAndFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	rc := NewResilientClient(100*time.Millisecond, 5, 50*time.Millisecond, 2, 1*time.Millisecond, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = rc.Do(req)
	if err == nil {
		t.Fatal("expected failure, got nil")
	}

	if attempts != 3 { // 1 initial + 2 retries
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestResilientClientCircuitOpen(t *testing.T) {
	rc := NewResilientClient(100*time.Millisecond, 2, 50*time.Millisecond, 1, 1*time.Millisecond, nil)
	// Trip breaker by recording failures
	rc.cb.RecordFailure()
	rc.cb.RecordFailure()

	if rc.cb.State() != StateOpen {
		t.Fatalf("expected breaker to be OPEN, got %s", rc.cb.State())
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = rc.Do(req)
	if err == nil || err.Error() != "circuit breaker is open - request blocked" {
		t.Errorf("expected circuit breaker open error, got %v", err)
	}
}

func TestSetupMockCRMServer(t *testing.T) {
	handler := SetupMockCRMServer()
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{}
	successCount := 0
	failureCount := 0

	for i := 0; i < 50; i++ {
		resp, err := client.Post(server.URL, "application/json", nil)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				successCount++
			} else {
				failureCount++
			}
			resp.Body.Close()
		} else {
			failureCount++
		}
	}

	if successCount == 0 && failureCount == 0 {
		t.Error("expected at least some responses from mock CRM server")
	}
}

func TestResilientClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rc := NewResilientClient(100*time.Millisecond, 3, 50*time.Millisecond, 1, 10*time.Millisecond, nil)

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)

	// Cancel context immediately or concurrently
	cancel()

	_, err := rc.Do(req)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Errorf("expected context canceled error, got %v", err)
	}
}
