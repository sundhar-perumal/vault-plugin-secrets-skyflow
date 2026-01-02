package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewEmitter_Disabled(t *testing.T) {
	cfg := &Config{
		Enabled:       false,
		ServiceName:   "test-service",
		Environment:   "test",
		TracesEnabled: true,
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if emitter == nil {
		t.Fatal("emitter should not be nil")
	}

	if emitter.IsTracesEnabled() {
		t.Error("traces should be disabled when master switch is off")
	}

	if emitter.IsMetricsEnabled() {
		t.Error("metrics should be disabled when master switch is off")
	}

	if emitter.IsEnabled() {
		t.Error("emitter should report as disabled")
	}
}

func TestNewEmitter_NoEndpoints(t *testing.T) {
	cfg := &Config{
		Enabled:        true,
		ServiceName:    "test-service",
		Environment:    "test",
		TracesEnabled:  true,
		MetricsEnabled: true,
		// No endpoints set
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without endpoints, providers won't be created
	if emitter.IsTracesEnabled() {
		t.Error("traces should be disabled without endpoint")
	}

	if emitter.IsMetricsEnabled() {
		t.Error("metrics should be disabled without endpoint")
	}
}

func TestEmitter_EmitTokenRequest(t *testing.T) {
	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
		Environment: "test",
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := emitter.EmitTokenRequest(context.Background(), "test-role")

	if ctx == nil {
		t.Error("context should not be nil")
	}
}

func TestEmitter_EmitTokenSuccess(t *testing.T) {
	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
		Environment: "test",
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic
	emitter.EmitTokenSuccess(context.Background(), "test-role", 100*time.Millisecond)
}

func TestEmitter_EmitTokenFailure(t *testing.T) {
	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
		Environment: "test",
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic
	emitter.EmitTokenFailure(context.Background(), "test-role", errors.New("test error"), 100*time.Millisecond)
}

func TestEmitter_EndTokenSpan(t *testing.T) {
	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
		Environment: "test",
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic
	emitter.EndTokenSpan(context.Background())
}

func TestEmitter_EmitConfigWrite(t *testing.T) {
	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
		Environment: "test",
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := emitter.EmitConfigWrite(context.Background(), "create", true)

	if ctx == nil {
		t.Error("context should not be nil")
	}
}

func TestEmitter_EmitRoleWrite(t *testing.T) {
	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
		Environment: "test",
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := emitter.EmitRoleWrite(context.Background(), "test-role", "create", true)

	if ctx == nil {
		t.Error("context should not be nil")
	}
}

func TestEmitter_EmitError(t *testing.T) {
	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
		Environment: "test",
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic
	emitter.EmitError(context.Background(), "test", errors.New("test error"), "error")
}

func TestEmitter_GetStartTime(t *testing.T) {
	before := time.Now()

	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
		Environment: "test",
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after := time.Now()

	startTime := emitter.GetStartTime()

	if startTime.Before(before) || startTime.After(after) {
		t.Error("start time should be between before and after")
	}
}

func TestEmitter_Close(t *testing.T) {
	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
		Environment: "test",
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic or error
	if err := emitter.Close(context.Background()); err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}

func TestEmitter_Flush(t *testing.T) {
	cfg := &Config{
		Enabled:     false,
		ServiceName: "test-service",
		Environment: "test",
	}

	emitter, err := NewEmitter(context.Background(), EmitterConfig{
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic
	emitter.Flush(context.Background())
}

