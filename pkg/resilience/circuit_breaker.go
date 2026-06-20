package resilience

import (
	"sync"
	"time"
)

// CircuitBreakerState represents the operational state of the circuit breaker.
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateHalfOpen
	StateOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateHalfOpen:
		return "HALF-OPEN"
	case StateOpen:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker implements the Circuit Breaker pattern.
type CircuitBreaker struct {
	mu            sync.Mutex
	state         CircuitBreakerState
	failures      int
	threshold     int
	cooldown      time.Duration
	lastFailure   time.Time
	onStateChange func(from, to CircuitBreakerState)
}

// NewCircuitBreaker creates a circuit breaker instance.
func NewCircuitBreaker(threshold int, cooldown time.Duration, onStateChange func(from, to CircuitBreakerState)) *CircuitBreaker {
	return &CircuitBreaker{
		state:         StateClosed,
		threshold:     threshold,
		cooldown:      cooldown,
		onStateChange: onStateChange,
	}
}

// State returns the current state.
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Allow checks if the request should be permitted.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.cooldown {
			old := cb.state
			cb.state = StateHalfOpen
			if cb.onStateChange != nil {
				cb.onStateChange(old, cb.state)
			}
			return true
		}
		return false
	}
	return true
}

// RecordSuccess handles successful executions.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen || cb.state == StateOpen {
		old := cb.state
		cb.state = StateClosed
		cb.failures = 0
		if cb.onStateChange != nil {
			cb.onStateChange(old, cb.state)
		}
	} else {
		cb.failures = 0
	}
}

// RecordFailure handles execution failures.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == StateHalfOpen || cb.failures >= cb.threshold {
		if cb.state != StateOpen {
			old := cb.state
			cb.state = StateOpen
			if cb.onStateChange != nil {
				cb.onStateChange(old, cb.state)
			}
		}
	}
}
