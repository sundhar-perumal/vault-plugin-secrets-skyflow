package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Telemetry defines the interface for instrumentation using OpenTelemetry Tracing.
type Telemetry interface {
	// Token operations
	OnTokenRequest(ctx context.Context, event TokenRequestEvent) context.Context
	OnTokenGenerate(ctx context.Context, event TokenGenerateEvent)

	// Config operations
	OnConfigWrite(ctx context.Context, event ConfigWriteEvent) context.Context
	OnConfigRead(ctx context.Context, event ConfigReadEvent)

	// Role operations
	OnRoleWrite(ctx context.Context, event RoleWriteEvent) context.Context
	OnRoleRead(ctx context.Context, event RoleReadEvent)

	// Error events
	OnError(ctx context.Context, event ErrorEvent)

	// Span management
	EndSpan(ctx context.Context)
}

// ============================================================================
// Event Types
// ============================================================================

// TokenRequestEvent is emitted when a token is requested
type TokenRequestEvent struct {
	RoleName  string    // Role name
	Timestamp time.Time // Event timestamp
}

// TokenGenerateEvent is emitted when a token is generated
type TokenGenerateEvent struct {
	RoleName  string        // Role name
	Success   bool          // Whether generation succeeded
	Duration  time.Duration // Time taken to generate
	Error     error         // Error if failed
	Timestamp time.Time     // Event timestamp
}

// ConfigWriteEvent is emitted when config is written
type ConfigWriteEvent struct {
	Operation string    // "create" or "update"
	Success   bool      // Whether write succeeded
	Timestamp time.Time // Event timestamp
}

// ConfigReadEvent is emitted when config is read
type ConfigReadEvent struct {
	Found     bool      // Whether config was found
	Timestamp time.Time // Event timestamp
}

// RoleWriteEvent is emitted when a role is written
type RoleWriteEvent struct {
	RoleName  string    // Role name
	Operation string    // "create" or "update"
	Success   bool      // Whether write succeeded
	Timestamp time.Time // Event timestamp
}

// RoleReadEvent is emitted when a role is read
type RoleReadEvent struct {
	RoleName  string    // Role name
	Found     bool      // Whether role was found
	Timestamp time.Time // Event timestamp
}

// ErrorEvent is emitted on any error
type ErrorEvent struct {
	Operation string    // Operation that failed
	Error     error     // The error
	Severity  string    // "warning", "error", "critical"
	Timestamp time.Time // Event timestamp
}

// ============================================================================
// SpanTelemetry Implementation (using OpenTelemetry Tracing)
// ============================================================================

// SpanTelemetry implements the Telemetry interface using OTEL Tracing.
type SpanTelemetry struct {
	provider *TracerProvider
}

// SpanConfig configures the SpanTelemetry
type SpanConfig struct {
	Provider *TracerProvider
}

// NewSpanTelemetry creates a new SpanTelemetry implementation
func NewSpanTelemetry(config SpanConfig) *SpanTelemetry {
	return &SpanTelemetry{
		provider: config.Provider,
	}
}

// ============================================================================
// Token Operations Implementation
// ============================================================================

// OnTokenRequest creates a span for token request
func (s *SpanTelemetry) OnTokenRequest(ctx context.Context, event TokenRequestEvent) context.Context {
	if s.provider == nil || !s.provider.IsEnabled() {
		return ctx
	}

	ctx, _ = s.provider.StartSpan(ctx, SpanTokenGenerate,
		AttrRole.String(event.RoleName),
	)

	return ctx
}

// OnTokenGenerate adds token generation result to current span
func (s *SpanTelemetry) OnTokenGenerate(ctx context.Context, event TokenGenerateEvent) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.SetAttributes(
		attribute.Float64("duration_ms", float64(event.Duration.Milliseconds())),
	)

	if event.Success {
		span.AddEvent(EventTokenGenerated)
		span.SetStatus(codes.Ok, "")
	} else {
		span.AddEvent(EventError)
		if event.Error != nil {
			span.RecordError(event.Error)
			span.SetStatus(codes.Error, event.Error.Error())
		}
	}
}

