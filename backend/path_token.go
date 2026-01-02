package backend

import (
	"context"
	"fmt"
	"math"
	"time"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	saUtil "github.com/skyflowapi/skyflow-go/serviceaccount/util"
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

	// Generate token
	token, err := b.generateTokenWithTimeout(ctx, config, role)
	if err != nil {
		duration := time.Since(start)
		b.metrics.recordTokenGeneration(duration, err)

		// Audit log
		b.auditLog(auditEvent{
			Timestamp: time.Now(),
			Operation: "token_generate",
			Role:      roleName,
			Success:   false,
			Duration:  duration.Milliseconds(),
			ClientIP:  req.Connection.RemoteAddr,
			Error:     err.Error(),
		})

		return logical.ErrorResponse("failed to generate token: %v", err), nil
	}

	duration := time.Since(start)
	b.metrics.recordTokenGeneration(duration, nil)

	// Structured logging
	b.logTokenOperation(logContext{
		operation: "token_generate",
		role:      roleName,
		duration:  duration,
	})

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

// generateTokenWithTimeout generates a token with context timeout
func (b *skyflowBackend) generateTokenWithTimeout(ctx context.Context, config *skyflowConfig, role *skyflowRole) (*saUtil.ResponseToken, error) {
	timeout := time.Duration(config.RequestTimeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Channel for result
	resultChan := make(chan struct {
		token *saUtil.ResponseToken
		err   error
	}, 1)

	go func() {
		token, err := b.generateToken(config, role)
		resultChan <- struct {
			token *saUtil.ResponseToken
			err   error
		}{token, err}
	}()

	select {
	case result := <-resultChan:
		return result.token, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("token generation timeout after %v: %w", timeout, ctx.Err())
	}
}

// generateToken generates a new Skyflow token with retry logic
func (b *skyflowBackend) generateToken(config *skyflowConfig, role *skyflowRole) (*saUtil.ResponseToken, error) {
	var token *saUtil.ResponseToken
	var lastErr error

	maxRetries := config.MaxRetries

	// Execute with circuit breaker
	err := b.circuitBreaker.call(func() error {
		// Retry loop with exponential backoff
		for attempt := 0; attempt <= maxRetries; attempt++ {
			// Determine which credentials to use
			if role.CredentialsFilePath != "" {
				token, lastErr = saUtil.GenerateBearerToken(role.CredentialsFilePath)
			} else if role.CredentialsJSON != "" {
				token, lastErr = saUtil.GenerateBearerTokenFromCreds(role.CredentialsJSON)
			} else if config.CredentialsFilePath != "" {
				token, lastErr = saUtil.GenerateBearerToken(config.CredentialsFilePath)
			} else if config.CredentialsJSON != "" {
				token, lastErr = saUtil.GenerateBearerTokenFromCreds(config.CredentialsJSON)
			} else {
				return fmt.Errorf("no credentials configured")
			}

			// Success case
			if lastErr == nil && token != nil {
				return nil
			}

			// Log the error
			b.Logger().Warn("token generation attempt failed",
				"attempt", attempt+1,
				"max_retries", maxRetries,
				"error", lastErr,
			)

			// Don't retry on final attempt
			if attempt < maxRetries {
				// Exponential backoff: 1s, 2s, 4s, 8s...
				backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
				time.Sleep(backoff)
			}
		}

		// All retries exhausted
		if lastErr != nil {
			return fmt.Errorf("failed to generate bearer token after %d attempts: %w", maxRetries+1, lastErr)
		}

		return fmt.Errorf("failed to generate bearer token: no token returned after %d attempts", maxRetries+1)
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}
