package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestSpanTelemetry_OnTokenRequest(t *testing.T) {
	// With nil provider
	st := NewSpanTelemetry(SpanConfig{Provider: nil})

	ctx := st.OnTokenRequest(context.Background(), TokenRequestEvent{
		RoleName:  "test-role",
		Timestamp: time.Now(),
	})

	if ctx == nil {
		t.Error("context should not be nil")
	}
}

func TestSpanTelemetry_OnTokenGenerate(t *testing.T) {
	st := NewSpanTelemetry(SpanConfig{Provider: nil})

	// Should not panic with nil provider
	st.OnTokenGenerate(context.Background(), TokenGenerateEvent{
		RoleName:  "test-role",
		Success:   true,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now(),
	})
}

func TestSpanTelemetry_OnConfigWrite(t *testing.T) {
	st := NewSpanTelemetry(SpanConfig{Provider: nil})

	ctx := st.OnConfigWrite(context.Background(), ConfigWriteEvent{
		Operation: "create",
		Success:   true,
		Timestamp: time.Now(),
	})

	if ctx == nil {
		t.Error("context should not be nil")
	}
}

func TestSpanTelemetry_OnRoleWrite(t *testing.T) {
	st := NewSpanTelemetry(SpanConfig{Provider: nil})

	ctx := st.OnRoleWrite(context.Background(), RoleWriteEvent{
		RoleName:  "test-role",
		Operation: "create",
		Success:   true,
		Timestamp: time.Now(),
	})

	if ctx == nil {
		t.Error("context should not be nil")
	}
}

func TestSpanTelemetry_OnError(t *testing.T) {
	st := NewSpanTelemetry(SpanConfig{Provider: nil})

	// Should not panic with nil provider
	st.OnError(context.Background(), ErrorEvent{
		Operation: "test",
		Severity:  "error",
		Timestamp: time.Now(),
	})
}

func TestSpanTelemetry_EndSpan(t *testing.T) {
	st := NewSpanTelemetry(SpanConfig{Provider: nil})

	// Should not panic
	st.EndSpan(context.Background())
}

