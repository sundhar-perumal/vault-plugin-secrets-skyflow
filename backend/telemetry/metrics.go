package telemetry

import (
	"fmt"
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// MetricsProvider wraps metric instruments for recording
type MetricsProvider struct {
	meter   metric.Meter
	enabled bool

	// Counters
	tokenGeneratesTotal metric.Int64Counter
	tokenErrorsTotal    metric.Int64Counter
	configWritesTotal   metric.Int64Counter
	roleWritesTotal     metric.Int64Counter
	configErrorsTotal   metric.Int64Counter
	roleErrorsTotal     metric.Int64Counter
	configReadsTotal    metric.Int64Counter
	roleReadsTotal      metric.Int64Counter
	healthChecksTotal   metric.Int64Counter
	sdkCallTotal        metric.Int64Counter
	sdkCallErrors       metric.Int64Counter

	// Histograms
	tokenGenerateDuration metric.Float64Histogram
	sdkCallDuration       metric.Float64Histogram

	// Internal state
	mu        sync.RWMutex
	startTime time.Time
}

// newMetricsProviderFromResolved creates a MetricsProvider from an existing MeterProvider using ResolvedConfig
func newMetricsProviderFromResolved(mp *sdkmetric.MeterProvider, cfg *ResolvedConfig) (*MetricsProvider, error) {
	meter := mp.Meter(
		TracerName,
		metric.WithInstrumentationVersion("1.0.0"),
	)

	p := &MetricsProvider{
		meter:     meter,
		enabled:   true,
		startTime: time.Now(),
	}

	if err := p.initMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics instruments: %w", err)
	}

	return p, nil
}

// initMetrics initializes all metric instruments
func (p *MetricsProvider) initMetrics() error {
	var err error

	// === COUNTERS ===

	p.tokenGeneratesTotal, err = p.meter.Int64Counter(
		"skyflow_total_tokens_generated",
		metric.WithDescription("Total number of tokens generated"),
		metric.WithUnit("{token}"),
	)
	if err != nil {
		return err
	}

	p.tokenErrorsTotal, err = p.meter.Int64Counter(
		"skyflow_total_tokens_failed",
		metric.WithDescription("Total number of token generation failures"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return err
	}

	p.configWritesTotal, err = p.meter.Int64Counter(
		"skyflow_total_config_created",
		metric.WithDescription("Total number of config operations"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		return err
	}

	p.roleWritesTotal, err = p.meter.Int64Counter(
		"skyflow_total_roles_created",
		metric.WithDescription("Total number of role operations"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		return err
	}

	p.configErrorsTotal, err = p.meter.Int64Counter(
		"skyflow_config_errors_total",
		metric.WithDescription("Total number of config errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return err
	}

	p.roleErrorsTotal, err = p.meter.Int64Counter(
		"skyflow_role_errors_total",
		metric.WithDescription("Total number of role errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return err
	}

	p.configReadsTotal, err = p.meter.Int64Counter(
		"skyflow_config_reads_total",
		metric.WithDescription("Total number of config reads"),
		metric.WithUnit("{read}"),
	)
	if err != nil {
		return err
	}

	p.roleReadsTotal, err = p.meter.Int64Counter(
		"skyflow_role_reads_total",
		metric.WithDescription("Total number of role reads"),
		metric.WithUnit("{read}"),
	)
	if err != nil {
		return err
	}

	p.healthChecksTotal, err = p.meter.Int64Counter(
		"skyflow_health_checks_total",
		metric.WithDescription("Total number of health checks"),
		metric.WithUnit("{check}"),
	)
	if err != nil {
		return err
	}

	p.sdkCallTotal, err = p.meter.Int64Counter(
		"skyflow_sdk_call_total",
		metric.WithDescription("Total number of Skyflow SDK calls"),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return err
	}

	p.sdkCallErrors, err = p.meter.Int64Counter(
		"skyflow_sdk_call_errors_total",
		metric.WithDescription("Total number of Skyflow SDK call errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return err
	}

	// === HISTOGRAMS ===

	p.tokenGenerateDuration, err = p.meter.Float64Histogram(
		"skyflow_token_generated_duration_ms",
		metric.WithDescription("Token generation latency in milliseconds"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000),
	)
	if err != nil {
		return err
	}

	p.sdkCallDuration, err = p.meter.Float64Histogram(
		"skyflow_sdk_call_duration_ms",
		metric.WithDescription("Skyflow SDK call latency in milliseconds"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000),
	)
	if err != nil {
		return err
	}

	return nil
}

// IsEnabled returns whether metrics are active
func (p *MetricsProvider) IsEnabled() bool {
	return p != nil && p.enabled
}

// ============================================================================
// Metric Recording Methods
// ============================================================================

// RecordTokenGenerate records a token generation
func (p *MetricsProvider) RecordTokenGenerate(ctx context.Context, role, vaultServiceName, skyflowVaultName string, durationMs float64, success bool) {
	if !p.IsEnabled() {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("role", role),
		attribute.String("vault_service_name", vaultServiceName),
		attribute.String("skyflow_vault_name", skyflowVaultName),
		attribute.Bool("success", success),
	)

	p.tokenGeneratesTotal.Add(ctx, 1, attrs)
	p.tokenGenerateDuration.Record(ctx, durationMs, attrs)
}

// RecordTokenError records a token generation error
func (p *MetricsProvider) RecordTokenError(ctx context.Context, role, vaultServiceName, skyflowVaultName, errorType string) {
	if !p.IsEnabled() {
		return
	}

	p.tokenErrorsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("role", role),
			attribute.String("vault_service_name", vaultServiceName),
			attribute.String("skyflow_vault_name", skyflowVaultName),
			attribute.String("error_type", errorType),
		),
	)
}

// RecordConfigWrite records a config write operation
func (p *MetricsProvider) RecordConfigWrite(ctx context.Context, operation string) {
	if !p.IsEnabled() {
		return
	}

	p.configWritesTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
		),
	)
}

// RecordRoleWrite records a role write operation
func (p *MetricsProvider) RecordRoleWrite(ctx context.Context, role, operation string) {
	if !p.IsEnabled() {
		return
	}

	p.roleWritesTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("role", role),
			attribute.String("operation", operation),
		),
	)
}

