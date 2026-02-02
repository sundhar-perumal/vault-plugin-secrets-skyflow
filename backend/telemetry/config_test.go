package telemetry

import (
	"os"
	"testing"
	"time"
)

func TestBuildConfig_DefaultValues(t *testing.T) {
	// Clear environment
	clearTelemetryEnv(t)

	cfg, err := BuildConfig(BuildConfigInput{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "dev",
	})

	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}

	if cfg.ServiceName != "test-service" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "test-service")
	}

	if cfg.ServiceVersion != "1.0.0" {
		t.Errorf("ServiceVersion = %q, want %q", cfg.ServiceVersion, "1.0.0")
	}

	if cfg.Environment != "dev" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "dev")
	}

	if cfg.SampleRate != 1.0 {
		t.Errorf("SampleRate = %v, want %v", cfg.SampleRate, 1.0)
	}
}

func TestBuildConfig_EnvironmentVariableOverrides(t *testing.T) {
	clearTelemetryEnv(t)

	// Set environment variables
	os.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "https://custom-endpoint/traces")
	os.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "https://custom-endpoint/metrics")
	os.Setenv("OTEL_SERVICE_NAME", "env-service-name")
	defer clearTelemetryEnv(t)

	cfg, err := BuildConfig(BuildConfigInput{
		ServiceName:    "", // Should fall through to ENV
		ServiceVersion: "1.0.0",
		Environment:    "uat",
	})

	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}

	if cfg.TracesEndpoint != "https://custom-endpoint/traces" {
		t.Errorf("TracesEndpoint = %q, want %q", cfg.TracesEndpoint, "https://custom-endpoint/traces")
	}

	if cfg.MetricsEndpoint != "https://custom-endpoint/metrics" {
		t.Errorf("MetricsEndpoint = %q, want %q", cfg.MetricsEndpoint, "https://custom-endpoint/metrics")
	}

	if cfg.ServiceName != "env-service-name" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "env-service-name")
	}
}

func TestBuildConfig_DefaultEndpointsPerEnvironment(t *testing.T) {
	clearTelemetryEnv(t)

	tests := []struct {
		env         string
		wantTraces  string
		wantMetrics string
	}{
		{
			env:         "dev",
			wantTraces:  "https://otel-dev.example.com/otlp/v1/traces",
			wantMetrics: "https://otel-dev.example.com/otlp/v1/metrics",
		},
		{
			env:         "uat",
			wantTraces:  "https://otel-uat.example.com/otlp/v1/traces",
			wantMetrics: "https://otel-uat.example.com/otlp/v1/metrics",
		},
		{
			env:         "cug",
			wantTraces:  "https://otel-cug.example.com/otlp/v1/traces",
			wantMetrics: "https://otel-cug.example.com/otlp/v1/metrics",
		},
		{
			env:         "prod",
			wantTraces:  "https://otel.example.com/otlp/v1/traces",
			wantMetrics: "https://otel.example.com/otlp/v1/metrics",
		},
		{
			env:         "unknown",
			wantTraces:  "",
			wantMetrics: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			clearTelemetryEnv(t)

			cfg, err := BuildConfig(BuildConfigInput{
				ServiceName:    "test",
				ServiceVersion: "1.0.0",
				Environment:    tt.env,
			})

			if err != nil {
				t.Fatalf("BuildConfig() error = %v", err)
			}

			if cfg.TracesEndpoint != tt.wantTraces {
				t.Errorf("TracesEndpoint = %q, want %q", cfg.TracesEndpoint, tt.wantTraces)
			}

			if cfg.MetricsEndpoint != tt.wantMetrics {
				t.Errorf("MetricsEndpoint = %q, want %q", cfg.MetricsEndpoint, tt.wantMetrics)
			}
		})
	}
}

func TestBuildConfig_RuntimeLocal_DevUat(t *testing.T) {
	clearTelemetryEnv(t)

	os.Setenv("RUNTIME_LOCAL", "true")
	defer clearTelemetryEnv(t)

	// dev environment should honor RUNTIME_LOCAL
	cfg, err := BuildConfig(BuildConfigInput{
		Environment: "dev",
	})

	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}

	if !cfg.UseNoOp {
		t.Error("UseNoOp = false, want true for dev with RUNTIME_LOCAL=true")
	}

	// uat environment should also honor RUNTIME_LOCAL
	cfg, err = BuildConfig(BuildConfigInput{
		Environment: "uat",
	})

	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}

	if !cfg.UseNoOp {
		t.Error("UseNoOp = false, want true for uat with RUNTIME_LOCAL=true")
	}
}

