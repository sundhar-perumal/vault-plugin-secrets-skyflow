package telemetry

import "context"

// NoOp is a no-operation telemetry implementation.
// Use this when telemetry is not needed or as a default.
type NoOp struct{}

// NewNoOp creates a new no-op telemetry instance
func NewNoOp() *NoOp {
	return &NoOp{}
}

// Ensure NoOp implements Telemetry interface
var _ Telemetry = (*NoOp)(nil)

func (n *NoOp) OnTokenRequest(ctx context.Context, event TokenRequestEvent) context.Context {
	return ctx
}
func (n *NoOp) OnTokenGenerate(ctx context.Context, event TokenGenerateEvent) {}
func (n *NoOp) OnConfigWrite(ctx context.Context, event ConfigWriteEvent) context.Context {
	return ctx
}
func (n *NoOp) OnConfigRead(ctx context.Context, event ConfigReadEvent)             {}
func (n *NoOp) OnRoleWrite(ctx context.Context, event RoleWriteEvent) context.Context { return ctx }
func (n *NoOp) OnRoleRead(ctx context.Context, event RoleReadEvent)                 {}
func (n *NoOp) OnError(ctx context.Context, event ErrorEvent)                       {}
func (n *NoOp) EndSpan(ctx context.Context)                                         {}