// ============================================================================
// Config Operations Implementation
// ============================================================================

// OnConfigWrite creates a span for config write
func (s *SpanTelemetry) OnConfigWrite(ctx context.Context, event ConfigWriteEvent) context.Context {
	if s.provider == nil || !s.provider.IsEnabled() {
		return ctx
	}

	ctx, span := s.provider.StartSpan(ctx, SpanConfigWrite,
		attribute.String("operation", event.Operation),
	)

	if event.Success {
		span.SetStatus(codes.Ok, "")
	}

	span.End()
	return ctx
}

// OnConfigRead adds config read result
func (s *SpanTelemetry) OnConfigRead(ctx context.Context, event ConfigReadEvent) {
	if s.provider == nil || !s.provider.IsEnabled() {
		return
	}

	_, span := s.provider.StartSpan(ctx, SpanConfigRead,
		attribute.Bool("found", event.Found),
	)
	span.End()
}

// ============================================================================
// Role Operations Implementation
// ============================================================================

// OnRoleWrite creates a span for role write
func (s *SpanTelemetry) OnRoleWrite(ctx context.Context, event RoleWriteEvent) context.Context {
	if s.provider == nil || !s.provider.IsEnabled() {
		return ctx
	}

	ctx, span := s.provider.StartSpan(ctx, SpanRoleWrite,
		AttrRole.String(event.RoleName),
		attribute.String("operation", event.Operation),
	)

	if event.Success {
		span.SetStatus(codes.Ok, "")
	}

	span.End()
	return ctx
}

// OnRoleRead adds role read result
func (s *SpanTelemetry) OnRoleRead(ctx context.Context, event RoleReadEvent) {
	if s.provider == nil || !s.provider.IsEnabled() {
		return
	}

	_, span := s.provider.StartSpan(ctx, SpanRoleRead,
		AttrRole.String(event.RoleName),
		attribute.Bool("found", event.Found),
	)
	span.End()
}

// ============================================================================
// Error Implementation
// ============================================================================

// OnError records an error in the current span
func (s *SpanTelemetry) OnError(ctx context.Context, event ErrorEvent) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.AddEvent(EventError,
		trace.WithAttributes(
			AttrErrorOperation.String(event.Operation),
			AttrErrorSeverity.String(event.Severity),
		),
	)

	if event.Error != nil {
		span.RecordError(event.Error)
		if event.Severity == "critical" || event.Severity == "error" {
			span.SetStatus(codes.Error, event.Error.Error())
		}
	}
}

// ============================================================================
// Span Management
// ============================================================================

// EndSpan ends the current span in context
func (s *SpanTelemetry) EndSpan(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.End()
	}
}

// Ensure SpanTelemetry implements Telemetry interface
var _ Telemetry = (*SpanTelemetry)(nil)

// ============================================================================
// Constants
// ============================================================================

const (
	// TracerName is the instrumentation name for this library
	TracerName = "github.com/sundhar-perumal/vault-plugin-secrets-skyflow"

	// Span names
	SpanTokenGenerate = "skyflow.token.generate"
	SpanConfigWrite   = "skyflow.config.write"
	SpanConfigRead    = "skyflow.config.read"
	SpanRoleWrite     = "skyflow.role.write"
	SpanRoleRead      = "skyflow.role.read"
	SpanHealthCheck   = "skyflow.health.check"

	// Event names
	EventTokenGenerated = "token.generated"
	EventConfigUpdated  = "config.updated"
	EventRoleUpdated    = "role.updated"
	EventError          = "error"
)

// ============================================================================
// Common Attribute Keys
// ============================================================================

var (
	// Role attributes
	AttrRole = attribute.Key("skyflow.role")

	// Error attributes
	AttrErrorOperation = attribute.Key("error.operation")
	AttrErrorSeverity  = attribute.Key("error.severity")
)

