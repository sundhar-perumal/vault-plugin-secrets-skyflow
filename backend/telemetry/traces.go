package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracesProvider wraps trace operations with semantic domain methods.
// All methods are nil-safe - they handle nil receiver gracefully (no-op when telemetry disabled).
type TracesProvider struct {
	tracer  trace.Tracer
	enabled bool
}

// newTracesProvider creates a TracesProvider
func newTracesProvider(enabled bool) *TracesProvider {
	return &TracesProvider{
		tracer:  otel.Tracer(TracerName),
		enabled: enabled,
	}
}

// IsEnabled returns whether tracing is enabled
func (t *TracesProvider) IsEnabled() bool {
	return t != nil && t.enabled
}

// Tracer returns the underlying OTEL tracer
func (t *TracesProvider) Tracer() trace.Tracer {
	if t == nil {
		return otel.Tracer(TracerName)
	}
	return t.tracer
}

// ============================================================================
// Start Methods - Token Operations
// ============================================================================

// StartTokenGenerate starts a span for token generation
func (t *TracesProvider) StartTokenGenerate(ctx context.Context, roleName string) (context.Context, trace.Span) {
	if !t.IsEnabled() {
		return noopSpan(ctx)
	}
	return t.tracer.Start(ctx, SpanSkyflowPluginTokenGenerate, trace.WithAttributes(
		AttrRole.String(roleName),
	))
}

// StartSDKAuth starts a span for Skyflow SDK authentication
func (t *TracesProvider) StartSDKAuth(ctx context.Context, roleName, credentialType string, roleIDsCount int) (context.Context, trace.Span) {
	if !t.IsEnabled() {
		return noopSpan(ctx)
	}
	ctx, span := t.tracer.Start(ctx, SpanSkyflowPluginSDKAuth,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			AttrRole.String(roleName),
			AttrCredentialType.String(credentialType),
			AttrRoleIDsCount.Int(roleIDsCount),
		),
	)
	span.AddEvent(EventSDKAuthStart)
	return ctx, span
}

// ============================================================================
// Start Methods - Config Operations
// ============================================================================

// StartConfigWrite starts a span for config write operation
func (t *TracesProvider) StartConfigWrite(ctx context.Context, operation string) (context.Context, trace.Span) {
	if !t.IsEnabled() {
		return noopSpan(ctx)
	}
	return t.tracer.Start(ctx, SpanSkyflowPluginConfigWrite, trace.WithAttributes(
		AttrOperation.String(operation),
	))
}

// StartConfigRead starts a span for config read operation
func (t *TracesProvider) StartConfigRead(ctx context.Context) (context.Context, trace.Span) {
	if !t.IsEnabled() {
		return noopSpan(ctx)
	}
	return t.tracer.Start(ctx, SpanSkyflowPluginConfigRead)
}

// ============================================================================
// Start Methods - Role Operations
// ============================================================================

// StartRoleWrite starts a span for role write operation
func (t *TracesProvider) StartRoleWrite(ctx context.Context, name, operation string) (context.Context, trace.Span) {
	if !t.IsEnabled() {
		return noopSpan(ctx)
	}
	return t.tracer.Start(ctx, SpanSkyflowPluginRoleWrite, trace.WithAttributes(
		AttrRole.String(name),
		AttrOperation.String(operation),
	))
}

// StartRoleRead starts a span for role read operation
func (t *TracesProvider) StartRoleRead(ctx context.Context, name string) (context.Context, trace.Span) {
	if !t.IsEnabled() {
		return noopSpan(ctx)
	}
	return t.tracer.Start(ctx, SpanSkyflowPluginRoleRead, trace.WithAttributes(
		AttrRole.String(name),
	))
}

// StartRoleList starts a span for role list operation
func (t *TracesProvider) StartRoleList(ctx context.Context) (context.Context, trace.Span) {
	if !t.IsEnabled() {
		return noopSpan(ctx)
	}
	return t.tracer.Start(ctx, SpanSkyflowPluginRoleList)
}

// StartRoleDelete starts a span for role delete operation
func (t *TracesProvider) StartRoleDelete(ctx context.Context, name string) (context.Context, trace.Span) {
	if !t.IsEnabled() {
		return noopSpan(ctx)
	}
	return t.tracer.Start(ctx, SpanSkyflowPluginRoleDelete, trace.WithAttributes(
		AttrRole.String(name),
	))
}

// ============================================================================
// Start Methods - Health Check
// ============================================================================

// StartHealthCheck starts a span for health check
func (t *TracesProvider) StartHealthCheck(ctx context.Context) (context.Context, trace.Span) {
	if !t.IsEnabled() {
		return noopSpan(ctx)
	}
	return t.tracer.Start(ctx, SpanSkyflowPluginHealthCheck)
}

// ============================================================================
// Record Methods - SDK Auth Events
// ============================================================================

// RecordSDKAuthSuccess records SDK auth success
func (t *TracesProvider) RecordSDKAuthSuccess(span trace.Span, durationMs float64) {
	t.setAttributes(span, AttrSDKDurationMs.Float64(durationMs))
	t.addEvent(span, EventSDKAuthSuccess)
	t.setOK(span)
}

