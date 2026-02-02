package telemetry

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestTracesProvider_NilSafety(t *testing.T) {
	var nilProvider *TracesProvider

	// All methods should be safe to call on nil receiver
	if nilProvider.IsEnabled() {
		t.Error("IsEnabled() on nil should return false")
	}

	// Start methods should return valid (noop) context and span
	ctx, span := nilProvider.StartTokenGenerate(context.Background(), "test")
	if ctx == nil {
		t.Error("StartTokenGenerate() returned nil context")
	}
	if span == nil {
		t.Error("StartTokenGenerate() returned nil span")
	}
	span.End()

	ctx, span = nilProvider.StartSDKAuth(context.Background(), "test", "json", 1)
	if ctx == nil {
		t.Error("StartSDKAuth() returned nil context")
	}
	if span == nil {
		t.Error("StartSDKAuth() returned nil span")
	}
	span.End()

	ctx, span = nilProvider.StartConfigWrite(context.Background(), "create")
	if ctx == nil {
		t.Error("StartConfigWrite() returned nil context")
	}
	if span == nil {
		t.Error("StartConfigWrite() returned nil span")
	}
	span.End()

	ctx, span = nilProvider.StartConfigRead(context.Background())
	if ctx == nil {
		t.Error("StartConfigRead() returned nil context")
	}
	if span == nil {
		t.Error("StartConfigRead() returned nil span")
	}
	span.End()

	ctx, span = nilProvider.StartRoleWrite(context.Background(), "test", "create")
	if ctx == nil {
		t.Error("StartRoleWrite() returned nil context")
	}
	if span == nil {
		t.Error("StartRoleWrite() returned nil span")
	}
	span.End()

	ctx, span = nilProvider.StartRoleRead(context.Background(), "test")
	if ctx == nil {
		t.Error("StartRoleRead() returned nil context")
	}
	if span == nil {
		t.Error("StartRoleRead() returned nil span")
	}
	span.End()

	ctx, span = nilProvider.StartRoleList(context.Background())
	if ctx == nil {
		t.Error("StartRoleList() returned nil context")
	}
	if span == nil {
		t.Error("StartRoleList() returned nil span")
	}
	span.End()

	ctx, span = nilProvider.StartRoleDelete(context.Background(), "test")
	if ctx == nil {
		t.Error("StartRoleDelete() returned nil context")
	}
	if span == nil {
		t.Error("StartRoleDelete() returned nil span")
	}
	span.End()

	ctx, span = nilProvider.StartHealthCheck(context.Background())
	if ctx == nil {
		t.Error("StartHealthCheck() returned nil context")
	}
	if span == nil {
		t.Error("StartHealthCheck() returned nil span")
	}
	span.End()

	// Record methods should not panic on nil
	nilProvider.RecordSDKAuthSuccess(nil, 100)
	nilProvider.RecordSDKAuthFailed(nil, 100, errors.New("test"))
	nilProvider.RecordTokenGenerated(nil, 100)
	nilProvider.RecordTokenFailed(nil, 100, errors.New("test"))
	nilProvider.RecordConfigUpdated(nil)
	nilProvider.RecordConfigFound(nil, true)
	nilProvider.RecordConfigError(nil, errors.New("test"))
	nilProvider.RecordConfigErrorWithMessage(nil, "test")
	nilProvider.RecordRoleUpdated(nil)
	nilProvider.RecordRoleFound(nil, true)
	nilProvider.RecordRoleDeleted(nil)
	nilProvider.RecordRoleError(nil, errors.New("test"))
	nilProvider.RecordRoleErrorWithMessage(nil, "test")
	nilProvider.RecordRoleListSuccess(nil)
	nilProvider.RecordHealthCheckSuccess(nil)
	nilProvider.RecordHealthCheckNotConfigured(nil)
	nilProvider.RecordHealthCheckError(nil, errors.New("test"))
}

func TestTracesProvider_DisabledSafety(t *testing.T) {
	provider := &TracesProvider{
		enabled: false,
	}

	if provider.IsEnabled() {
		t.Error("IsEnabled() should return false for disabled provider")
	}

	// Start methods should return valid (noop) context and span
	ctx, span := provider.StartTokenGenerate(context.Background(), "test")
	if ctx == nil {
		t.Error("StartTokenGenerate() returned nil context")
	}
	if span == nil {
		t.Error("StartTokenGenerate() returned nil span")
	}
	span.End()

	// Record methods should not panic on disabled provider
	provider.RecordSDKAuthSuccess(span, 100)
	provider.RecordTokenGenerated(span, 100)
	provider.RecordConfigUpdated(span)
}

