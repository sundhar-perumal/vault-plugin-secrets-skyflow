# Skyflow Vault Plugin - Testing Guide

Testing documentation for the Skyflow Vault Plugin.

## Test Overview

The project includes comprehensive unit tests covering:

| Component | Test File |
|-----------|-----------|
| Backend initialization | `backend_test.go` |
| Configuration | `config_test.go` |
| Role management | `role_test.go` |
| Circuit breaker | `circuitbreaker_test.go` |
| Metrics | `metrics_test.go` |
| Telemetry | `telemetry/*_test.go` |

## Running Tests

### Basic Commands

```bash
# Run all tests
go test ./...

# Run with verbose output
go test ./... -v

# Run specific package
go test ./backend/...

# Run specific test
go test -v -run TestBackend_Factory ./backend/...
```

### Coverage Reports

```bash
# Run tests with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# Display coverage by function
go tool cover -func=coverage.out
```

### Race Detection

```bash
# Run with race detector
go test ./... -race
```

## Test Structure

### Test File Organization

```
backend/
├── backend.go          → backend_test.go
├── config.go           → config_test.go
├── role.go             → role_test.go
├── circuitbreaker.go   → circuitbreaker_test.go
├── metrics.go          → metrics_test.go
└── telemetry/
    ├── config.go       → config_test.go
    ├── emitter.go      → emitter_test.go
    ├── noop.go         → noop_test.go
    └── telemetry.go    → telemetry_test.go
```

### Test Naming Convention

```go
func TestComponent_Feature(t *testing.T)
func TestComponent_Feature_EdgeCase(t *testing.T)
```

## Integration Testing

### Prerequisites

1. Running Vault server
2. Valid Skyflow credentials
3. Plugin registered and enabled

### Manual Integration Test

```bash
#!/bin/bash

export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root'

# Configure backend
vault write skyflow/config \
    credentials_file_path="/path/to/credentials.json"

# Create role
vault write skyflow/roles/test-role \
    description="Integration test role" \
    vault_id="test-vault"

# Generate token
vault read skyflow/creds/test-role

# Check health
vault read skyflow/health

# Clean up
vault delete skyflow/roles/test-role
vault delete skyflow/config
```

## Coverage Goals

| Target | Coverage |
|--------|----------|
| Line coverage | >80% |
| Branch coverage | >70% |
| Critical paths | 100% |

## Troubleshooting

### Common Issues

**Tests timeout:**
```bash
go test ./... -timeout 30m
```

**Race conditions:**
```bash
go test ./... -race
```

**Flaky tests:**
```bash
go test ./... -count=10
```

**Memory issues:**
```bash
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

