package telemetry

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// Environment-specific OTEL Endpoints (Code-based mapping)
// ============================================================================

// otelEndpoints maps environment to OTEL collector endpoints
// These are default endpoints; can be overridden by ENV vars
var otelEndpoints = map[string]struct {
	Traces  string
	Metrics string
}{
	"dev": {
		Traces:  "https://otel-dev.example.com/otlp/v1/traces",
		Metrics: "https://otel-dev.example.com/otlp/v1/metrics",
	},
	"uat": {
		Traces:  "https://otel-uat.example.com/otlp/v1/traces",
		Metrics: "https://otel-uat.example.com/otlp/v1/metrics",
	},
	"cug": {
		Traces:  "https://otel-cug.example.com/otlp/v1/traces",
		Metrics: "https://otel-cug.example.com/otlp/v1/metrics",
	},
	"prod": {
		Traces:  "https://otel.example.com/otlp/v1/traces",
		Metrics: "https://otel.example.com/otlp/v1/metrics",
	},
}

// ============================================================================
// BuildConfig - Unified config builder
// ============================================================================

// BuildConfigInput contains all inputs for building telemetry configuration
type BuildConfigInput struct {
	// Required fields
	ServiceName    string // Service name (default: "skyflow-vault-plugin")
	ServiceVersion string // Service version
	Environment    string // Environment (dev, uat, cug, prod)

	// Optional fields
	ServiceNamespace string  // Team/namespace (default: "go-skyflow-harshicorp-plugin")
	SampleRate       float64 // Trace sample rate 0.0-1.0 (default: 1.0 = 100%)
}

// ResolvedConfig is the final merged configuration used by providers
type ResolvedConfig struct {
	// Master switch (if false, all telemetry off)
	Enabled bool

	// UseNoOp indicates whether to use NoOp emitter (no external calls)
	// This is determined by RUNTIME_LOCAL env var and Environment
	UseNoOp bool

	// Service identity
	ServiceName      string
	ServiceNamespace string
	ServiceVersion   string
	Environment      string

	// Traces configuration
	TracesEndpoint string
	TracesHeaders  map[string]string
	TracesInsecure bool
	TracesTimeout  time.Duration

	// Metrics configuration
	MetricsEndpoint       string
	MetricsHeaders        map[string]string
	MetricsInsecure       bool
	MetricsExportInterval time.Duration

	// Sample rate for traces (0.0 to 1.0)
	SampleRate float64
}

// IsTracesEnabled returns true if tracing should be active
// Requires: master enabled + not NoOp + traces endpoint available
func (c *ResolvedConfig) IsTracesEnabled() bool {
	return c.Enabled && !c.UseNoOp && c.TracesEndpoint != ""
}

// IsMetricsEnabled returns true if metrics should be active
// Requires: master enabled + not NoOp + metrics endpoint available
func (c *ResolvedConfig) IsMetricsEnabled() bool {
	return c.Enabled && !c.UseNoOp && c.MetricsEndpoint != ""
}

// BuildConfig builds ResolvedConfig with the following priority (highest to lowest):
//  1. Environment variables (OTEL_*, TELEMETRY_*) - HIGHEST
//  2. Code-based defaults (otelEndpoints map)
//  3. Default values
//
// RUNTIME_LOCAL behavior:
// - dev/uat: RUNTIME_LOCAL=true -> NoOp emitter (fast, no external calls)
// - cug/prod: RUNTIME_LOCAL is ignored for safety (always real OTEL)
func BuildConfig(input BuildConfigInput) (*ResolvedConfig, error) {
	config := &ResolvedConfig{
		Enabled:    true,
		SampleRate: 1.0,
	}

	// === ENVIRONMENT (Required) ===
	config.Environment = input.Environment
	if config.Environment == "" {
		config.Environment = os.Getenv("ENV")
	}
	if config.Environment == "" {
		config.Environment = "unknown"
	}

	// === RUNTIME_LOCAL (NoOp emitter selection) ===
	// RUNTIME_LOCAL=true allows developers to run with NoOp telemetry (no external calls)
	// Safety: Only honored in dev/uat environments; cug/prod always use real OTEL
	runtimeLocal := strings.ToLower(os.Getenv("RUNTIME_LOCAL")) == "true"

	if config.Environment == "cug" || config.Environment == "prod" {
		// cug/prod: RUNTIME_LOCAL is ignored for safety - always use real OTEL
		if runtimeLocal {
			logWarnf("RUNTIME_LOCAL=true ignored in %s environment - telemetry remains enabled for safety", config.Environment)
		}
		config.UseNoOp = false
	} else {
		// dev/uat: Honor RUNTIME_LOCAL
		if runtimeLocal {
			logWarnf("telemetry disabled via RUNTIME_LOCAL=true in %s environment - using NoOp emitter", config.Environment)
			config.UseNoOp = true
		}
	}

	// === MASTER SWITCH ===
	// Check TELEMETRY_ENABLED env var
	if val := os.Getenv("TELEMETRY_ENABLED"); val != "" {
		config.Enabled = strings.ToLower(val) == "true" || val == "1"
	}

	// === SERVICE NAME ===
	config.ServiceName = resolveStringValue(
		input.ServiceName,
		"OTEL_SERVICE_NAME",
		"skyflow-vault-plugin",
	)

	// === SERVICE NAMESPACE ===
	config.ServiceNamespace = resolveStringValue(
		input.ServiceNamespace,
		"SERVICE_NAMESPACE",
		"go-skyflow-harshicorp-plugin",
	)

	// === SERVICE VERSION ===
	config.ServiceVersion = input.ServiceVersion
	if config.ServiceVersion == "" {
		config.ServiceVersion = "unknown"
	}

	// === TRACES ENDPOINT ===
	// Priority: ENV > code-based mapping > empty (disabled)
	config.TracesEndpoint = resolveStringValue(
		"",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		getDefaultTracesEndpoint(config.Environment),
	)

	// Check TELEMETRY_TRACES_ENABLED to override
	if val := os.Getenv("TELEMETRY_TRACES_ENABLED"); val != "" {
		if strings.ToLower(val) == "false" || val == "0" {
			config.TracesEndpoint = "" // Disable traces
		}
	}

	// Traces insecure (auto-detect from URL or ENV)
	config.TracesInsecure = resolveBoolFlag(
		nil,
		"OTEL_EXPORTER_OTLP_INSECURE",
		strings.HasPrefix(config.TracesEndpoint, "http://"),
	)

	// Traces headers
	config.TracesHeaders = resolveHeaders("OTEL_EXPORTER_OTLP_HEADERS")

	// Traces timeout
	config.TracesTimeout = resolveDuration(
		os.Getenv("OTEL_EXPORTER_OTLP_TIMEOUT"),
		30*time.Second,
	)

	// === METRICS ENDPOINT ===
	// Priority: ENV > code-based mapping > empty (disabled)
	config.MetricsEndpoint = resolveStringValue(
		"",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		getDefaultMetricsEndpoint(config.Environment),
	)

	// Check TELEMETRY_METRICS_ENABLED to override
	if val := os.Getenv("TELEMETRY_METRICS_ENABLED"); val != "" {
		if strings.ToLower(val) == "false" || val == "0" {
			config.MetricsEndpoint = "" // Disable metrics
		}
	}

	// Metrics insecure
	config.MetricsInsecure = resolveBoolFlag(
		nil,
		"OTEL_EXPORTER_OTLP_METRICS_INSECURE",
		strings.HasPrefix(config.MetricsEndpoint, "http://"),
	)

	// Metrics headers
	config.MetricsHeaders = resolveHeaders("OTEL_EXPORTER_OTLP_METRICS_HEADERS")

	// Metrics export interval
	config.MetricsExportInterval = resolveDuration(
		os.Getenv("TELEMETRY_METRICS_EXPORT_INTERVAL"),
		60*time.Second,
	)

	// === SAMPLE RATE ===
	// Priority: input > ENV > default (1.0)
	config.SampleRate = resolveSampleRate(input.SampleRate, "TELEMETRY_SAMPLE_RATE", 1.0)

	return config, nil
}