// RecordConfigError records a config error
func (p *MetricsProvider) RecordConfigError(ctx context.Context, operation, errorType string) {
	if !p.IsEnabled() {
		return
	}

	p.configErrorsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("error_type", errorType),
		),
	)
}

// RecordRoleError records a role error
func (p *MetricsProvider) RecordRoleError(ctx context.Context, role, operation, errorType string) {
	if !p.IsEnabled() {
		return
	}

	p.roleErrorsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("role", role),
			attribute.String("operation", operation),
			attribute.String("error_type", errorType),
		),
	)
}

// RecordConfigRead records a config read operation
func (p *MetricsProvider) RecordConfigRead(ctx context.Context, operation string) {
	if !p.IsEnabled() {
		return
	}

	p.configReadsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
		),
	)
}

// RecordRoleRead records a role read operation
func (p *MetricsProvider) RecordRoleRead(ctx context.Context, role, operation string) {
	if !p.IsEnabled() {
		return
	}

	p.roleReadsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("role", role),
			attribute.String("operation", operation),
		),
	)
}

// RecordHealthCheck records a health check operation
func (p *MetricsProvider) RecordHealthCheck(ctx context.Context, status string) {
	if !p.IsEnabled() {
		return
	}

	p.healthChecksTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("status", status),
		),
	)
}

// RecordSkyflowSDKCall records a Skyflow SDK call with duration
func (p *MetricsProvider) RecordSkyflowSDKCall(ctx context.Context, roleName, status string, durationMs float64) {
	if !p.IsEnabled() {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("role", roleName),
		attribute.String("status", status),
	)

	p.sdkCallTotal.Add(ctx, 1, attrs)
	p.sdkCallDuration.Record(ctx, durationMs, attrs)
}

// RecordSkyflowSDKCallError records a Skyflow SDK call error
func (p *MetricsProvider) RecordSkyflowSDKCallError(ctx context.Context, roleName, errorType string) {
	if !p.IsEnabled() {
		return
	}

	p.sdkCallErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("role", roleName),
			attribute.String("error_type", errorType),
		),
	)
}