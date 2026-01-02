package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"github.com/hashicorp/vault/sdk/logical"
	saUtil "github.com/skyflowapi/skyflow-go/serviceaccount/util"
)

// skyflowConfig represents the backend configuration
type skyflowConfig struct {
	// Credentials - one of these must be provided
	CredentialsFilePath string `json:"credentials_file_path,omitempty"`
	CredentialsJSON     string `json:"credentials_json,omitempty"`

	// Advanced settings
	MaxRetries     int `json:"max_retries"`
	RequestTimeout int `json:"request_timeout"`

	// Metadata
	Description string    `json:"description,omitempty"`
	Version     int       `json:"version"`
	LastUpdated time.Time `json:"last_updated"`
	UpdatedBy   string    `json:"updated_by,omitempty"`
}

// defaultConfig returns a config with default values
func defaultConfig() *skyflowConfig {
	return &skyflowConfig{
		MaxRetries:     3,
		RequestTimeout: 30, // 30 seconds
		Version:        1,
		LastUpdated:    time.Now(),
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

	// Validate retry settings
	if c.MaxRetries < 0 || c.MaxRetries > 10 {
		return fmt.Errorf("max_retries must be between 0 and 10")
	}

	// Validate timeout
	if c.RequestTimeout < 1 || c.RequestTimeout > 300 {
		return fmt.Errorf("request_timeout must be between 1 and 300 seconds")
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
		"updated_by":  config.UpdatedBy,
		"description": config.Description,
	})
	if err != nil {
		return fmt.Errorf("failed to create history entry: %w", err)
	}

	if err := s.Put(ctx, historyEntry); err != nil {
		b.Logger().Warn("failed to save config history", "error", err)
		// Don't fail the operation if history save fails
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
func (c *skyflowConfig) validateCredentials() error {
	var token *saUtil.ResponseToken
	var err error

	// Try to generate a token to validate credentials
	if c.CredentialsFilePath != "" {
		token, err = saUtil.GenerateBearerToken(c.CredentialsFilePath)
	} else if c.CredentialsJSON != "" {
		token, err = saUtil.GenerateBearerTokenFromCreds(c.CredentialsJSON)
	}

	if err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}

	if token == nil {
		return fmt.Errorf("credential validation failed: no token returned")
	}

	return nil
}
