package backend

import (
	"testing"
	"time"
)

func TestMetrics_NewMetrics(t *testing.T) {
	m := newMetrics()

	if m == nil {
		t.Fatal("metrics should not be nil")
	}

	stats := m.getStats()

	if stats["total_requests"].(uint64) != 0 {
		t.Errorf("expected 0 total requests, got %v", stats["total_requests"])
	}

	if stats["token_generations"].(uint64) != 0 {
		t.Errorf("expected 0 token generations, got %v", stats["token_generations"])
	}

	if stats["token_errors"].(uint64) != 0 {
		t.Errorf("expected 0 token errors, got %v", stats["token_errors"])
	}
}

func TestMetrics_RecordTokenGeneration_Success(t *testing.T) {
	m := newMetrics()

	m.recordTokenGeneration(100*time.Millisecond, nil)

	stats := m.getStats()

	if stats["total_requests"].(uint64) != 1 {
		t.Errorf("expected 1 total request, got %v", stats["total_requests"])
	}

	if stats["token_generations"].(uint64) != 1 {
		t.Errorf("expected 1 token generation, got %v", stats["token_generations"])
	}

	if stats["token_errors"].(uint64) != 0 {
		t.Errorf("expected 0 token errors, got %v", stats["token_errors"])
	}
}

func TestMetrics_RecordTokenGeneration_Error(t *testing.T) {
	m := newMetrics()

	m.recordTokenGeneration(50*time.Millisecond, errTestError)

	stats := m.getStats()

	if stats["total_requests"].(uint64) != 1 {
		t.Errorf("expected 1 total request, got %v", stats["total_requests"])
	}

	if stats["token_generations"].(uint64) != 0 {
		t.Errorf("expected 0 token generations, got %v", stats["token_generations"])
	}

	if stats["token_errors"].(uint64) != 1 {
		t.Errorf("expected 1 token error, got %v", stats["token_errors"])
	}
}

func TestMetrics_AverageResponseTime(t *testing.T) {
	m := newMetrics()

	m.recordTokenGeneration(100*time.Millisecond, nil)
	m.recordTokenGeneration(200*time.Millisecond, nil)
	m.recordTokenGeneration(300*time.Millisecond, nil)

	stats := m.getStats()

	// Average should be (100+200+300)/3 = 200ms
	avgResponseTime := stats["avg_response_time_ms"].(float64)
	if avgResponseTime < 190 || avgResponseTime > 210 {
		t.Errorf("expected avg response time ~200ms, got %v", avgResponseTime)
	}
}

func TestMetrics_ErrorRate(t *testing.T) {
	m := newMetrics()

	// 3 successes, 1 error = 25% error rate
	m.recordTokenGeneration(100*time.Millisecond, nil)
	m.recordTokenGeneration(100*time.Millisecond, nil)
	m.recordTokenGeneration(100*time.Millisecond, nil)
	m.recordTokenGeneration(100*time.Millisecond, errTestError)

	stats := m.getStats()

	errorRate := stats["error_rate"].(float64)
	if errorRate < 0.24 || errorRate > 0.26 {
		t.Errorf("expected error rate ~0.25, got %v", errorRate)
	}
}

func TestMetrics_Reset(t *testing.T) {
	m := newMetrics()

	m.recordTokenGeneration(100*time.Millisecond, nil)
	m.recordTokenGeneration(100*time.Millisecond, errTestError)

	m.reset()

	stats := m.getStats()

	if stats["total_requests"].(uint64) != 0 {
		t.Errorf("expected 0 total requests after reset, got %v", stats["total_requests"])
	}

	if stats["token_generations"].(uint64) != 0 {
		t.Errorf("expected 0 token generations after reset, got %v", stats["token_generations"])
	}

	if stats["token_errors"].(uint64) != 0 {
		t.Errorf("expected 0 token errors after reset, got %v", stats["token_errors"])
	}
}

// Test error for use in tests
var errTestError = func() error {
	return &testError{message: "test error"}
}()

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

