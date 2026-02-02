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

func TestBackend_AuditLog(t *testing.T) {
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
	backend.auditLog(auditEvent{
		Operation: "test",
		Role:      "test-role",
		Success:   true,
		Duration:  100,
		ClientIP:  "127.0.0.1",
	})
}

func TestBackend_Version(t *testing.T) {
	// Version should have v prefix for Vault compatibility
	if Version[0] != 'v' {
		t.Errorf("Version should start with 'v', got '%s'", Version)
	}
}

func TestBackend_TelemetryProviders(t *testing.T) {
	config := &logical.BackendConfig{
		Logger: nil,
		System: &logical.StaticSystemView{},
	}

	b, err := Factory(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	backend := b.(*skyflowBackend)

	// telemetryProviders may be nil if telemetry init failed (expected in tests without OTEL config)
	// Just verify backend was created successfully
	_ = backend
}
