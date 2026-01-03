package backend

import (
	"context"
	"fmt"
	"os"
	"time"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/skyflowapi/skyflow-go/v2/serviceaccount"
	"github.com/skyflowapi/skyflow-go/v2/utils/common"
	skyflowError "github.com/skyflowapi/skyflow-go/v2/utils/error"
	"github.com/skyflowapi/skyflow-go/v2/utils/logger"
)

// pathToken returns the path configuration for token generation
func pathToken(b *skyflowBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "creds/" + framework.GenericNameRegex("name"),

			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the role",
					Required:    true,
				},
				"ctx": {
					Type:        framework.TypeString,
					Description: "Context data to include in the token (optional)",
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathTokenRead,
					Summary:  "Generate a Skyflow bearer token for the specified role.",
				},
			},

			HelpSynopsis:    "Generate Skyflow bearer token.",
			HelpDescription: "Generate a bearer token for authenticating with Skyflow APIs using the specified role configuration.",
		},
	}
}

// pathTokenRead handles token generation
func (b *skyflowBackend) pathTokenRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	start := time.Now()
	roleName := data.Get("name").(string)

	// Get optional context data
	ctxData := ""
	if val, ok := data.GetOk("ctx"); ok {
		ctxData = val.(string)
	}

	// Start telemetry span
	if b.emitter != nil {
		ctx = b.emitter.EmitTokenRequest(ctx, roleName)
		defer b.emitter.EndTokenSpan(ctx)
	}

	// Get role
	role, err := b.getRole(ctx, req.Storage, roleName)
	if err != nil {
		return nil, err
	}

	if role == nil {
		return logical.ErrorResponse("role %q not found", roleName), nil
	}

	// Get config
	config, err := b.getConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return logical.ErrorResponse("backend not configured"), nil
	}

	// Generate token using config credentials and role's Skyflow role IDs
	token, tokenErr := b.generateToken(config, role, ctxData)
	duration := time.Since(start)

	if tokenErr != nil {
		// Record telemetry failure
		if b.emitter != nil {
			b.emitter.EmitTokenFailure(ctx, roleName, tokenErr, duration)
		}

		// Audit log
		b.auditLog(auditEvent{
			Timestamp: time.Now(),
			Operation: "token_generate",
			Role:      roleName,
			Success:   false,
			Duration:  duration.Milliseconds(),
			ClientIP:  req.Connection.RemoteAddr,
			Error:     tokenErr.Error(),
		})

		return logical.ErrorResponse("failed to generate token: %v", tokenErr), nil
	}

	// Record telemetry success
	if b.emitter != nil {
		b.emitter.EmitTokenSuccess(ctx, roleName, duration)
	}

	// Audit log
	b.auditLog(auditEvent{
		Timestamp: time.Now(),
		Operation: "token_generate",
		Role:      roleName,
		Success:   true,
		Duration:  duration.Milliseconds(),
		ClientIP:  req.Connection.RemoteAddr,
	})

	b.Logger().Info("token generated", "role", roleName, "duration_ms", duration.Milliseconds())

	return &logical.Response{
		Data: map[string]interface{}{
			"access_token": token.AccessToken,
			"token_type":   token.TokenType,
		},
	}, nil
}

// generateToken generates a Skyflow token using config credentials and role's Skyflow role IDs
func (b *skyflowBackend) generateToken(config *skyflowConfig, role *skyflowRole, ctxData string) (token *common.TokenResponse, returnErr error) {
	// Recover from SDK panics - defensive measure
	defer func() {
		if r := recover(); r != nil {
			returnErr = fmt.Errorf("token generation panic: %v", r)
		}
	}()

	var sdkErr *skyflowError.SkyflowError

	// Use role's Skyflow role IDs for scoped token generation
	opts := common.BearerTokenOptions{
		LogLevel: logger.DEBUG,
		RoleIDs:  role.RoleIDs,
		Ctx:      ctxData,
	}

	// Use config credentials (file path or JSON)
	if config.CredentialsFilePath != "" {
		if _, statErr := os.Stat(config.CredentialsFilePath); os.IsNotExist(statErr) {
			return nil, fmt.Errorf("credentials file not found: %s: %w", config.CredentialsFilePath, statErr)
		}
		token, sdkErr = serviceaccount.GenerateBearerToken(config.CredentialsFilePath, opts)
	} else if config.CredentialsJSON != "" {
		token, sdkErr = serviceaccount.GenerateBearerTokenFromCreds(config.CredentialsJSON, opts)
	} else {
		return nil, fmt.Errorf("no credentials configured")
	}

	if sdkErr != nil {
		return nil, fmt.Errorf("failed to generate bearer token: %w", sdkErr)
	}

	if token == nil || token.AccessToken == "" {
		return nil, fmt.Errorf("token generation returned empty token")
	}

	return token, nil
}
