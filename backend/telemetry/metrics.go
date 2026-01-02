package telemetry

import (
	"fmt"
	"context"
	"net/url"
	"strings"
	"sync"
	"time"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// MetricsProvider wraps the OpenTelemetry SDK metrics components
type MetricsProvider struct {
	config        *Config
	meterProvider *sdkmetric.MeterProvider
	meter         metric.Meter
	enabled       bool

	// Counters
	tokenGeneratesTotal metric.Int64Counter
	tokenErrorsTotal    metric.Int64Counter
	configWritesTotal   metric.Int64Counter
	roleWritesTotal     metric.Int64Counter

	// Histograms
	tokenGenerateDuration metric.Float64Histogram

	// Internal state
	mu        sync.RWMutex
	startTime time.Time
}

// NewMetricsProvider creates a new MetricsProvider from config
func NewMetricsProvider(ctx context.Context, config *Config) (*MetricsProvider, error) {
	if !config.Enabled || !config.MetricsEnabled {
		return &MetricsProvider{config: config, enabled: false}, nil
	}

	// Build OTLP HTTP exporter options
	opts := []otlpmetrichttp.Option{}

	if config.MetricsEndpoint != "" {
		endpoint, urlPath, useInsecure := parseMetricsEndpointURL(config.MetricsEndpoint)

		opts = append(opts, otlpmetrichttp.WithEndpoint(endpoint))

		if urlPath != "" && urlPath != "/v1/metrics" {
			opts = append(opts, otlpmetrichttp.WithURLPath(urlPath))
		}

		if useInsecure || config.MetricsInsecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
	}

	if len(config.MetricsHeaders) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(config.MetricsHeaders))
	}

	// Create OTLP HTTP exporter
	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metrics exporter: %w", err)
	}

	// Build resource with service information
	attrs := []attribute.KeyValue{
		semconv.ServiceName(config.ServiceName),
		semconv.ServiceVersion("1.0.0"),
		attribute.String("environment", config.Environment),
	}

	if config.ServiceNamespace != "" {
		attrs = append(attrs, semconv.ServiceNamespace(config.ServiceNamespace))
	}

	res := resource.NewWithAttributes(semconv.SchemaURL, attrs...)

	// Configure periodic reader
	exportInterval := config.MetricsExportInterval
	if exportInterval == 0 {
		exportInterval = 60 * time.Second
	}

	reader := sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithInterval(exportInterval),
	)

	// Create MeterProvider
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)

	// Register as global provider
	otel.SetMeterProvider(mp)

	// Create meter for this library
	meter := mp.Meter(
		TracerName,
		metric.WithInstrumentationVersion("1.0.0"),
	)

	// Create provider instance
	p := &MetricsProvider{
		config:        config,
		meterProvider: mp,
		meter:         meter,
		enabled:       true,
		startTime:     time.Now(),
	}

	// Initialize metrics
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
		"skyflow_token_generates_total",
		metric.WithDescription("Total number of token generations"),
		metric.WithUnit("{generation}"),
	)
	if err != nil {
		return err
	}

	p.tokenErrorsTotal, err = p.meter.Int64Counter(
		"skyflow_token_errors_total",
		metric.WithDescription("Total number of token generation errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return err
	}

	p.configWritesTotal, err = p.meter.Int64Counter(
		"skyflow_config_writes_total",
		metric.WithDescription("Total number of configuration writes"),
		metric.WithUnit("{write}"),
	)
	if err != nil {
		return err
	}

	p.roleWritesTotal, err = p.meter.Int64Counter(
		"skyflow_role_writes_total",
		metric.WithDescription("Total number of role writes"),
		metric.WithUnit("{write}"),
	)
	if err != nil {
		return err
	}

	// === HISTOGRAMS ===

	p.tokenGenerateDuration, err = p.meter.Float64Histogram(
		"skyflow_token_generate_duration_ms",
		metric.WithDescription("Token generation latency in milliseconds"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000),
	)
	if err != nil {
		return err
	}

	return nil
}

// parseMetricsEndpointURL parses a full URL for metrics endpoint
func parseMetricsEndpointURL(rawURL string) (endpoint string, urlPath string, useInsecure bool) {
	if !strings.Contains(rawURL, "://") {
		return rawURL, "", false
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, "", false
	}

	endpoint = parsed.Host
	urlPath = parsed.Path
	if urlPath == "/" {
		urlPath = ""
	}
	useInsecure = parsed.Scheme == "http"

	return endpoint, urlPath, useInsecure
}

// IsEnabled returns whether metrics are active
func (p *MetricsProvider) IsEnabled() bool {
	return p.enabled && p.meterProvider != nil
}

// Shutdown gracefully shuts down the provider
func (p *MetricsProvider) Shutdown(ctx context.Context) error {
	if p.meterProvider == nil {
		return nil
	}
	return p.meterProvider.Shutdown(ctx)
}

// ForceFlush forces an immediate export of all pending metrics
func (p *MetricsProvider) ForceFlush(ctx context.Context) error {
	if p.meterProvider == nil {
		return nil
	}
	return p.meterProvider.ForceFlush(ctx)
}

// ============================================================================
// Metric Recording Methods
// ============================================================================

// RecordTokenGenerate records a token generation
func (p *MetricsProvider) RecordTokenGenerate(ctx context.Context, role string, durationMs float64, success bool) {
	if !p.IsEnabled() {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("role", role),
		attribute.Bool("success", success),
	)

	p.tokenGeneratesTotal.Add(ctx, 1, attrs)
	p.tokenGenerateDuration.Record(ctx, durationMs, attrs)
}

// RecordTokenError records a token generation error
func (p *MetricsProvider) RecordTokenError(ctx context.Context, role, errorType string) {
	if !p.IsEnabled() {
		return
	}

	p.tokenErrorsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("role", role),
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