// ============================================================================
// Helper Functions - Default Endpoints
// ============================================================================

// getDefaultTracesEndpoint returns the default traces endpoint for environment
func getDefaultTracesEndpoint(env string) string {
	if endpoints, ok := otelEndpoints[env]; ok {
		return endpoints.Traces
	}
	return ""
}

// getDefaultMetricsEndpoint returns the default metrics endpoint for environment
func getDefaultMetricsEndpoint(env string) string {
	if endpoints, ok := otelEndpoints[env]; ok {
		return endpoints.Metrics
	}
	return ""
}

// ============================================================================
// Helper Functions - Priority Resolution
// ============================================================================

// resolveBoolFlag resolves a boolean with priority: clientValue > ENV > default
func resolveBoolFlag(clientValue *bool, envVar string, defaultValue bool) bool {
	// 1. client value (highest priority)
	if clientValue != nil {
		return *clientValue
	}

	// 2. Environment variable
	if envVar != "" {
		if envVal := os.Getenv(envVar); envVal != "" {
			return strings.ToLower(envVal) == "true" || envVal == "1"
		}
	}

	// 3. Default
	return defaultValue
}

// resolveStringValue resolves a string with priority: clientValue > ENV > default
func resolveStringValue(clientValue, envVar, defaultValue string) string {
	// 1. client value (highest priority)
	if clientValue != "" {
		return clientValue
	}

	// 2. Environment variable
	if envVar != "" {
		if envVal := os.Getenv(envVar); envVal != "" {
			return envVal
		}
	}

	// 3. Default
	return defaultValue
}

// resolveHeaders resolves headers from ENV
func resolveHeaders(envVar string) map[string]string {
	if envVar == "" {
		return nil
	}

	raw := os.Getenv(envVar)
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

// resolveDuration parses a duration string with fallback
func resolveDuration(value string, defaultValue time.Duration) time.Duration {
	if value == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return d
}

// resolveSampleRate resolves sample rate with priority: clientValue > ENV > default
// Valid range is 0.0 to 1.0; values outside this range are clamped
func resolveSampleRate(clientValue float64, envVar string, defaultValue float64) float64 {
	// 1. Check client value (if non-zero, use it)
	if clientValue > 0 {
		return clampSampleRate(clientValue)
	}

	// 2. Check environment variable
	if envVar != "" {
		if envVal := os.Getenv(envVar); envVal != "" {
			if f, err := strconv.ParseFloat(envVal, 64); err == nil {
				return clampSampleRate(f)
			}
		}
	}

	// 3. Default
	return defaultValue
}

// clampSampleRate ensures sample rate is between 0.0 and 1.0
func clampSampleRate(rate float64) float64 {
	if rate < 0.0 {
		return 0.0
	}
	if rate > 1.0 {
		return 1.0
	}
	return rate
}

// ============================================================================
// Helper Functions - Logging
// ============================================================================

// logWarnf outputs a warning message to stderr
// Used for RUNTIME_LOCAL warnings that should be visible for monitoring
func logWarnf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[WARN] skyflow-vault-plugin/telemetry: "+format+"\n", args...)
}

// logInfof outputs an info message to stderr
func logInfof(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[INFO] skyflow-vault-plugin/telemetry: "+format+"\n", args...)
}
