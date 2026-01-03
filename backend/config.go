package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/skyflowapi/skyflow-go/v2/serviceaccount"
	"github.com/skyflowapi/skyflow-go/v2/utils/common"
	skyflowError "github.com/skyflowapi/skyflow-go/v2/utils/error"
	"github.com/skyflowapi/skyflow-go/v2/utils/logger"
)

// skyflowConfig represents the backend configuration
type skyflowConfig struct {
	// Credentials - one of these must be provided
	CredentialsFilePath string `json:"credentials_file_path,omitempty"`
	CredentialsJSON     string `json:"credentials_json,omitempty"`

	// Metadata
	Description string    `json:"description,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Version     int       `json:"version"`
	LastUpdated time.Time `json:"last_updated"`
}

// defaultConfig returns a config with default values
func defaultConfig() *skyflowConfig {
	return &skyflowConfig{
		Version:     1,
		LastUpdated: time.Now(),
	}
}

// validate checks if the configuration is valid
func (c *skyflowConfig) validate() error {
	// Must have exactly one credential source
	if c.CredentialsFilePath == "" && c.CredentialsJSON == "" {
		return fmt.Errorf("either credentials_file_path or credentials_json must be provided")
	}

	if c.CredentialsFilePath != "" && c.CredentialsJSON != "" {
		return fmt.Errorf("only one of credentials_file_path or credentials_json can be provided")
	}

	// Validate JSON format if provided
	if c.CredentialsJSON != "" {
		var js json.RawMessage
		if err := json.Unmarshal([]byte(c.CredentialsJSON), &js); err != nil {
			return fmt.Errorf("credentials_json must be valid JSON: %w", err)
		}
	}

	return nil
}

// getConfig retrieves the backend configuration from storage
func (b *skyflowBackend) getConfig(ctx context.Context, s logical.Storage) (*skyflowConfig, error) {
	entry, err := s.Get(ctx, "config")
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	if entry == nil {
		return nil, nil
	}

	config := &skyflowConfig{}
	if err := entry.DecodeJSON(config); err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	return config, nil
}

// saveConfig stores the configuration in Vault storage
func (b *skyflowBackend) saveConfig(ctx context.Context, s logical.Storage, config *skyflowConfig) error {
	entry, err := logical.StorageEntryJSON("config", config)
	if err != nil {
		return fmt.Errorf("failed to create storage entry: %w", err)
	}

	if err := s.Put(ctx, entry); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// saveConfigWithHistory stores config and maintains version history
func (b *skyflowBackend) saveConfigWithHistory(ctx context.Context, s logical.Storage, config *skyflowConfig) error {
	// Increment version
	config.Version++
	config.LastUpdated = time.Now()

	// Save current config
	if err := b.saveConfig(ctx, s, config); err != nil {
		return err
	}

	// Save to history
	historyKey := fmt.Sprintf("config_history/%d", config.Version)
	historyEntry, err := logical.StorageEntryJSON(historyKey, map[string]interface{}{
		"version":     config.Version,
		"timestamp":   config.LastUpdated.Format(time.RFC3339),
		"description": config.Description,
	})
	if err != nil {
		return fmt.Errorf("failed to create history entry: %w", err)
	}

	if err := s.Put(ctx, historyEntry); err != nil {
		b.Logger().Warn("failed to save config history", "error", err)
	}

	return nil
}

// deleteConfig removes the configuration from storage
func (b *skyflowBackend) deleteConfig(ctx context.Context, s logical.Storage) error {
	if err := s.Delete(ctx, "config"); err != nil {
		return fmt.Errorf("failed to delete configuration: %w", err)
	}

	return nil
}

// validateCredentials tests that credentials can generate tokens
func (c *skyflowConfig) validateCredentials() (returnErr error) {
	// Recover from SDK panics - defensive measure for unexpected SDK behavior
	defer func() {
		if r := recover(); r != nil {
			returnErr = fmt.Errorf("credential validation panic: %v", r)
		}
	}()

	var token *common.TokenResponse
	var sdkErr *skyflowError.SkyflowError

	opts := common.BearerTokenOptions{LogLevel: logger.DEBUG}

	// Try to generate a token to validate credentials
	if c.CredentialsFilePath != "" {
		if _, statErr := os.Stat(c.CredentialsFilePath); os.IsNotExist(statErr) {
			return fmt.Errorf("credentials file not found: %s: %w", c.CredentialsFilePath, statErr)
		}
		token, sdkErr = serviceaccount.GenerateBearerToken(c.CredentialsFilePath, opts)
	} else if c.CredentialsJSON != "" {
		token, sdkErr = serviceaccount.GenerateBearerTokenFromCreds(c.CredentialsJSON, opts)
	}

	if sdkErr != nil {
		return fmt.Errorf("credential validation failed: %w", sdkErr)
	}

	if token == nil || token.AccessToken == "" {
		return fmt.Errorf("credential validation failed: no token returned")
	}

	return nil
}
