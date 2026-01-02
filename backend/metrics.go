package backend

import (
	"sync"
	"time"
)

// metrics tracks plugin performance metrics
type metrics struct {
	tokenGenerations  uint64
	tokenErrors       uint64
	totalResponseTime time.Duration
	requestCount      uint64
	mu                sync.RWMutex
}

// newMetrics creates a new metrics instance
func newMetrics() *metrics {
	return &metrics{}
}

// recordTokenGeneration records metrics for a token generation
func (m *metrics) recordTokenGeneration(duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestCount++
	m.totalResponseTime += duration

	if err != nil {
		m.tokenErrors++
	} else {
		m.tokenGenerations++
	}
}

// getStats returns current metrics statistics
func (m *metrics) getStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var avgResponseTime float64
	if m.requestCount > 0 {
		avgResponseTime = float64(m.totalResponseTime.Milliseconds()) / float64(m.requestCount)
	}

	var errorRate float64
	if m.requestCount > 0 {
		errorRate = float64(m.tokenErrors) / float64(m.requestCount)
	}

	return map[string]interface{}{
		"total_requests":       m.requestCount,
		"token_generations":    m.tokenGenerations,
		"token_errors":         m.tokenErrors,
		"avg_response_time_ms": avgResponseTime,
		"error_rate":           errorRate,
	}
}

// reset clears all metrics (useful for testing)
func (m *metrics) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokenGenerations = 0
	m.tokenErrors = 0
	m.totalResponseTime = 0
	m.requestCount = 0
}
