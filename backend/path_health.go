package backend

import (
	"context"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// pathHealth returns the path configuration for health checks
func pathHealth(b *skyflowBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "health$",

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathHealthRead,
					Summary:  "Health check endpoint.",
				},
			},

			HelpSynopsis:    "Health check endpoint.",
			HelpDescription: "Returns health status of the plugin including configuration and connectivity checks.",
		},
	}
}

// pathHealthRead performs health checks
func (b *skyflowBackend) pathHealthRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	response := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Check configuration
	config, err := b.getConfig(ctx, req.Storage)
	if err != nil {
		response["healthy"] = false
		response["error"] = "failed to load configuration"
		response["details"] = err.Error()
		return &logical.Response{Data: response}, nil
	}

	if config == nil {
		response["healthy"] = false
		response["error"] = "backend not configured"
		return &logical.Response{Data: response}, nil
	}

	response["configuration_status"] = "ok"

	// Test token generation with a test role
	testRole := defaultRole("health-check")
	start := time.Now()

	_, err = b.generateToken(config, testRole)
	duration := time.Since(start)

	response["response_time_ms"] = duration.Milliseconds()

	if err != nil {
		response["healthy"] = false
		response["connectivity_status"] = "failed"
		response["error"] = err.Error()
	} else {
		response["healthy"] = true
		response["connectivity_status"] = "ok"
	}

	// Add circuit breaker status
	response["circuit_breaker"] = b.circuitBreaker.getStats()

	return &logical.Response{Data: response}, nil
}
