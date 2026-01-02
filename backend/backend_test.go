package backend

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestBackend_Factory(t *testing.T) {
	config := &logical.BackendConfig{
		Logger: nil,
		System: &logical.StaticSystemView{},
	}

	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	if b == nil {
		t.Fatal("backend is nil")
	}
}

func TestBackend_Invalidate(t *testing.T) {
	config := &logical.BackendConfig{
		Logger: nil,
		System: &logical.StaticSystemView{},
	}

	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	backend := b.(*skyflowBackend)

	t.Run("Invalidate config", func(t *testing.T) {
		// Should not panic
		backend.invalidate(context.Background(), "config")
	})

	t.Run("Invalidate role", func(t *testing.T) {
		// Should not panic
		backend.invalidate(context.Background(), "role/testrole")
	})
}

func TestBackend_Cleanup(t *testing.T) {
	config := &logical.BackendConfig{
		Logger: nil,
		System: &logical.StaticSystemView{},
	}

	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	backend := b.(*skyflowBackend)

	// Should not panic
	backend.cleanup(context.Background())
}

func TestBackend_Metrics(t *testing.T) {
	config := &logical.BackendConfig{
		Logger: nil,
		System: &logical.StaticSystemView{},
	}

	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	backend := b.(*skyflowBackend)

	t.Run("Initial metrics", func(t *testing.T) {
		stats := backend.metrics.getStats()

		if stats["total_requests"].(uint64) != 0 {
			t.Errorf("expected 0 total requests, got %v", stats["total_requests"])
		}

		if stats["token_generations"].(uint64) != 0 {
			t.Errorf("expected 0 token generations, got %v", stats["token_generations"])
		}
	})

	t.Run("Reset metrics", func(t *testing.T) {
		backend.metrics.reset()

		stats := backend.metrics.getStats()
		if stats["total_requests"].(uint64) != 0 {
			t.Errorf("expected 0 total requests after reset, got %v", stats["total_requests"])
		}
	})
}

func TestBackend_CircuitBreaker(t *testing.T) {
	config := &logical.BackendConfig{
		Logger: nil,
		System: &logical.StaticSystemView{},
	}

	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	backend := b.(*skyflowBackend)

	t.Run("Initial state", func(t *testing.T) {
		state := backend.circuitBreaker.getState()
		if state != "closed" {
			t.Errorf("expected initial state 'closed', got '%s'", state)
		}
	})

	t.Run("Reset circuit breaker", func(t *testing.T) {
		backend.circuitBreaker.reset()

		state := backend.circuitBreaker.getState()
		if state != "closed" {
			t.Errorf("expected state 'closed' after reset, got '%s'", state)
		}
	})

	t.Run("Get stats", func(t *testing.T) {
		stats := backend.circuitBreaker.getStats()

		if stats["state"] != "closed" {
			t.Errorf("expected state 'closed', got '%s'", stats["state"])
		}

		if stats["failures"].(int) != 0 {
			t.Errorf("expected 0 failures, got %v", stats["failures"])
		}
	})
}
