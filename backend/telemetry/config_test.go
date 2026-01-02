package telemetry

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ServiceName != "skyflow-vault-plugin" {
		t.Errorf("expected service name 'skyflow-vault-plugin', got '%s'", cfg.ServiceName)
	}

	if cfg.ServiceNamespace != "skyflow" {
		t.Errorf("expected service namespace 'skyflow', got '%s'", cfg.ServiceNamespace)
	}

	if !cfg.Enabled {
		t.Error("expected telemetry to be enabled by default")
	}

	if !cfg.TracesEnabled {
		t.Error("expected traces to be enabled by default")
	}

	if !cfg.MetricsEnabled {
		t.Error("expected metrics to be enabled by default")
	}

	if cfg.SampleRate != 1.0 {
		t.Errorf("expected sample rate 1.0, got %f", cfg.SampleRate)
	}

	if cfg.MetricsExportInterval != 60*time.Second {
		t.Errorf("expected metrics export interval 60s, got %v", cfg.MetricsExportInterval)
	}
}

func TestConfigFromEnv(t *testing.T) {
	// Save original env vars
	origEnabled := os.Getenv("TELEMETRY_ENABLED")
	origServiceName := os.Getenv("OTEL_SERVICE_NAME")
	origEnv := os.Getenv("ENV")

	// Set test env vars
	os.Setenv("TELEMETRY_ENABLED", "true")
	os.Setenv("OTEL_SERVICE_NAME", "test-service")
	os.Setenv("ENV", "test")

	defer func() {
		os.Setenv("TELEMETRY_ENABLED", origEnabled)
		os.Setenv("OTEL_SERVICE_NAME", origServiceName)
		os.Setenv("ENV", origEnv)
	}()

	cfg := ConfigFromEnv()

	if cfg.ServiceName != "test-service" {
		t.Errorf("expected service name 'test-service', got '%s'", cfg.ServiceName)
	}

	if cfg.Environment != "test" {
		t.Errorf("expected environment 'test', got '%s'", cfg.Environment)
	}

	if !cfg.Enabled {
		t.Error("expected telemetry to be enabled")
	}
}

func TestConfigFromEnv_Disabled(t *testing.T) {
	origEnabled := os.Getenv("TELEMETRY_ENABLED")
	os.Setenv("TELEMETRY_ENABLED", "false")
	defer os.Setenv("TELEMETRY_ENABLED", origEnabled)

	cfg := ConfigFromEnv()

	if cfg.Enabled {
		t.Error("expected telemetry to be disabled")
	}
}

func TestConfig_IsTracesEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "All enabled with endpoint",
			config: &Config{
				Enabled:        true,
				TracesEnabled:  true,
				TracesEndpoint: "http://localhost:4318",
			},
			expected: true,
		},
		{
			name: "Master switch disabled",
			config: &Config{
				Enabled:        false,
				TracesEnabled:  true,
				TracesEndpoint: "http://localhost:4318",
			},
			expected: false,
		},
		{
			name: "Traces disabled",
			config: &Config{
				Enabled:        true,
				TracesEnabled:  false,
				TracesEndpoint: "http://localhost:4318",
			},
			expected: false,
		},
		{
			name: "No endpoint",
			config: &Config{
				Enabled:        true,
				TracesEnabled:  true,
				TracesEndpoint: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsTracesEnabled()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConfig_IsMetricsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "All enabled with endpoint",
			config: &Config{
				Enabled:         true,
				MetricsEnabled:  true,
				MetricsEndpoint: "http://localhost:4318",
			},
			expected: true,
		},
		{
			name: "Master switch disabled",
			config: &Config{
				Enabled:         false,
				MetricsEnabled:  true,
				MetricsEndpoint: "http://localhost:4318",
			},
			expected: false,
		},
		{
			name: "Metrics disabled",
			config: &Config{
				Enabled:         true,
				MetricsEnabled:  false,
				MetricsEndpoint: "http://localhost:4318",
			},
			expected: false,
		},
		{
			name: "No endpoint",
			config: &Config{
				Enabled:         true,
				MetricsEnabled:  true,
				MetricsEndpoint: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsMetricsEnabled()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: nil,
		},
		{
			name:  "Single header",
			input: "key=value",
			expected: map[string]string{
				"key": "value",
			},
		},
		{
			name:  "Multiple headers",
			input: "key1=value1,key2=value2",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:  "Header with spaces",
			input: "key1=value1, key2=value2",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHeaders(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d headers, got %d", len(tt.expected), len(result))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected header %s=%s, got %s", k, v, result[k])
				}
			}
		})
	}
}

