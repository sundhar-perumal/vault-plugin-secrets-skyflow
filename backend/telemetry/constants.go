package telemetry

import "go.opentelemetry.io/otel/attribute"

// ============================================================================
// Tracer Name
// ============================================================================

// TracerName is the instrumentation name for this library
const TracerName = "github.com/sundhar-perumal/vault-plugin-secrets-skyflow"

// ============================================================================
// Span Names - Token Operations
// ============================================================================

const (
	SpanSkyflowPluginTokenGenerate = "SkyflowPlugin.Token.Generate"
	SpanSkyflowPluginSDKAuth       = "SkyflowPlugin.SDK.Auth"
)

// ============================================================================
// Span Names - Config Operations
// ============================================================================

const (
	SpanSkyflowPluginConfigWrite = "SkyflowPlugin.Config.Write"
	SpanSkyflowPluginConfigRead  = "SkyflowPlugin.Config.Read"
)

// ============================================================================
// Span Names - Role Operations
// ============================================================================

const (
	SpanSkyflowPluginRoleWrite  = "SkyflowPlugin.Role.Write"
	SpanSkyflowPluginRoleRead   = "SkyflowPlugin.Role.Read"
	SpanSkyflowPluginRoleList   = "SkyflowPlugin.Role.List"
	SpanSkyflowPluginRoleDelete = "SkyflowPlugin.Role.Delete"
)

// ============================================================================
// Span Names - Health Check
// ============================================================================

const (
	SpanSkyflowPluginHealthCheck = "SkyflowPlugin.Health.Check"
)

// ============================================================================
// Status Messages
// ============================================================================

const (
	StatusNotConfigured = "not_configured"
)

// ============================================================================
// Event Names
// ============================================================================

const (
	// Token events
	EventTokenGenerated = "token.generated"
	EventTokenFailed    = "token.failed"

	// SDK auth events
	EventSDKAuthStart   = "sdk.auth.start"
	EventSDKAuthSuccess = "sdk.auth.success"
	EventSDKAuthFailed  = "sdk.auth.failed"

	// Config events
	EventConfigUpdated = "config.updated"
	EventConfigFailed  = "config.failed"

	// Role events
	EventRoleUpdated = "role.updated"
	EventRoleFailed  = "role.failed"

	// Error
	EventError = "error"
)

// ============================================================================
// Attribute Keys
// ============================================================================

var (
	// Role attributes
	AttrRole           = attribute.Key("skyflow.role")
	AttrCredentialType = attribute.Key("credential_type")
	AttrRoleIDsCount   = attribute.Key("role_ids_count")

	// Operation attributes
	AttrOperation = attribute.Key("operation")
	AttrFound     = attribute.Key("found")

	// Error attributes
	AttrErrorOperation = attribute.Key("error.operation")
	AttrErrorSeverity  = attribute.Key("error.severity")
	AttrErrorMessage   = attribute.Key("error.message")

	// Duration and status
	AttrDurationMs    = attribute.Key("duration_ms")
	AttrSDKDurationMs = attribute.Key("sdk_duration_ms")
	AttrSuccess       = attribute.Key("success")
)