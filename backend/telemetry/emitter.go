package telemetry

import (
	"context"
	"time"
)

// Emitter wraps telemetry providers and provides simplified emit methods.
type Emitter struct {
	t           Telemetry
	tracer      *TracerProvider
	metrics     *MetricsProvider
	config      *Config
	startTime   time.Time
	serviceName string
	environment string
}

// EmitterConfig configures the Emitter
type EmitterConfig struct {
	Config         *Config
	ServiceName    string
	ServiceVersion string
	Environment    string
}

// NewEmitter creates a new Emitter with configuration
func NewEmitter(ctx context.Context, config EmitterConfig) (*Emitter, error) {
	cfg := config.Config
	if cfg == nil {
		cfg = ConfigFromEnv()
	}

	serviceName := config.ServiceName
	if serviceName == "" {
		serviceName = cfg.ServiceName
	}
	if config.ServiceVersion != "" {
		cfg.ServiceVersion = config.ServiceVersion
	}
	environment := config.Environment
	if environment == "" {
		environment = cfg.Environment
	}

	e := &Emitter{
		config:      cfg,
		startTime:   time.Now(),
		serviceName: serviceName,
		environment: environment,
	}

	// Initialize tracer provider if enabled
	if cfg.IsTracesEnabled() {
		tracerProvider, err := NewTracerProvider(ctx, cfg)
		if err == nil {
			e.tracer = tracerProvider
		}
	}

	// Initialize metrics provider if enabled
	if cfg.IsMetricsEnabled() {
		metricsProvider, err := NewMetricsProvider(ctx, cfg)
		if err == nil {
			e.metrics = metricsProvider
		}
	}

	// Create telemetry handler
	if e.tracer != nil && e.tracer.IsEnabled() {
		e.t = NewSpanTelemetry(SpanConfig{
			Provider: e.tracer,
		})
	} else {
		e.t = NewNoOp()
	}

	return e, nil
}

// Close gracefully shuts down the emitter
func (e *Emitter) Close(ctx context.Context) error {
	var err error
	if e.tracer != nil {
		if terr := e.tracer.Shutdown(ctx); terr != nil {
			err = terr
		}
	}
	if e.metrics != nil {
		if merr := e.metrics.Shutdown(ctx); merr != nil && err == nil {
			err = merr
		}
	}
	return err
}

// Flush forces an immediate flush of pending spans and metrics
func (e *Emitter) Flush(ctx context.Context) {
	if e.tracer != nil {
		_ = e.tracer.ForceFlush(ctx)
	}
	if e.metrics != nil {
		_ = e.metrics.ForceFlush(ctx)
	}
}

// IsTracesEnabled returns whether trace export is active
func (e *Emitter) IsTracesEnabled() bool {
	return e.tracer != nil && e.tracer.IsEnabled()
}

// IsMetricsEnabled returns whether metrics export is active
func (e *Emitter) IsMetricsEnabled() bool {
	return e.metrics != nil && e.metrics.IsEnabled()
}

// IsEnabled returns whether any telemetry export is active
func (e *Emitter) IsEnabled() bool {
	return e.IsTracesEnabled() || e.IsMetricsEnabled()
}

// ============================================================================
// Token Operations
// ============================================================================

// EmitTokenRequest creates a token request span
func (e *Emitter) EmitTokenRequest(ctx context.Context, roleName string) context.Context {
	return e.t.OnTokenRequest(ctx, TokenRequestEvent{
		RoleName:  roleName,
		Timestamp: time.Now(),
	})
}

// EmitTokenSuccess records successful token generation
func (e *Emitter) EmitTokenSuccess(ctx context.Context, roleName string, duration time.Duration) {
	// Record metric
	if e.IsMetricsEnabled() {
		e.metrics.RecordTokenGenerate(ctx, roleName, float64(duration.Milliseconds()), true)
	}

	e.t.OnTokenGenerate(ctx, TokenGenerateEvent{
		RoleName:  roleName,
		Success:   true,
		Duration:  duration,
		Timestamp: time.Now(),
	})
}

// EmitTokenFailure records failed token generation
func (e *Emitter) EmitTokenFailure(ctx context.Context, roleName string, err error, duration time.Duration) {
	// Record metrics
	if e.IsMetricsEnabled() {
		e.metrics.RecordTokenGenerate(ctx, roleName, float64(duration.Milliseconds()), false)
		e.metrics.RecordTokenError(ctx, roleName, "generation_failed")
	}

	e.t.OnTokenGenerate(ctx, TokenGenerateEvent{
		RoleName:  roleName,
		Success:   false,
		Duration:  duration,
		Error:     err,
		Timestamp: time.Now(),
	})
}

// EndTokenSpan ends the current token request span
func (e *Emitter) EndTokenSpan(ctx context.Context) {
	e.t.EndSpan(ctx)
}

// ============================================================================
// Config Operations
// ============================================================================

// EmitConfigWrite records a config write operation
func (e *Emitter) EmitConfigWrite(ctx context.Context, operation string, success bool) context.Context {
	// Record metric
	if e.IsMetricsEnabled() && success {
		e.metrics.RecordConfigWrite(ctx, operation)
	}

	return e.t.OnConfigWrite(ctx, ConfigWriteEvent{
		Operation: operation,
		Success:   success,
		Timestamp: time.Now(),
	})
}

// ============================================================================
// Role Operations
// ============================================================================

// EmitRoleWrite records a role write operation
func (e *Emitter) EmitRoleWrite(ctx context.Context, roleName, operation string, success bool) context.Context {
	// Record metric
	if e.IsMetricsEnabled() && success {
		e.metrics.RecordRoleWrite(ctx, roleName, operation)
	}

	return e.t.OnRoleWrite(ctx, RoleWriteEvent{
		RoleName:  roleName,
		Operation: operation,
		Success:   success,
		Timestamp: time.Now(),
	})
}

// ============================================================================
// Error Operations
// ============================================================================

// EmitError records an error
func (e *Emitter) EmitError(ctx context.Context, operation string, err error, severity string) {
	e.t.OnError(ctx, ErrorEvent{
		Operation: operation,
		Error:     err,
		Severity:  severity,
		Timestamp: time.Now(),
	})
}

// GetStartTime returns the emitter's start time
func (e *Emitter) GetStartTime() time.Time {
	return e.startTime
}

// GetMetricsProvider returns the metrics provider
func (e *Emitter) GetMetricsProvider() *MetricsProvider {
	return e.metrics
}

// GetTracerProvider returns the tracer provider
func (e *Emitter) GetTracerProvider() *TracerProvider {
	return e.tracer
}