func TestBuildConfig_RuntimeLocal_CugProd_Ignored(t *testing.T) {
	clearTelemetryEnv(t)

	os.Setenv("RUNTIME_LOCAL", "true")
	defer clearTelemetryEnv(t)

	// cug environment should ignore RUNTIME_LOCAL
	cfg, err := BuildConfig(BuildConfigInput{
		Environment: "cug",
	})

	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}

	if cfg.UseNoOp {
		t.Error("UseNoOp = true, want false for cug (RUNTIME_LOCAL ignored)")
	}

	// prod environment should ignore RUNTIME_LOCAL
	cfg, err = BuildConfig(BuildConfigInput{
		Environment: "prod",
	})

	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}

	if cfg.UseNoOp {
		t.Error("UseNoOp = true, want false for prod (RUNTIME_LOCAL ignored)")
	}
}

func TestBuildConfig_TelemetryEnabled(t *testing.T) {
	clearTelemetryEnv(t)

	os.Setenv("TELEMETRY_ENABLED", "false")
	defer clearTelemetryEnv(t)

	cfg, err := BuildConfig(BuildConfigInput{
		Environment: "dev",
	})

	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}

	if cfg.Enabled {
		t.Error("Enabled = true, want false when TELEMETRY_ENABLED=false")
	}
}

func TestBuildConfig_TracesDisabled(t *testing.T) {
	clearTelemetryEnv(t)

	os.Setenv("TELEMETRY_TRACES_ENABLED", "false")
	defer clearTelemetryEnv(t)

	cfg, err := BuildConfig(BuildConfigInput{
		Environment: "dev",
	})

	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}

	if cfg.TracesEndpoint != "" {
		t.Errorf("TracesEndpoint = %q, want empty when TELEMETRY_TRACES_ENABLED=false", cfg.TracesEndpoint)
	}
}

func TestBuildConfig_MetricsDisabled(t *testing.T) {
	clearTelemetryEnv(t)

	os.Setenv("TELEMETRY_METRICS_ENABLED", "false")
	defer clearTelemetryEnv(t)

	cfg, err := BuildConfig(BuildConfigInput{
		Environment: "dev",
	})

	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}

	if cfg.MetricsEndpoint != "" {
		t.Errorf("MetricsEndpoint = %q, want empty when TELEMETRY_METRICS_ENABLED=false", cfg.MetricsEndpoint)
	}
}