func TestTracesProvider_EnabledMethods(t *testing.T) {
	provider := newTracesProvider(true)

	if !provider.IsEnabled() {
		t.Error("IsEnabled() should return true for enabled provider")
	}

	// Tracer should not be nil
	if provider.Tracer() == nil {
		t.Error("Tracer() returned nil")
	}
}

func TestTracesProvider_StartMethods_CreateSpans(t *testing.T) {
	provider := newTracesProvider(true)

	tests := []struct {
		name   string
		startF func() (context.Context, trace.Span)
	}{
		{
			name: "StartTokenGenerate",
			startF: func() (context.Context, trace.Span) {
				return provider.StartTokenGenerate(context.Background(), "test-role")
			},
		},
		{
			name: "StartSDKAuth",
			startF: func() (context.Context, trace.Span) {
				return provider.StartSDKAuth(context.Background(), "test-role", "json", 2)
			},
		},
		{
			name: "StartConfigWrite",
			startF: func() (context.Context, trace.Span) {
				return provider.StartConfigWrite(context.Background(), "create")
			},
		},
		{
			name: "StartConfigRead",
			startF: func() (context.Context, trace.Span) {
				return provider.StartConfigRead(context.Background())
			},
		},
		{
			name: "StartRoleWrite",
			startF: func() (context.Context, trace.Span) {
				return provider.StartRoleWrite(context.Background(), "test-role", "create")
			},
		},
		{
			name: "StartRoleRead",
			startF: func() (context.Context, trace.Span) {
				return provider.StartRoleRead(context.Background(), "test-role")
			},
		},
		{
			name: "StartRoleList",
			startF: func() (context.Context, trace.Span) {
				return provider.StartRoleList(context.Background())
			},
		},
		{
			name: "StartRoleDelete",
			startF: func() (context.Context, trace.Span) {
				return provider.StartRoleDelete(context.Background(), "test-role")
			},
		},
		{
			name: "StartHealthCheck",
			startF: func() (context.Context, trace.Span) {
				return provider.StartHealthCheck(context.Background())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, span := tt.startF()
			if ctx == nil {
				t.Errorf("%s() returned nil context", tt.name)
			}
			if span == nil {
				t.Errorf("%s() returned nil span", tt.name)
			}
			span.End()
		})
	}
}

func TestTracesProvider_RecordMethods_NoErrors(t *testing.T) {
	provider := newTracesProvider(true)
	ctx, span := provider.StartTokenGenerate(context.Background(), "test-role")
	defer span.End()

	// All record methods should not panic
	testErr := errors.New("test error")

	// SDK auth records
	provider.RecordSDKAuthSuccess(span, 100.0)
	provider.RecordSDKAuthFailed(span, 100.0, testErr)

	// Token records
	provider.RecordTokenGenerated(span, 100.0)
	provider.RecordTokenFailed(span, 100.0, testErr)

	// Config records
	provider.RecordConfigUpdated(span)
	provider.RecordConfigFound(span, true)
	provider.RecordConfigFound(span, false)
	provider.RecordConfigError(span, testErr)
	provider.RecordConfigErrorWithMessage(span, "test message")

	// Role records
	provider.RecordRoleUpdated(span)
	provider.RecordRoleFound(span, true)
	provider.RecordRoleFound(span, false)
	provider.RecordRoleDeleted(span)
	provider.RecordRoleError(span, testErr)
	provider.RecordRoleErrorWithMessage(span, "test message")
	provider.RecordRoleListSuccess(span)

	// Health check records
	provider.RecordHealthCheckSuccess(span)
	provider.RecordHealthCheckNotConfigured(span)
	provider.RecordHealthCheckError(span, testErr)

	// SpanFromContext should work
	spanFromCtx := provider.SpanFromContext(ctx)
	if spanFromCtx == nil {
		t.Error("SpanFromContext() returned nil")
	}
}

func TestTracesProvider_RecordError_NilError(t *testing.T) {
	provider := newTracesProvider(true)
	_, span := provider.StartTokenGenerate(context.Background(), "test-role")
	defer span.End()

	// Should not panic with nil error
	provider.RecordSDKAuthFailed(span, 100.0, nil)
	provider.RecordTokenFailed(span, 100.0, nil)
	provider.RecordConfigError(span, nil)
	provider.RecordRoleError(span, nil)
	provider.RecordHealthCheckError(span, nil)
}

func TestNoopSpan(t *testing.T) {
	ctx, span := noopSpan(context.Background())
	if ctx == nil {
		t.Error("noopSpan returned nil context")
	}
	if span == nil {
		t.Error("noopSpan returned nil span")
	}
	span.End()
}

func TestTracesProvider_Tracer_NilReceiver(t *testing.T) {
	var nilProvider *TracesProvider
	tracer := nilProvider.Tracer()
	if tracer == nil {
		t.Error("Tracer() on nil receiver should return a valid tracer")
	}
}
