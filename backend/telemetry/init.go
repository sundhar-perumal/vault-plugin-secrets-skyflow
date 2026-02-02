package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// ============================================================================
// Initialization
// ============================================================================

// Providers holds the initialized telemetry providers
type Providers struct {
	tracerProvider  *sdktrace.TracerProvider
	metricsProvider *sdkmetric.MeterProvider
	traces          *TracesProvider
	metrics         *MetricsProvider
	config          *ResolvedConfig
}

// Init initializes telemetry with traces and metrics using BuildConfigInput.
// Returns providers, shutdown function, and any error.
// If telemetry is disabled or UseNoOp, returns nil providers - OTEL uses built-in noop.
func Init(ctx context.Context, input BuildConfigInput) (*Providers, func(context.Context) error, error) {
	cfg, err := BuildConfig(input)
	if err != nil {
		return nil, nil, err
	}

	return InitWithConfig(ctx, cfg)
}

// InitWithConfig initializes telemetry with a pre-built ResolvedConfig.
func InitWithConfig(ctx context.Context, cfg *ResolvedConfig) (*Providers, func(context.Context) error, error) {
	// Log telemetry status on startup
	logTelemetryStatus(cfg)

	// If UseNoOp (RUNTIME_LOCAL=true in dev/uat), skip provider setup
	// OTEL will use built-in noop tracer automatically
	if cfg.UseNoOp {
		return nil, func(ctx context.Context) error { return nil }, nil
	}

	// If not enabled, return nil providers (noop)
	if !cfg.Enabled {
		return nil, func(ctx context.Context) error { return nil }, nil
	}

	providers := &Providers{config: cfg}

	// Initialize TracerProvider
	if cfg.IsTracesEnabled() {
		tp, err := setupTracerProvider(ctx, cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to setup tracer provider: %w", err)
		}
		providers.tracerProvider = tp
		providers.traces = newTracesProvider(true)
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.TraceContext{})
	}

	// Initialize MetricsProvider
	if cfg.IsMetricsEnabled() {
		mp, metrics, err := setupMetricsProvider(ctx, cfg)
		if err != nil {
			// Cleanup tracer if metrics fail
			if providers.tracerProvider != nil {
				_ = providers.tracerProvider.Shutdown(ctx)
			}
			return nil, nil, fmt.Errorf("failed to setup metrics provider: %w", err)
		}
		providers.metricsProvider = mp
		providers.metrics = metrics
		otel.SetMeterProvider(mp)
	}

	// Return shutdown function
	shutdown := func(ctx context.Context) error {
		var errs []error
		if providers.tracerProvider != nil {
			if err := providers.tracerProvider.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
			}
		}
		if providers.metricsProvider != nil {
			if err := providers.metricsProvider.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("metrics shutdown: %w", err))
			}
		}
		return errors.Join(errs...)
	}

	return providers, shutdown, nil
}

// Metrics returns the MetricsProvider for recording metrics
func (p *Providers) Metrics() *MetricsProvider {
	if p == nil {
		return nil
	}
	return p.metrics
}

// Traces returns the TracesProvider for recording traces
func (p *Providers) Traces() *TracesProvider {
	if p == nil {
		return nil
	}
	return p.traces
}

// IsEnabled returns whether telemetry is enabled
func (p *Providers) IsEnabled() bool {
	return p != nil && (p.tracerProvider != nil || p.metricsProvider != nil)
}

// ============================================================================
// Provider Setup
// ============================================================================

func setupTracerProvider(ctx context.Context, cfg *ResolvedConfig) (*sdktrace.TracerProvider, error) {
	opts := buildTracerExporterOptions(cfg)

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP traces exporter: %w", err)
	}

	res := buildResource(cfg)
	sampler := buildSampler(cfg.SampleRate)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sampler),
	)

	return tp, nil
}

