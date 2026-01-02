package backend

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// pathMetrics returns the path configuration for metrics
func pathMetrics(b *skyflowBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "metrics$",

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathMetricsRead,
					Summary:  "Get detailed metrics.",
				},
			},

			HelpSynopsis:    "Get detailed metrics.",
			HelpDescription: "Retrieve comprehensive performance metrics including token generation and circuit breaker status.",
		},
	}
}

// pathMetricsRead returns detailed metrics
func (b *skyflowBackend) pathMetricsRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	// Get basic metrics
	stats := b.metrics.getStats()

	// Add circuit breaker stats
	stats["circuit_breaker"] = b.circuitBreaker.getStats()

	return &logical.Response{
		Data: stats,
	}, nil
}