func TestResolvedConfig_IsTracesEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config *ResolvedConfig
		want   bool
	}{
		{
			name: "enabled with endpoint",
			config: &ResolvedConfig{
				Enabled:        true,
				UseNoOp:        false,
				TracesEndpoint: "https://endpoint/traces",
			},
			want: true,
		},
		{
			name: "disabled",
			config: &ResolvedConfig{
				Enabled:        false,
				UseNoOp:        false,
				TracesEndpoint: "https://endpoint/traces",
			},
			want: false,
		},
		{
			name: "noop mode",
			config: &ResolvedConfig{
				Enabled:        true,
				UseNoOp:        true,
				TracesEndpoint: "https://endpoint/traces",
			},
			want: false,
		},
		{
			name: "no endpoint",
			config: &ResolvedConfig{
				Enabled:        true,
				UseNoOp:        false,
				TracesEndpoint: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsTracesEnabled(); got != tt.want {
				t.Errorf("IsTracesEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolvedConfig_IsMetricsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config *ResolvedConfig
		want   bool
	}{
		{
			name: "enabled with endpoint",
			config: &ResolvedConfig{
				Enabled:         true,
				UseNoOp:         false,
				MetricsEndpoint: "https://endpoint/metrics",
			},
			want: true,
		},
		{
			name: "disabled",
			config: &ResolvedConfig{
				Enabled:         false,
				UseNoOp:         false,
				MetricsEndpoint: "https://endpoint/metrics",
			},
			want: false,
		},
		{
			name: "noop mode",
			config: &ResolvedConfig{
				Enabled:         true,
				UseNoOp:         true,
				MetricsEndpoint: "https://endpoint/metrics",
			},
			want: false,
		},
		{
			name: "no endpoint",
			config: &ResolvedConfig{
				Enabled:         true,
				UseNoOp:         false,
				MetricsEndpoint: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsMetricsEnabled(); got != tt.want {
				t.Errorf("IsMetricsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveHeaders(t *testing.T) {
	clearTelemetryEnv(t)

	os.Setenv("TEST_HEADERS", "key1=value1,key2=value2")
	defer os.Unsetenv("TEST_HEADERS")

	headers := resolveHeaders("TEST_HEADERS")

	if headers == nil {
		t.Fatal("headers is nil")
	}

	if headers["key1"] != "value1" {
		t.Errorf("headers[key1] = %q, want %q", headers["key1"], "value1")
	}

	if headers["key2"] != "value2" {
		t.Errorf("headers[key2] = %q, want %q", headers["key2"], "value2")
	}
}

func TestResolveDuration(t *testing.T) {
	tests := []struct {
		value        string
		defaultValue time.Duration
		want         time.Duration
	}{
		{"10s", 30 * time.Second, 10 * time.Second},
		{"1m", 30 * time.Second, 1 * time.Minute},
		{"", 30 * time.Second, 30 * time.Second},
		{"invalid", 30 * time.Second, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := resolveDuration(tt.value, tt.defaultValue)
			if got != tt.want {
				t.Errorf("resolveDuration(%q, %v) = %v, want %v", tt.value, tt.defaultValue, got, tt.want)
			}
		})
	}
}

func TestClampSampleRate(t *testing.T) {
	tests := []struct {
		rate float64
		want float64
	}{
		{0.5, 0.5},
		{1.0, 1.0},
		{0.0, 0.0},
		{-0.1, 0.0},
		{1.5, 1.0},
	}

	for _, tt := range tests {
		got := clampSampleRate(tt.rate)
		if got != tt.want {
			t.Errorf("clampSampleRate(%v) = %v, want %v", tt.rate, got, tt.want)
		}
	}
}

func TestResolveSampleRate(t *testing.T) {
	clearTelemetryEnv(t)

	// Test with client value
	got := resolveSampleRate(0.5, "", 1.0)
	if got != 0.5 {
		t.Errorf("resolveSampleRate(0.5, ...) = %v, want 0.5", got)
	}

	// Test with env var
	os.Setenv("TEST_SAMPLE_RATE", "0.3")
	defer os.Unsetenv("TEST_SAMPLE_RATE")

	got = resolveSampleRate(0, "TEST_SAMPLE_RATE", 1.0)
	if got != 0.3 {
		t.Errorf("resolveSampleRate with env = %v, want 0.3", got)
	}

	// Test with default
	os.Unsetenv("TEST_SAMPLE_RATE")
	got = resolveSampleRate(0, "TEST_SAMPLE_RATE", 1.0)
	if got != 1.0 {
		t.Errorf("resolveSampleRate with default = %v, want 1.0", got)
	}
}

func TestResolveStringValue(t *testing.T) {
	clearTelemetryEnv(t)

	// Test with client value (highest priority)
	got := resolveStringValue("client-value", "TEST_STRING", "default")
	if got != "client-value" {
		t.Errorf("resolveStringValue() = %q, want %q", got, "client-value")
	}

	// Test with env var
	os.Setenv("TEST_STRING", "env-value")
	defer os.Unsetenv("TEST_STRING")

	got = resolveStringValue("", "TEST_STRING", "default")
	if got != "env-value" {
		t.Errorf("resolveStringValue() = %q, want %q", got, "env-value")
	}

	// Test with default
	os.Unsetenv("TEST_STRING")
	got = resolveStringValue("", "TEST_STRING", "default")
	if got != "default" {
		t.Errorf("resolveStringValue() = %q, want %q", got, "default")
	}
}

func TestResolveBoolFlag(t *testing.T) {
	clearTelemetryEnv(t)

	trueVal := true
	falseVal := false

	// Test with client value (highest priority)
	got := resolveBoolFlag(&trueVal, "TEST_BOOL", false)
	if !got {
		t.Errorf("resolveBoolFlag() = %v, want %v", got, true)
	}

	got = resolveBoolFlag(&falseVal, "TEST_BOOL", true)
	if got {
		t.Errorf("resolveBoolFlag() = %v, want %v", got, false)
	}

	// Test with env var
	os.Setenv("TEST_BOOL", "true")
	defer os.Unsetenv("TEST_BOOL")

	got = resolveBoolFlag(nil, "TEST_BOOL", false)
	if !got {
		t.Errorf("resolveBoolFlag() = %v, want %v", got, true)
	}

	// Test with default
	os.Unsetenv("TEST_BOOL")
	got = resolveBoolFlag(nil, "TEST_BOOL", true)
	if !got {
		t.Errorf("resolveBoolFlag() = %v, want %v", got, true)
	}
}

// clearTelemetryEnv clears all telemetry-related environment variables
func clearTelemetryEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_INSECURE",
		"OTEL_EXPORTER_OTLP_METRICS_INSECURE",
		"OTEL_EXPORTER_OTLP_HEADERS",
		"OTEL_EXPORTER_OTLP_METRICS_HEADERS",
		"OTEL_EXPORTER_OTLP_TIMEOUT",
		"OTEL_SERVICE_NAME",
		"SERVICE_NAMESPACE",
		"TELEMETRY_ENABLED",
		"TELEMETRY_TRACES_ENABLED",
		"TELEMETRY_METRICS_ENABLED",
		"TELEMETRY_METRICS_EXPORT_INTERVAL",
		"TELEMETRY_SAMPLE_RATE",
		"RUNTIME_LOCAL",
		"ENV",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}
