package backend

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestConfig_DefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("expected default max_retries 3, got %d", cfg.MaxRetries)
	}

	if cfg.RequestTimeout != 30 {
		t.Errorf("expected default request_timeout 30, got %d", cfg.RequestTimeout)
	}

	if cfg.Version != 1 {
		t.Errorf("expected default version 1, got %d", cfg.Version)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *skyflowConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid config with file path",
			config: &skyflowConfig{
				CredentialsFilePath: "/path/to/creds.json",
				MaxRetries:          3,
				RequestTimeout:      30,
			},
			wantError: false,
		},
		{
			name: "Valid config with JSON",
			config: &skyflowConfig{
				CredentialsJSON: `{"key": "value"}`,
				MaxRetries:      3,
				RequestTimeout:  30,
			},
			wantError: false,
		},
		{
			name: "No credentials",
			config: &skyflowConfig{
				MaxRetries:     3,
				RequestTimeout: 30,
			},
			wantError: true,
			errorMsg:  "either credentials_file_path or credentials_json must be provided",
		},
		{
			name: "Both credentials provided",
			config: &skyflowConfig{
				CredentialsFilePath: "/path/to/creds.json",
				CredentialsJSON:     `{"key": "value"}`,
				MaxRetries:          3,
				RequestTimeout:      30,
			},
			wantError: true,
			errorMsg:  "only one of credentials_file_path or credentials_json can be provided",
		},
		{
			name: "Invalid JSON",
			config: &skyflowConfig{
				CredentialsJSON: `{invalid json}`,
				MaxRetries:      3,
				RequestTimeout:  30,
			},
			wantError: true,
			errorMsg:  "credentials_json must be valid JSON",
		},
		{
			name: "Negative max retries",
			config: &skyflowConfig{
				CredentialsFilePath: "/path/to/creds.json",
				MaxRetries:          -1,
				RequestTimeout:      30,
			},
			wantError: true,
			errorMsg:  "max_retries must be between 0 and 10",
		},
		{
			name: "Max retries too high",
			config: &skyflowConfig{
				CredentialsFilePath: "/path/to/creds.json",
				MaxRetries:          11,
				RequestTimeout:      30,
			},
			wantError: true,
			errorMsg:  "max_retries must be between 0 and 10",
		},
		{
			name: "Request timeout too low",
			config: &skyflowConfig{
				CredentialsFilePath: "/path/to/creds.json",
				MaxRetries:          3,
				RequestTimeout:      0,
			},
			wantError: true,
			errorMsg:  "request_timeout must be between 1 and 300 seconds",
		},
		{
			name: "Request timeout too high",
			config: &skyflowConfig{
				CredentialsFilePath: "/path/to/creds.json",
				MaxRetries:          3,
				RequestTimeout:      301,
			},
			wantError: true,
			errorMsg:  "request_timeout must be between 1 and 300 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfig_GetSaveDelete(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}

	config := &logical.BackendConfig{
		Logger:      nil,
		System:      &logical.StaticSystemView{},
		StorageView: storage,
	}

	b, err := Factory(ctx, config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	backend := b.(*skyflowBackend)

	t.Run("Get non-existent config", func(t *testing.T) {
		cfg, err := backend.getConfig(ctx, storage)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg != nil {
			t.Error("expected nil config for non-existent entry")
		}
	})

	t.Run("Save and get config", func(t *testing.T) {
		testConfig := &skyflowConfig{
			CredentialsFilePath: "/test/path.json",
			MaxRetries:          5,
			RequestTimeout:      60,
			Description:         "Test config",
		}

		err := backend.saveConfig(ctx, storage, testConfig)
		if err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		cfg, err := backend.getConfig(ctx, storage)
		if err != nil {
			t.Fatalf("failed to get config: %v", err)
		}

		if cfg == nil {
			t.Fatal("config should not be nil")
		}

		if cfg.CredentialsFilePath != testConfig.CredentialsFilePath {
			t.Errorf("expected credentials_file_path '%s', got '%s'",
				testConfig.CredentialsFilePath, cfg.CredentialsFilePath)
		}

		if cfg.Description != testConfig.Description {
			t.Errorf("expected description '%s', got '%s'",
				testConfig.Description, cfg.Description)
		}
	})

	t.Run("Delete config", func(t *testing.T) {
		err := backend.deleteConfig(ctx, storage)
		if err != nil {
			t.Fatalf("failed to delete config: %v", err)
		}

		cfg, err := backend.getConfig(ctx, storage)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg != nil {
			t.Error("config should be nil after deletion")
		}
	})
}

func TestConfig_JSONValidation(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantError bool
	}{
		{
			name:      "Valid JSON object",
			json:      `{"key": "value", "number": 123}`,
			wantError: false,
		},
		{
			name:      "Valid JSON array",
			json:      `[1, 2, 3]`,
			wantError: false,
		},
		{
			name:      "Invalid JSON",
			json:      `{key: value}`,
			wantError: true,
		},
		{
			name:      "Empty JSON",
			json:      `{}`,
			wantError: false,
		},
		{
			name:      "Malformed JSON",
			json:      `{"key": }`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &skyflowConfig{
				CredentialsJSON: tt.json,
				MaxRetries:      3,
				RequestTimeout:  30,
			}

			err := cfg.validate()
			if tt.wantError && err == nil {
				t.Error("expected error for invalid JSON")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