// RecordSDKAuthFailed records SDK auth failure
func (t *TracesProvider) RecordSDKAuthFailed(span trace.Span, durationMs float64, err error) {
	t.setAttributes(span, AttrSDKDurationMs.Float64(durationMs))
	t.addEvent(span, EventSDKAuthFailed)
	t.recordError(span, err)
}

// ============================================================================
// Record Methods - Token Events
// ============================================================================

// RecordTokenGenerated records successful token generation
func (t *TracesProvider) RecordTokenGenerated(span trace.Span, durationMs float64) {
	t.addEvent(span, EventTokenGenerated, AttrDurationMs.Float64(durationMs))
	t.setOK(span)
}

// RecordTokenFailed records token generation failure
func (t *TracesProvider) RecordTokenFailed(span trace.Span, durationMs float64, err error) {
	t.addEvent(span, EventTokenFailed, AttrDurationMs.Float64(durationMs))
	t.recordError(span, err)
}

// ============================================================================
// Record Methods - Config Events
// ============================================================================

// RecordConfigUpdated records config updated event
func (t *TracesProvider) RecordConfigUpdated(span trace.Span) {
	t.addEvent(span, EventConfigUpdated)
	t.setOK(span)
}

// RecordConfigFound records config found with attribute
func (t *TracesProvider) RecordConfigFound(span trace.Span, found bool) {
	t.setAttributes(span, AttrFound.Bool(found))
	t.setOK(span)
}

// RecordConfigError records config operation failure
func (t *TracesProvider) RecordConfigError(span trace.Span, err error) {
	t.recordError(span, err)
}

// RecordConfigErrorWithMessage records config operation failure with custom message
func (t *TracesProvider) RecordConfigErrorWithMessage(span trace.Span, message string) {
	if !t.IsEnabled() || span == nil || !span.IsRecording() {
		return
	}
	span.SetStatus(codes.Error, message)
}

// ============================================================================
// Record Methods - Role Events
// ============================================================================

// RecordRoleUpdated records role updated event
func (t *TracesProvider) RecordRoleUpdated(span trace.Span) {
	t.addEvent(span, EventRoleUpdated)
	t.setOK(span)
}

// RecordRoleFound records role found with attribute
func (t *TracesProvider) RecordRoleFound(span trace.Span, found bool) {
	t.setAttributes(span, AttrFound.Bool(found))
	t.setOK(span)
}

// RecordRoleDeleted records role deletion success
func (t *TracesProvider) RecordRoleDeleted(span trace.Span) {
	t.setOK(span)
}

// RecordRoleError records role operation failure
func (t *TracesProvider) RecordRoleError(span trace.Span, err error) {
	t.recordError(span, err)
}

// RecordRoleErrorWithMessage records role operation failure with custom message
func (t *TracesProvider) RecordRoleErrorWithMessage(span trace.Span, message string) {
	if !t.IsEnabled() || span == nil || !span.IsRecording() {
		return
	}
	span.SetStatus(codes.Error, message)
}

// RecordRoleListSuccess records role list success
func (t *TracesProvider) RecordRoleListSuccess(span trace.Span) {
	t.setOK(span)
}

// ============================================================================
// Record Methods - Health Check Events
// ============================================================================

// RecordHealthCheckSuccess records successful health check
func (t *TracesProvider) RecordHealthCheckSuccess(span trace.Span) {
	t.setOK(span)
}

// RecordHealthCheckNotConfigured records health check when not configured
func (t *TracesProvider) RecordHealthCheckNotConfigured(span trace.Span) {
	if !t.IsEnabled() || span == nil || !span.IsRecording() {
		return
	}
	span.SetStatus(codes.Ok, StatusNotConfigured)
}

// RecordHealthCheckError records health check failure
func (t *TracesProvider) RecordHealthCheckError(span trace.Span, err error) {
	t.recordError(span, err)
}

// ============================================================================
// Utility Methods
// ============================================================================

// SpanFromContext returns the span from context
func (t *TracesProvider) SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ============================================================================
// Internal helper methods
// ============================================================================

// noopSpan returns a noop span when tracing is disabled
func noopSpan(ctx context.Context) (context.Context, trace.Span) {
	return trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "")
}

// addEvent adds an event to the span with optional attributes
func (t *TracesProvider) addEvent(span trace.Span, event string, attrs ...attribute.KeyValue) {
	if !t.IsEnabled() || span == nil || !span.IsRecording() {
		return
	}
	if len(attrs) > 0 {
		span.AddEvent(event, trace.WithAttributes(attrs...))
	} else {
		span.AddEvent(event)
	}
}

// setAttributes sets attributes on the span
func (t *TracesProvider) setAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if !t.IsEnabled() || span == nil || !span.IsRecording() || len(attrs) == 0 {
		return
	}
	span.SetAttributes(attrs...)
}

// setOK sets the span status to OK
func (t *TracesProvider) setOK(span trace.Span) {
	if !t.IsEnabled() || span == nil || !span.IsRecording() {
		return
	}
	span.SetStatus(codes.Ok, "")
}

// recordError records an error on the span and sets error status
func (t *TracesProvider) recordError(span trace.Span, err error) {
	if !t.IsEnabled() || span == nil || !span.IsRecording() || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
