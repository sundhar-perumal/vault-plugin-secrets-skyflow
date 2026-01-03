// Package integration provides integration tests for the Skyflow Vault Plugin.
//
// These tests require valid Skyflow credentials and make actual API calls.
//
// Run modes:
//   - DEV mode (strict):  TEST_MODE=dev go test ./test/integration/...
//   - CI mode (lenient):  go test ./test/integration/...
//
// In DEV mode, tests FAIL if token generation fails.
// In CI mode (default), tests PASS if errors are handled gracefully.
package integration

import (
	"context"
	"os"
	"testing"

	"github.com/sundhar-perumal/vault-plugin-secrets-skyflow/backend"
	"github.com/hashicorp/vault/sdk/logical"
)

// TestMain runs before all tests - global setup and teardown
func TestMain(m *testing.M) {
	// Global setup
	setup()

	// Run all tests
	code := m.Run()

	// Global teardown
	teardown()

	os.Exit(code)
}

func setup() {
	// Log the test mode
	if isStrictMode() {
		println("=== Running in DEV mode (strict) ===")
	} else {
		println("=== Running in CI mode (lenient) ===")
	}
}

func teardown() {
	// Any global cleanup can go here
}

// ============================================================================
// Test Mode Configuration
// ============================================================================

// testMode returns the current test mode ("dev" or "ci")
func testMode() string {
	mode := os.Getenv("TEST_MODE")
	if mode == "dev" {
		return "dev"
	}
	return "ci" // default: lenient
}

// isStrictMode returns true if running in dev (strict) mode
// In strict mode, tests fail if token generation fails
func isStrictMode() bool {
	return testMode() == "dev"
}

// testBackend holds the test backend instance
type testBackend struct {
	backend logical.Backend
	storage logical.Storage
}

// newTestBackend creates a new test backend for integration tests
func newTestBackend(t *testing.T) *testBackend {
	t.Helper()

	ctx := context.Background()
	storage := &logical.InmemStorage{}

	config := &logical.BackendConfig{
		Logger:      nil,
		System:      &logical.StaticSystemView{},
		StorageView: storage,
	}

	b, err := backend.Factory(ctx, config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	return &testBackend{
		backend: b,
		storage: storage,
	}
}

// writeConfig writes configuration to the backend
func (tb *testBackend) writeConfig(t *testing.T, data map[string]interface{}) *logical.Response {
	t.Helper()

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "config",
		Storage:   tb.storage,
		Data:      data,
		Connection: &logical.Connection{
			RemoteAddr: "127.0.0.1",
		},
	}

	resp, err := tb.backend.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	return resp
}

// writeRole writes a role to the backend
func (tb *testBackend) writeRole(t *testing.T, name string, data map[string]interface{}) *logical.Response {
	t.Helper()

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "roles/" + name,
		Storage:   tb.storage,
		Data:      data,
		Connection: &logical.Connection{
			RemoteAddr: "127.0.0.1",
		},
	}

	resp, err := tb.backend.HandleRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to write role: %v", err)
	}

	return resp
}

// readCreds reads credentials for a role
func (tb *testBackend) readCreds(t *testing.T, name string) (*logical.Response, error) {
	t.Helper()

	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "creds/" + name,
		Storage:   tb.storage,
		Connection: &logical.Connection{
			RemoteAddr: "127.0.0.1",
		},
	}

	return tb.backend.HandleRequest(context.Background(), req)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// readCredentialsFile reads a JSON credentials file and returns its content
func readCredentialsFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read credentials file: %v", err)
	}

	return string(data)
}

