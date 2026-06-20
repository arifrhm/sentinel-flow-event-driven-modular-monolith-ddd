package resilience

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// ResilientClient wraps standard HTTP clients with circuit breaking and retries.
type ResilientClient struct {
	client     *http.Client
	cb         *CircuitBreaker
	maxRetries int
	baseDelay  time.Duration
}

// NewResilientClient creates a new resilient HTTP client wrapper.
func NewResilientClient(timeout time.Duration, cbThreshold int, cbCooldown time.Duration, maxRetries int, baseDelay time.Duration, onCBChange func(from, to CircuitBreakerState)) *ResilientClient {
	return &ResilientClient{
		client: &http.Client{
			Timeout: timeout,
		},
		cb:         NewCircuitBreaker(cbThreshold, cbCooldown, onCBChange),
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

// Do executes an HTTP request, managing retries, exponential backoff, and circuit breaker status.
func (rc *ResilientClient) Do(req *http.Request) (*http.Response, error) {
	if !rc.cb.Allow() {
		return nil, errors.New("circuit breaker is open - request blocked")
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= rc.maxRetries; attempt++ {
		// If attempt > 0, sleep with exponential backoff and jitter
		if attempt > 0 {
			jitter := time.Duration(rand.Intn(100)) * time.Millisecond
			delay := rc.baseDelay * (1 << uint(attempt-1)) + jitter

			select {
			case <-req.Context().Done():
				rc.cb.RecordFailure()
				return nil, req.Context().Err()
			case <-time.After(delay):
			}
		}

		// Create a new request copy for retrying to prevent reuse errors
		var retryReq *http.Request
		if attempt > 0 {
			retryReq = req.Clone(req.Context())
		} else {
			retryReq = req
		}

		resp, err = rc.client.Do(retryReq)

		// If it succeeded and status is fine (< 500), record success and return
		if err == nil && resp.StatusCode < 500 {
			rc.cb.RecordSuccess()
			return resp, nil
		}

		// Connection failures or server-side 5xx errors qualify for retries and count as circuit breaker failures
		rc.cb.RecordFailure()
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", rc.maxRetries, err)
	}
	return resp, fmt.Errorf("request failed with status %s after %d retries", resp.Status, rc.maxRetries)
}

// SetupMockCRMServer returns an http.Handler that simulates a live CRM (HubSpot/Mailchimp API).
// It simulates 3 states: 
// 1. Success (80% of times)
// 2. Random 503 Internal Server Error (15% of times)
// 3. High latency resulting in timeouts (5% of times)
func SetupMockCRMServer() http.Handler {
	rand.Seed(time.Now().UnixNano())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roll := rand.Intn(100)

		// Simulate network latency
		if roll < 5 {
			// Hang the connection to trigger a timeout
			time.Sleep(3 * time.Second)
			w.WriteHeader(http.StatusGatewayTimeout)
			w.Write([]byte(`{"error": "timeout"}`))
			return
		}

		if roll >= 5 && roll < 20 {
			// Fail with 503
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": "service unavailable"}`))
			return
		}

		// Otherwise success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "synced", "message": "contact updated successfully"}`))
	})
}
