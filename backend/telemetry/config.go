package telemetry

import (
	"os"
	"strings"
	"time"
)

// Config represents the telemetry configuration
type Config struct {
	// Master switch
	Enabled bool

	// Service identity
	ServiceName      string
	ServiceNamespace string
	Environment      string

	// Traces configuration
	TracesEnabled  bool
	TracesEndpoint string
	TracesHeaders  map[string]string
	TracesInsecure bool

	// Metrics configuration
	MetricsEnabled        bool
	MetricsEndpoint       string
	MetricsHeaders        map[string]string
	MetricsInsecure       bool
	MetricsExportInterval time.Duration

	// Sampling
	SampleRate float64
}

// DefaultConfig returns default telemetry configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:               true,
		ServiceName:           "skyflow-vault-plugin",
		ServiceNamespace:      "skyflow",
		Environment:           "unknown",
		TracesEnabled:         true,
		MetricsEnabled:        true,
		MetricsExportInterval: 60 * time.Second,
		SampleRate:            1.0,
	}
}

// ConfigFromEnv builds configuration from environment variables
func ConfigFromEnv() *Config {
	config := DefaultConfig()

	// Master switch
	if val := os.Getenv("TELEMETRY_ENABLED"); val != "" {
		config.Enabled = strings.ToLower(val) == "true" || val == "1"
	}

	// Service identity
	if val := os.Getenv("OTEL_SERVICE_NAME"); val != "" {
		config.ServiceName = val
	}
	if val := os.Getenv("SERVICE_NAMESPACE"); val != "" {
		config.ServiceNamespace = val
	}
	if val := os.Getenv("ENV"); val != "" {
		config.Environment = val
	}

	// Traces
	if val := os.Getenv("TELEMETRY_TRACES_ENABLED"); val != "" {
		config.TracesEnabled = strings.ToLower(val) == "true" || val == "1"
	}
	if val := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"); val != "" {
		config.TracesEndpoint = val
	}
	if val := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"); val != "" {
		config.TracesInsecure = strings.ToLower(val) == "true" || val == "1"
	}
	config.TracesHeaders = parseHeaders(os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"))

	// Metrics
	if val := os.Getenv("TELEMETRY_METRICS_ENABLED"); val != "" {
		config.MetricsEnabled = strings.ToLower(val) == "true" || val == "1"
	}
	if val := os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"); val != "" {
		config.MetricsEndpoint = val
	}
	if val := os.Getenv("OTEL_EXPORTER_OTLP_METRICS_INSECURE"); val != "" {
		config.MetricsInsecure = strings.ToLower(val) == "true" || val == "1"
	}
	config.MetricsHeaders = parseHeaders(os.Getenv("OTEL_EXPORTER_OTLP_METRICS_HEADERS"))

	return config
}

// parseHeaders parses comma-separated key=value headers
func parseHeaders(raw string) map[string]string {
	if raw == "" {
		return nil
	}

	headers := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return headers
}

// IsTracesEnabled returns whether traces are enabled (considering master switch)
func (c *Config) IsTracesEnabled() bool {
	return c.Enabled && c.TracesEnabled && c.TracesEndpoint != ""
}

// IsMetricsEnabled returns whether metrics are enabled (considering master switch)
func (c *Config) IsMetricsEnabled() bool {
	return c.Enabled && c.MetricsEnabled && c.MetricsEndpoint != ""
}

