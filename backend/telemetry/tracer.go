package telemetry

import (
	"fmt"
	"context"
	"net/url"
	"strings"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider wraps the OpenTelemetry SDK tracing components
type TracerProvider struct {
	config         *Config
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	enabled        bool
}

// NewTracerProvider creates a new TracerProvider from config
func NewTracerProvider(ctx context.Context, config *Config) (*TracerProvider, error) {
	if !config.Enabled || !config.TracesEnabled {
		return &TracerProvider{config: config, enabled: false}, nil
	}

	// Build OTLP HTTP exporter options
	opts := []otlptracehttp.Option{}

	if config.TracesEndpoint != "" {
		endpoint, urlPath, useInsecure := parseEndpointURL(config.TracesEndpoint)

		opts = append(opts, otlptracehttp.WithEndpoint(endpoint))

		if urlPath != "" && urlPath != "/v1/traces" {
			opts = append(opts, otlptracehttp.WithURLPath(urlPath))
		}

		if useInsecure || config.TracesInsecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
	}

	if len(config.TracesHeaders) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(config.TracesHeaders))
	}

	// Create OTLP HTTP exporter
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP traces exporter: %w", err)
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

	// Configure sampler
	var sampler sdktrace.Sampler
	if config.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SampleRate <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SampleRate)
	}

	// Create TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sampler),
	)

	// Register as global provider
	otel.SetTracerProvider(tp)

	// Create tracer for this library
	tracer := tp.Tracer(
		TracerName,
		trace.WithInstrumentationVersion("1.0.0"),
	)

	return &TracerProvider{
		config:         config,
		tracerProvider: tp,
		tracer:         tracer,
		enabled:        true,
	}, nil
}

// parseEndpointURL parses a full URL and extracts components for OTLP exporter
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

// GetTracer returns the OTEL tracer for creating spans
func (p *TracerProvider) GetTracer() trace.Tracer {
	if p.tracer == nil {
		return otel.Tracer(TracerName)
	}
	return p.tracer
}

// IsEnabled returns whether tracing is active
func (p *TracerProvider) IsEnabled() bool {
	return p.enabled && p.tracerProvider != nil
}

// Shutdown gracefully shuts down the provider
func (p *TracerProvider) Shutdown(ctx context.Context) error {
	if p.tracerProvider == nil {
		return nil
	}
	return p.tracerProvider.Shutdown(ctx)
}

// ForceFlush forces an immediate export of all pending spans
func (p *TracerProvider) ForceFlush(ctx context.Context) error {
	if p.tracerProvider == nil {
		return nil
	}
	return p.tracerProvider.ForceFlush(ctx)
}

// ============================================================================
// Span Creation Helpers
// ============================================================================

// StartSpan starts a new span with the given name and attributes
func (p *TracerProvider) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if !p.IsEnabled() {
		return ctx, trace.SpanFromContext(ctx)
	}
	return p.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// StartSpanWithKind starts a new span with specific kind
func (p *TracerProvider) StartSpanWithKind(ctx context.Context, name string, kind trace.SpanKind, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if !p.IsEnabled() {
		return ctx, trace.SpanFromContext(ctx)
	}
	return p.tracer.Start(ctx, name,
		trace.WithSpanKind(kind),
		trace.WithAttributes(attrs...),
	)
}

// ============================================================================
// Span Event Helpers
// ============================================================================

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetSpanError marks the span as having an error
func SetSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanOK marks the span as successful
func SetSpanOK(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetStatus(codes.Ok, "")
	}
}

// EndSpan ends the span from context
func EndSpan(span trace.Span) {
	if span != nil {
		span.End()
	}
}

