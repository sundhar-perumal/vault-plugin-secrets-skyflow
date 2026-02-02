package backend

import (
	"context"
	"os"
	"time"

	"github.com/sundhar-perumal/vault-plugin-secrets-skyflow/backend/telemetry"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	backendHelp = `
The Skyflow secrets engine generates bearer tokens for authenticating with Skyflow APIs.
After mounting this secrets engine, you can configure service account credentials and
define roles that specify token generation parameters.
`
)

// Version information - set via ldflags at build time
var (
	Version   = "v1.0.0"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// skyflowBackend implements logical.Backend
type skyflowBackend struct {
	*framework.Backend

	// Telemetry providers
	telemetryProviders *telemetry.Providers
	telemetryShutdown  func(context.Context) error
}

// Factory returns a new backend as logical.Backend
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	// Get environment from ENV variable, default to "unknown"
	environment := os.Getenv("ENV")
	if environment == "" {
		environment = "unknown"
	}

	b := &skyflowBackend{}

	// Initialize telemetry (respects RUNTIME_LOCAL and ENV for local development)
	// If disabled or fails, OTEL uses built-in noop tracer automatically
	providers, shutdown, err := telemetry.Init(ctx, telemetry.BuildConfigInput{
		ServiceName:    "skyflow-vault-plugin",
		ServiceVersion: Version,
		Environment:    environment,
	})
	if err != nil {
		// Log warning but don't fail - telemetry is optional
		// OTEL will use built-in noop tracer
		if conf.Logger != nil {
			conf.Logger.Warn("telemetry initialization failed, continuing without telemetry", "error", err)
		}
	} else {
		b.telemetryProviders = providers
		b.telemetryShutdown = shutdown

		// Log telemetry status via Vault logger (appears in Vault logs)
		if conf.Logger != nil {
			tracesEnabled := providers != nil && providers.Traces() != nil && providers.Traces().IsEnabled()
			metricsEnabled := providers != nil && providers.Metrics() != nil
			if tracesEnabled || metricsEnabled {
				conf.Logger.Info("telemetry initialized",
					"environment", environment,
					"traces_enabled", tracesEnabled,
					"metrics_enabled", metricsEnabled,
					"ENV", os.Getenv("ENV"),
					"TELEMETRY_ENABLED", os.Getenv("TELEMETRY_ENABLED"),
					"RUNTIME_LOCAL", os.Getenv("RUNTIME_LOCAL"),
				)
			} else {
				conf.Logger.Info("telemetry disabled or noop",
					"environment", environment,
					"providers_nil", providers == nil,
				)
			}
		}
	}

	b.Backend = &framework.Backend{
		Help:           backendHelp,
		BackendType:    logical.TypeLogical,
		RunningVersion: Version,

		Paths: framework.PathAppend(
			pathConfig(b),
			pathRoles(b),
			pathToken(b),
			pathHealth(b),
		),

		PathsSpecial: &logical.Paths{
			SealWrapStorage: []string{
				"config",
				"role/*",
			},
		},

		Secrets:    []*framework.Secret{},
		Invalidate: b.invalidate,
		Clean:      b.cleanup,
	}

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

// metrics returns the metrics provider (nil-safe)
func (b *skyflowBackend) metrics() *telemetry.MetricsProvider {
	if b.telemetryProviders == nil {
		return nil
	}
	return b.telemetryProviders.Metrics()
}

// traces returns the traces provider (nil-safe)
func (b *skyflowBackend) traces() *telemetry.TracesProvider {
	if b.telemetryProviders == nil {
		return nil
	}
	return b.telemetryProviders.Traces()
}

// invalidate is called when a key is updated
func (b *skyflowBackend) invalidate(ctx context.Context, key string) {
	b.Logger().Debug("key invalidated", "key", key)
}

// cleanup is called during backend cleanup
func (b *skyflowBackend) cleanup(ctx context.Context) {
	if b.telemetryShutdown != nil {
		if err := b.telemetryShutdown(ctx); err != nil {
			b.Logger().Warn("telemetry shutdown error", "error", err)
		}
	}
	b.Logger().Info("backend cleanup complete")
}

// auditEvent represents an audit log entry
type auditEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Operation string    `json:"operation"`
	Role      string    `json:"role"`
	Success   bool      `json:"success"`
	Duration  int64     `json:"duration_ms"`
	ClientIP  string    `json:"client_ip,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// auditLog writes audit events
func (b *skyflowBackend) auditLog(event auditEvent) {
	fields := []interface{}{
		"timestamp", event.Timestamp.Format(time.RFC3339),
		"operation", event.Operation,
		"role", event.Role,
		"success", event.Success,
		"duration_ms", event.Duration,
	}

	if event.TraceID != "" {
		fields = append(fields, "trace_id", event.TraceID)
	}

	if event.ClientIP != "" {
		fields = append(fields, "client_ip", event.ClientIP)
	}

	if event.Error != "" {
		fields = append(fields, "error", event.Error)
	}

	b.Logger().Info("audit", fields...)
}