func setupMetricsProvider(ctx context.Context, cfg *ResolvedConfig) (*sdkmetric.MeterProvider, *MetricsProvider, error) {
	opts := buildMetricsExporterOptions(cfg)

	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTLP metrics exporter: %w", err)
	}

	res := buildResource(cfg)

	exportInterval := cfg.MetricsExportInterval
	if exportInterval == 0 {
		exportInterval = 60 * time.Second
	}

	reader := sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithInterval(exportInterval),
	)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)

	// Create MetricsProvider wrapper for recording
	metrics, err := newMetricsProviderFromResolved(mp, cfg)
	if err != nil {
		return nil, nil, err
	}

	return mp, metrics, nil
}

func buildTracerExporterOptions(cfg *ResolvedConfig) []otlptracehttp.Option {
	var opts []otlptracehttp.Option

	if cfg.TracesEndpoint != "" {
		endpoint, urlPath, useInsecure := parseEndpointURL(cfg.TracesEndpoint)
		opts = append(opts, otlptracehttp.WithEndpoint(endpoint))

		if urlPath != "" && urlPath != "/v1/traces" {
			opts = append(opts, otlptracehttp.WithURLPath(urlPath))
		}

		if useInsecure || cfg.TracesInsecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
	}

	if len(cfg.TracesHeaders) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.TracesHeaders))
	}

	return opts
}

func buildMetricsExporterOptions(cfg *ResolvedConfig) []otlpmetrichttp.Option {
	var opts []otlpmetrichttp.Option

	if cfg.MetricsEndpoint != "" {
		endpoint, urlPath, useInsecure := parseEndpointURL(cfg.MetricsEndpoint)
		opts = append(opts, otlpmetrichttp.WithEndpoint(endpoint))

		if urlPath != "" && urlPath != "/v1/metrics" {
			opts = append(opts, otlpmetrichttp.WithURLPath(urlPath))
		}

		if useInsecure || cfg.MetricsInsecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
	}

	if len(cfg.MetricsHeaders) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.MetricsHeaders))
	}

	return opts
}

func buildResource(cfg *ResolvedConfig) *resource.Resource {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		attribute.String("environment", cfg.Environment),
	}

	if cfg.ServiceNamespace != "" {
		attrs = append(attrs, semconv.ServiceNamespace(cfg.ServiceNamespace))
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...)
}

func buildSampler(sampleRate float64) sdktrace.Sampler {
	if sampleRate >= 1.0 {
		return sdktrace.AlwaysSample()
	}
	if sampleRate <= 0.0 {
		return sdktrace.NeverSample()
	}
	return sdktrace.TraceIDRatioBased(sampleRate)
}

func parseEndpointURL(rawURL string) (endpoint string, urlPath string, useInsecure bool) {
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

// ============================================================================
// Startup Logging
// ============================================================================

// logTelemetryStatus logs the resolved telemetry configuration on startup
func logTelemetryStatus(cfg *ResolvedConfig) {
	if cfg.UseNoOp {
		logInfof("disabled (RUNTIME_LOCAL=true, env=%s)", cfg.Environment)
		return
	}

	if !cfg.Enabled {
		logInfof("disabled (TELEMETRY_ENABLED=false)")
		return
	}

	tracesStatus := "off"
	if cfg.IsTracesEnabled() {
		tracesStatus = cfg.TracesEndpoint
	}

	metricsStatus := "off"
	if cfg.IsMetricsEnabled() {
		metricsStatus = cfg.MetricsEndpoint
	}

	logInfof("enabled (env=%s, traces=%s, metrics=%s, sample_rate=%.2f)",
		cfg.Environment,
		tracesStatus,
		metricsStatus,
		cfg.SampleRate,
	)
}

// ============================================================================
// Trace Context Extraction (W3C traceparent)
// ============================================================================

// ExtractTraceContext extracts W3C trace context from HTTP headers (traceparent header).
// Returns a context with the extracted trace context, or the original context if no valid trace found.
func ExtractTraceContext(ctx context.Context, headers http.Header) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(headers))
}

// InjectTraceContext injects W3C trace context into HTTP headers (traceparent header).
func InjectTraceContext(ctx context.Context, headers http.Header) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(headers))
}