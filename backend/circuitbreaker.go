package backend

import (
	"fmt"
	"sync"
	"time"
)

// circuitBreaker implements the circuit breaker pattern
type circuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	failures     int
	lastFailTime time.Time
	state        string // "closed", "open", "half-open"
	mu           sync.RWMutex
}

// newCircuitBreaker creates a new circuit breaker
func newCircuitBreaker(maxFailures int, resetTimeout time.Duration) *circuitBreaker {
	return &circuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        "closed",
	}
}

// call executes the given function with circuit breaker protection
func (cb *circuitBreaker) call(fn func() error) error {
	cb.mu.Lock()

	// Check if circuit is open
	if cb.state == "open" {
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			// Transition to half-open state
			cb.state = "half-open"
			cb.mu.Unlock()
		} else {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker is open, rejecting request")
		}
	} else {
		cb.mu.Unlock()
	}

	// Execute function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		// Open circuit if max failures reached
		if cb.failures >= cb.maxFailures {
			cb.state = "open"
		}

		return err
	}

	// Success - reset circuit
	if cb.state == "half-open" {
		cb.state = "closed"
	}
	cb.failures = 0

	return nil
}

// getState returns the current circuit breaker state
func (cb *circuitBreaker) getState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// reset resets the circuit breaker to closed state
func (cb *circuitBreaker) reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = "closed"
	cb.lastFailTime = time.Time{}
}

// getStats returns circuit breaker statistics
func (cb *circuitBreaker) getStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stats := map[string]interface{}{
		"state":        cb.state,
		"failures":     cb.failures,
		"max_failures": cb.maxFailures,
	}

	if !cb.lastFailTime.IsZero() {
		stats["last_failure"] = cb.lastFailTime.Format(time.RFC3339)
		stats["time_since_failure"] = int64(time.Since(cb.lastFailTime).Seconds())
	}

	return stats
}
