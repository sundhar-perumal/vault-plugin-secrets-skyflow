package backend

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_NewCircuitBreaker(t *testing.T) {
	cb := newCircuitBreaker(5, 60*time.Second)

	if cb == nil {
		t.Fatal("circuit breaker should not be nil")
	}

	if cb.maxFailures != 5 {
		t.Errorf("expected maxFailures 5, got %d", cb.maxFailures)
	}

	if cb.resetTimeout != 60*time.Second {
		t.Errorf("expected resetTimeout 60s, got %v", cb.resetTimeout)
	}

	if cb.state != "closed" {
		t.Errorf("expected initial state 'closed', got '%s'", cb.state)
	}
}

func TestCircuitBreaker_Call_Success(t *testing.T) {
	cb := newCircuitBreaker(3, 1*time.Second)

	err := cb.call(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if cb.getState() != "closed" {
		t.Errorf("expected state 'closed', got '%s'", cb.getState())
	}
}

func TestCircuitBreaker_Call_Failure(t *testing.T) {
	cb := newCircuitBreaker(3, 1*time.Second)

	testErr := errors.New("test error")

	err := cb.call(func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("expected error '%v', got '%v'", testErr, err)
	}

	// Should still be closed after one failure
	if cb.getState() != "closed" {
		t.Errorf("expected state 'closed', got '%s'", cb.getState())
	}
}

func TestCircuitBreaker_Opens_AfterMaxFailures(t *testing.T) {
	cb := newCircuitBreaker(3, 1*time.Second)

	testErr := errors.New("test error")

	// Cause 3 failures
	for i := 0; i < 3; i++ {
		cb.call(func() error {
			return testErr
		})
	}

	// Should now be open
	if cb.getState() != "open" {
		t.Errorf("expected state 'open', got '%s'", cb.getState())
	}
}

func TestCircuitBreaker_Rejects_WhenOpen(t *testing.T) {
	cb := newCircuitBreaker(2, 1*time.Hour) // Long timeout

	testErr := errors.New("test error")

	// Cause 2 failures to open the circuit
	for i := 0; i < 2; i++ {
		cb.call(func() error {
			return testErr
		})
	}

	// Attempt another call - should be rejected
	err := cb.call(func() error {
		return nil
	})

	if err == nil {
		t.Error("expected error when circuit is open")
	}

	if err.Error() != "circuit breaker is open, rejecting request" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	cb := newCircuitBreaker(2, 10*time.Millisecond)

	testErr := errors.New("test error")

	// Cause failures to open circuit
	for i := 0; i < 2; i++ {
		cb.call(func() error {
			return testErr
		})
	}

	if cb.getState() != "open" {
		t.Fatalf("expected state 'open', got '%s'", cb.getState())
	}

	// Wait for reset timeout
	time.Sleep(20 * time.Millisecond)

	// Next call should transition to half-open
	cb.call(func() error {
		return nil
	})

	// Should now be closed after successful call in half-open
	if cb.getState() != "closed" {
		t.Errorf("expected state 'closed' after success in half-open, got '%s'", cb.getState())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := newCircuitBreaker(2, 1*time.Hour)

	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.call(func() error {
			return testErr
		})
	}

	if cb.getState() != "open" {
		t.Fatalf("expected state 'open', got '%s'", cb.getState())
	}

	// Reset
	cb.reset()

	if cb.getState() != "closed" {
		t.Errorf("expected state 'closed' after reset, got '%s'", cb.getState())
	}

	stats := cb.getStats()
	if stats["failures"].(int) != 0 {
		t.Errorf("expected 0 failures after reset, got %v", stats["failures"])
	}
}

func TestCircuitBreaker_GetStats(t *testing.T) {
	cb := newCircuitBreaker(5, 30*time.Second)

	stats := cb.getStats()

	if stats["state"] != "closed" {
		t.Errorf("expected state 'closed', got '%s'", stats["state"])
	}

	if stats["failures"].(int) != 0 {
		t.Errorf("expected 0 failures, got %v", stats["failures"])
	}

	if stats["max_failures"].(int) != 5 {
		t.Errorf("expected max_failures 5, got %v", stats["max_failures"])
	}

	// last_failure should not be present when no failures
	if _, ok := stats["last_failure"]; ok {
		t.Error("last_failure should not be present when there are no failures")
	}
}

func TestCircuitBreaker_GetStats_WithFailure(t *testing.T) {
	cb := newCircuitBreaker(5, 30*time.Second)

	// Cause a failure
	cb.call(func() error {
		return errors.New("test error")
	})

	stats := cb.getStats()

	if stats["failures"].(int) != 1 {
		t.Errorf("expected 1 failure, got %v", stats["failures"])
	}

	if _, ok := stats["last_failure"]; !ok {
		t.Error("last_failure should be present after a failure")
	}

	if _, ok := stats["time_since_failure"]; !ok {
		t.Error("time_since_failure should be present after a failure")
	}
}

