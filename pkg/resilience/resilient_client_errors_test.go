package resilience

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResilientClientCancellationDuringRetry(t *testing.T) {
	// Create a server that fails with 500
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	
	ts := httptest.NewServer(server)
	defer ts.Close()

	rc := NewResilientClient(100*time.Millisecond, 5, 50*time.Millisecond, 2, 50*time.Millisecond, nil)
	
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)

	// In a goroutine, cancel the context after the first attempt starts
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := rc.Do(req)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

func TestResilientClientConnectionFailure(t *testing.T) {
	rc := NewResilientClient(10*time.Millisecond, 5, 50*time.Millisecond, 0, 1*time.Millisecond, nil)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:9999", nil)
	
	_, err := rc.Do(req)
	if err == nil {
		t.Fatal("expected DNS/connection error, got nil")
	}
}
