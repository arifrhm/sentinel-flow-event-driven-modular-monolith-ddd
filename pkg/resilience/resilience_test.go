package resilience

import (
	"testing"
	"time"
)

func TestCircuitBreakerStates(t *testing.T) {
	var stateChanges []string
	cb := NewCircuitBreaker(3, 50*time.Millisecond, func(from, to CircuitBreakerState) {
		stateChanges = append(stateChanges, from.String()+"->"+to.String())
	})

	// 1. Verify initial state is Closed
	if cb.State() != StateClosed {
		t.Errorf("Expected initial state CLOSED, got %s", cb.State())
	}
	if !cb.Allow() {
		t.Error("Expected request to be allowed in CLOSED state")
	}

	// 2. Record 2 failures (less than threshold 3)
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Errorf("Expected state to remain CLOSED after 2 failures, got %s", cb.State())
	}

	// 3. Record 3rd failure to trip the breaker
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Errorf("Expected state to be OPEN after 3 failures, got %s", cb.State())
	}
	if cb.Allow() {
		t.Error("Expected request to be blocked in OPEN state")
	}

	// 4. Verify state transitions logged
	if len(stateChanges) != 1 || stateChanges[0] != "CLOSED->OPEN" {
		t.Errorf("Expected state change 'CLOSED->OPEN', logged: %v", stateChanges)
	}

	// 5. Test Cooldown / Half-Open state transition
	time.Sleep(60 * time.Millisecond) // Wait for cooldown duration (50ms)
	if !cb.Allow() {
		t.Error("Expected request to be allowed as probe in HALF-OPEN state")
	}
	if cb.State() != StateHalfOpen {
		t.Errorf("Expected state to transition to HALF-OPEN on check, got %s", cb.State())
	}

	// 6. Test Successful probe resets breaker to CLOSED
	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Errorf("Expected state to return to CLOSED on success, got %s", cb.State())
	}
	if !cb.Allow() {
		t.Error("Expected request to be allowed in CLOSED state")
	}

	// Test unrecognized state String fallback
	badStateStr := CircuitBreakerState(999).String()
	if badStateStr != "UNKNOWN" {
		t.Errorf("expected UNKNOWN for state 999, got %s", badStateStr)
	}
}
