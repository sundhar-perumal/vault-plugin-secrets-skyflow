package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNoOp_Implementation(t *testing.T) {
	noop := NewNoOp()

	if noop == nil {
		t.Fatal("NoOp should not be nil")
	}

	// Ensure it implements Telemetry interface (compile-time check)
	var _ Telemetry = noop
}

func TestNoOp_OnTokenRequest(t *testing.T) {
	noop := NewNoOp()
	ctx := context.Background()

	result := noop.OnTokenRequest(ctx, TokenRequestEvent{
		RoleName:  "test-role",
		Timestamp: time.Now(),
	})

	if result != ctx {
		t.Error("OnTokenRequest should return the same context")
	}
}

func TestNoOp_OnTokenGenerate(t *testing.T) {
	noop := NewNoOp()

	// Should not panic
	noop.OnTokenGenerate(context.Background(), TokenGenerateEvent{
		RoleName:  "test-role",
		Success:   true,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now(),
	})
}

func TestNoOp_OnConfigWrite(t *testing.T) {
	noop := NewNoOp()
	ctx := context.Background()

	result := noop.OnConfigWrite(ctx, ConfigWriteEvent{
		Operation: "create",
		Success:   true,
		Timestamp: time.Now(),
	})

	if result != ctx {
		t.Error("OnConfigWrite should return the same context")
	}
}

func TestNoOp_OnConfigRead(t *testing.T) {
	noop := NewNoOp()

	// Should not panic
	noop.OnConfigRead(context.Background(), ConfigReadEvent{
		Found:     true,
		Timestamp: time.Now(),
	})
}

func TestNoOp_OnRoleWrite(t *testing.T) {
	noop := NewNoOp()
	ctx := context.Background()

	result := noop.OnRoleWrite(ctx, RoleWriteEvent{
		RoleName:  "test-role",
		Operation: "create",
		Success:   true,
		Timestamp: time.Now(),
	})

	if result != ctx {
		t.Error("OnRoleWrite should return the same context")
	}
}

func TestNoOp_OnRoleRead(t *testing.T) {
	noop := NewNoOp()

	// Should not panic
	noop.OnRoleRead(context.Background(), RoleReadEvent{
		RoleName:  "test-role",
		Found:     true,
		Timestamp: time.Now(),
	})
}

func TestNoOp_OnError(t *testing.T) {
	noop := NewNoOp()

	// Should not panic
	noop.OnError(context.Background(), ErrorEvent{
		Operation: "test",
		Error:     errors.New("test error"),
		Severity:  "error",
		Timestamp: time.Now(),
	})
}

func TestNoOp_EndSpan(t *testing.T) {
	noop := NewNoOp()

	// Should not panic
	noop.EndSpan(context.Background())
}

