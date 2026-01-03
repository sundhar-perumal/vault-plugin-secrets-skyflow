package backend

import (
	"context"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// pathConfig returns the path configuration for managing backend config
func pathConfig(b *skyflowBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "config",

			Fields: map[string]*framework.FieldSchema{
				"credentials_file_path": {
					Type:        framework.TypeString,
					Description: "Path to Skyflow service account credentials JSON file",
				},
				"credentials_json": {
					Type:        framework.TypeString,
					Description: "Skyflow service account credentials as JSON string",
				},
				"description": {
					Type:        framework.TypeString,
					Description: "Description of this Skyflow configuration",
				},
				"tags": {
					Type:        framework.TypeCommaStringSlice,
					Description: "Tags for organizing configurations",
				},
				"validate_credentials": {
					Type:        framework.TypeBool,
					Description: "Validate credentials by generating a test token (default: true)",
					Default:     true,
				},
			},

			ExistenceCheck: b.pathConfigExistenceCheck,

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigWrite,
					Summary:  "Configure the Skyflow backend with service account credentials.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigWrite,
					Summary:  "Update the Skyflow backend configuration.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigRead,
					Summary:  "Read the current Skyflow backend configuration.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathConfigDelete,
					Summary:  "Delete the Skyflow backend configuration.",
				},
			},

			HelpSynopsis:    "Configure the Skyflow secrets engine.",
			HelpDescription: "Configure credentials for Skyflow token generation.",
		},
	}
}

// pathConfigExistenceCheck checks if config exists
func (b *skyflowBackend) pathConfigExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	config, err := b.getConfig(ctx, req.Storage)
	if err != nil {
		return false, err
	}

	return config != nil, nil
}

// pathConfigWrite handles create and update operations for config
func (b *skyflowBackend) pathConfigWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	operation := "create"
	if req.Operation == logical.UpdateOperation {
		operation = "update"
	}

	config := defaultConfig()

	// Load existing config if updating
	if req.Operation == logical.UpdateOperation {
		existingConfig, err := b.getConfig(ctx, req.Storage)
		if err != nil {
			return nil, err
		}
		if existingConfig != nil {
			config = existingConfig
		}
	}

	// Update fields from request
	if credPath, ok := data.GetOk("credentials_file_path"); ok {
		config.CredentialsFilePath = credPath.(string)
		config.CredentialsJSON = "" // Clear JSON if file path is set
	}

	if credJSON, ok := data.GetOk("credentials_json"); ok {
		config.CredentialsJSON = credJSON.(string)
		config.CredentialsFilePath = "" // Clear file path if JSON is set
	}

	if desc, ok := data.GetOk("description"); ok {
		config.Description = desc.(string)
	}

	if tags, ok := data.GetOk("tags"); ok {
		config.Tags = tags.([]string)
	}

	// Validate configuration
	if err := config.validate(); err != nil {
		return logical.ErrorResponse("invalid configuration: %s", err.Error()), nil
	}

	// Validate credentials if requested
	validateCreds := true
	if val, ok := data.GetOk("validate_credentials"); ok {
		validateCreds = val.(bool)
	}

	if validateCreds {
		b.Logger().Info("validating credentials")
		if err := config.validateCredentials(); err != nil {
			return logical.ErrorResponse("credential validation failed: %s", err.Error()), nil
		}
		b.Logger().Info("credentials validated successfully")
	}

	// Save configuration with history
	if err := b.saveConfigWithHistory(ctx, req.Storage, config); err != nil {
		return nil, err
	}

	// Emit telemetry
	if b.emitter != nil {
		b.emitter.EmitConfigWrite(ctx, operation, true)
	}

	b.Logger().Info("configuration updated",
		"operation", req.Operation,
		"version", config.Version,
	)

	return nil, nil
}

// pathConfigRead handles read operations for config
func (b *skyflowBackend) pathConfigRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.getConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, nil
	}

	// Don't return sensitive credentials, only metadata
	responseData := map[string]interface{}{
		"credentials_configured": true,
		"description":            config.Description,
		"tags":                   config.Tags,
		"version":                config.Version,
		"last_updated":           config.LastUpdated.Format(time.RFC3339),
	}

	if config.CredentialsFilePath != "" {
		responseData["credentials_type"] = "file_path"
		responseData["credentials_file_path"] = config.CredentialsFilePath
	} else {
		responseData["credentials_type"] = "json"
	}

	return &logical.Response{
		Data: responseData,
	}, nil
}

// pathConfigDelete handles delete operations for config
func (b *skyflowBackend) pathConfigDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	if err := b.deleteConfig(ctx, req.Storage); err != nil {
		return nil, err
	}

	b.Logger().Info("configuration deleted")

	return nil, nil
}
