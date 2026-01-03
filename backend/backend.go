package backend

import (
	"context"
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

	// Telemetry emitter for traces and metrics
	emitter *telemetry.Emitter
}

// Factory returns a new backend as logical.Backend
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	// Initialize telemetry emitter (respects ENV=dev for local development)
	emitter, _ := telemetry.NewEmitter(ctx, telemetry.EmitterConfig{
		ServiceName:    "skyflow-vault-plugin",
		ServiceVersion: Version,
		Environment:    "production",
	})

	b := &skyflowBackend{
		emitter: emitter,
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

// invalidate is called when a key is updated
func (b *skyflowBackend) invalidate(ctx context.Context, key string) {
	b.Logger().Debug("key invalidated", "key", key)
}

// cleanup is called during backend cleanup
func (b *skyflowBackend) cleanup(ctx context.Context) {
	if b.emitter != nil {
		b.emitter.Close(ctx)
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

	if event.ClientIP != "" {
		fields = append(fields, "client_ip", event.ClientIP)
	}

	if event.Error != "" {
		fields = append(fields, "error", event.Error)
	}

	b.Logger().Info("audit", fields...)
}
