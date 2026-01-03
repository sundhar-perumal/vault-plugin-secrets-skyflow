package backend

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestConfig_DefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.Version != 1 {
		t.Errorf("expected default version 1, got %d", cfg.Version)
	}

	if cfg.LastUpdated.IsZero() {
		t.Error("expected LastUpdated to be set")
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
			},
			wantError: false,
		},
		{
			name: "Valid config with JSON",
			config: &skyflowConfig{
				CredentialsJSON: `{"key": "value"}`,
			},
			wantError: false,
		},
		{
			name:      "No credentials",
			config:    &skyflowConfig{},
			wantError: true,
			errorMsg:  "either credentials_file_path or credentials_json must be provided",
		},
		{
			name: "Both credentials provided",
			config: &skyflowConfig{
				CredentialsFilePath: "/path/to/creds.json",
				CredentialsJSON:     `{"key": "value"}`,
			},
			wantError: true,
			errorMsg:  "only one of credentials_file_path or credentials_json can be provided",
		},
		{
			name: "Invalid JSON",
			config: &skyflowConfig{
				CredentialsJSON: `{invalid json}`,
			},
			wantError: true,
			errorMsg:  "credentials_json must be valid JSON",
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

func TestConfig_ValidateCredentials(t *testing.T) {
	tests := []struct {
		name   string
		config *skyflowConfig
	}{
		{
			name: "Non-existent file path",
			config: &skyflowConfig{
				CredentialsFilePath: "/non/existent/file.json",
			},
		},
		{
			name: "Invalid JSON credentials",
			config: &skyflowConfig{
				CredentialsJSON: `{"invalid": "credentials"}`,
			},
		},
		{
			name: "Empty JSON credentials",
			config: &skyflowConfig{
				CredentialsJSON: `{}`,
			},
		},
		{
			name: "Malformed credentials structure",
			config: &skyflowConfig{
				CredentialsJSON: `{"clientID": "", "privateKey": ""}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should NOT panic, but return an error gracefully
			err := tt.config.validateCredentials()
			if err == nil {
				t.Error("expected error for invalid credentials, got nil")
			}
			t.Logf("Got expected error: %v", err)
		})
	}
}

func TestConfig_ValidateCredentials_NoCredentials(t *testing.T) {
	// Test with no credentials set - should return error without panic
	config := &skyflowConfig{}

	err := config.validateCredentials()
	// With no credentials, token will be nil and should return error
	if err == nil {
		t.Error("expected error when no credentials provided")
	}
	t.Logf("Got expected error: %v", err)
}
