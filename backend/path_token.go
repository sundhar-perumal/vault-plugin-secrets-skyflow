package backend

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/skyflowapi/skyflow-go/v2/serviceaccount"
	"github.com/skyflowapi/skyflow-go/v2/utils/common"
	skyflowError "github.com/skyflowapi/skyflow-go/v2/utils/error"
	"github.com/skyflowapi/skyflow-go/v2/utils/logger"
	"github.com/sundhar-perumal/vault-plugin-secrets-skyflow/backend/telemetry"
	"go.opentelemetry.io/otel/trace"
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
	traces := b.traces()

	// Debug log: Environment variables for telemetry debugging
	b.Logger().Debug("token request ENV debug",
		"role", roleName,
		"ENV", os.Getenv("ENV"),
		"TELEMETRY_ENABLED", os.Getenv("TELEMETRY_ENABLED"),
		"RUNTIME_LOCAL", os.Getenv("RUNTIME_LOCAL"),
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"),
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"),
		"telemetry_providers_nil", b.telemetryProviders == nil,
		"traces_nil", traces == nil,
		"traces_enabled", traces != nil && traces.IsEnabled(),
	)

	// Extract trace context from traceparent header (W3C standard)
	ctx = telemetry.ExtractTraceContext(ctx, req.Headers)

	// Extract skyflowVaultName from mount point (e.g., "skyflow/order/" -> "order")
	skyflowVaultName := "unknown"
	parts := strings.Split(strings.Trim(req.MountPoint, "/"), "/")
	if len(parts) > 0 {
		skyflowVaultName = parts[len(parts)-1]
	}

	// Extract vaultServiceName from header (sent by client)
	vaultServiceName := "direct"
	if vals, ok := req.Headers["Application-Source"]; ok && len(vals) > 0 && vals[0] != "" {
		vaultServiceName = vals[0]
	}

	// Get optional context data
	ctxData := ""
	if val, ok := data.GetOk("ctx"); ok {
		ctxData = val.(string)
	}

	// Start telemetry span (inherits parent span from extracted trace context)
	ctx, span := traces.StartTokenGenerate(ctx, roleName)
	defer span.End()

	b.Logger().Debug("token request received", "role", roleName)

	// Get role
	role, err := b.getRole(ctx, req.Storage, roleName)
	if err != nil {
		traces.RecordTokenFailed(span, float64(time.Since(start).Milliseconds()), err)
		return nil, err
	}

	if role == nil {
		traces.RecordTokenFailed(span, float64(time.Since(start).Milliseconds()), fmt.Errorf("role not found"))
		return logical.ErrorResponse("role %q not found", roleName), nil
	}

	// Get config
	config, err := b.getConfig(ctx, req.Storage)
	if err != nil {
		traces.RecordTokenFailed(span, float64(time.Since(start).Milliseconds()), err)
		return nil, err
	}

	if config == nil {
		traces.RecordTokenFailed(span, float64(time.Since(start).Milliseconds()), fmt.Errorf("backend not configured"))
		return logical.ErrorResponse("backend not configured"), nil
	}

	// Determine credential type for telemetry
	credentialType := "json"
	if config.CredentialsFilePath != "" {
		credentialType = "file_path"
	}

	// Start inner span for Skyflow SDK authentication
	ctx, sdkSpan := traces.StartSDKAuth(ctx, roleName, credentialType, len(role.RoleIDs))

	// Generate token using config credentials and role's Skyflow role IDs
	sdkCallStart := time.Now()
	token, tokenErr := b.generateToken(config, role, ctxData)
	sdkCallDuration := time.Since(sdkCallStart)
	duration := time.Since(start)

	// End SDK auth span
	if tokenErr != nil {
		traces.RecordSDKAuthFailed(sdkSpan, float64(sdkCallDuration.Milliseconds()), tokenErr)
	} else {
		traces.RecordSDKAuthSuccess(sdkSpan, float64(sdkCallDuration.Milliseconds()))
	}
	sdkSpan.End()

	if tokenErr != nil {
		// Record telemetry failure
		traces.RecordTokenFailed(span, float64(duration.Milliseconds()), tokenErr)

		// Record metrics
		if m := b.metrics(); m != nil {
			m.RecordTokenGenerate(ctx, roleName, vaultServiceName, skyflowVaultName, float64(duration.Milliseconds()), false)
			m.RecordTokenError(ctx, roleName, vaultServiceName, skyflowVaultName, "generation_failed")
		}

		// Audit log
		traceID := trace.SpanContextFromContext(ctx).TraceID().String()
		b.auditLog(auditEvent{
			Timestamp: time.Now(),
			Operation: "token_generate",
			Role:      roleName,
			Success:   false,
			Duration:  duration.Milliseconds(),
			ClientIP:  req.Connection.RemoteAddr,
			TraceID:   traceID,
			Error:     tokenErr.Error(),
		})

		return logical.ErrorResponse("failed to generate token: %v", tokenErr), nil
	}

	// Record telemetry success
	traces.RecordTokenGenerated(span, float64(duration.Milliseconds()))

	// Record metrics
	if m := b.metrics(); m != nil {
		m.RecordTokenGenerate(ctx, roleName, vaultServiceName, skyflowVaultName, float64(duration.Milliseconds()), true)
		m.RecordSkyflowSDKCall(ctx, roleName, "success", float64(sdkCallDuration.Milliseconds()))
	}

	// Audit log
	traceID := trace.SpanContextFromContext(ctx).TraceID().String()
	b.auditLog(auditEvent{
		Timestamp: time.Now(),
		Operation: "token_generate",
		Role:      roleName,
		Success:   true,
		Duration:  duration.Milliseconds(),
		ClientIP:  req.Connection.RemoteAddr,
		TraceID:   traceID,
	})

	b.Logger().Info("token generated", "role", roleName, "trace_id", traceID, "duration_ms", duration.Milliseconds())

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
